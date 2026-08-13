package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// ChatCompletions handles OpenAI Chat Completions API requests.
// POST /v1/chat/completions
func (h *OpenAIGatewayHandler) ChatCompletions(c *gin.Context) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.chat_completions",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	reqStream, ok := parseOpenAICompatibleStream(body)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", invalidStreamFieldTypeMessage)
		return
	}

	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	setOpsRequestContext(c, reqModel, reqStream)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(reqStream, false)))

	if decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIChat, reqModel, body); decision != nil && decision.Blocked {
		h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
		return
	}
	if h.rejectIfCyberSessionBlocked(c, apiKey, body, reqModel, cyberBlockFormatChat) {
		return
	}

	// 解析渠道级模型映射
	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, reqModel)
	routeCtx := c.Request.Context()
	routingModel, routingAccountIDs, routed, routeErr := h.gatewayService.ResolveQuotaAllowedGroupRoute(routeCtx, apiKey.Group, reqModel, subject.UserID, nil)
	if routeErr != nil {
		status, code, message, retryAfter := billingErrorDetails(routeErr)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}
	// 模型路由命中：候选账号的模型支持由账号自身 model_mapping 决定，而非客户端请求名。
	// 将候选账号 ID 集合写入 ctx，供 OpenAI 账号选择链路按账号模型判断候选。
	if routed && len(routingAccountIDs) > 0 {
		routeIDs := make(map[int64]struct{}, len(routingAccountIDs))
		for _, id := range routingAccountIDs {
			routeIDs[id] = struct{}{}
		}
		routeCtx = service.WithRouteAccountIDs(routeCtx, routeIDs)
	}

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai_chat_completions.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	sessionHash := h.gatewayService.GenerateSessionHash(c, body)
	promptCacheKey := h.gatewayService.ExtractSessionID(c, body)

	failedAccountIDs := make(map[int64]struct{})
	var lastFailoverErr *service.UpstreamFailoverError
	var candidateFailures []service.ModelCandidateFailure

	for {
		// If all routing accounts have been tried and failed upstream, re-resolve
		// the route to advance to the next priority candidate.
		if len(routingAccountIDs) > 0 {
			allExcluded := true
			for _, id := range routingAccountIDs {
				if _, ok := failedAccountIDs[id]; !ok {
					allExcluded = false
					break
				}
			}
			if allExcluded {
				newModel, newIDs, _, reRouteErr := h.gatewayService.ResolveQuotaAllowedGroupRoute(
					routeCtx, apiKey.Group, reqModel, subject.UserID, failedAccountIDs)
				if reRouteErr != nil {
					if len(candidateFailures) > 0 {
						status, message := modelCandidatesExhaustedDetails(candidateFailures)
						h.handleStreamingAwareError(c, status, "all_model_candidates_failed", message, streamStarted)
					} else if lastFailoverErr != nil {
						h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
					} else {
						h.handleStreamingAwareError(c, http.StatusBadGateway, "api_error", "Upstream request failed", streamStarted)
					}
					return
				}
				routingModel = newModel
				routingAccountIDs = newIDs
				if len(newIDs) > 0 {
					routeIDs := make(map[int64]struct{}, len(newIDs))
					for _, id := range newIDs {
						routeIDs[id] = struct{}{}
					}
					routeCtx = service.WithRouteAccountIDs(routeCtx, routeIDs)
				}
			}
		}

		reqLog.Debug("openai_chat_completions.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			routeCtx,
			apiKey.GroupID,
			"",
			sessionHash,
			routingModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
			service.OpenAIEndpointCapabilityChatCompletions,
			false,
		)
		if err != nil {
			if failures := service.ModelCandidateFailuresFromError(err); routed && len(failures) > 0 {
				for _, failure := range failures {
					candidateFailures = appendModelCandidateFailure(candidateFailures, failure)
				}
				for _, id := range routingAccountIDs {
					failedAccountIDs[id] = struct{}{}
				}
				continue
			}
			reqLog.Warn("openai_chat_completions.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if routed && len(routingAccountIDs) > 0 {
				for _, id := range routingAccountIDs {
					failedAccountIDs[id] = struct{}{}
				}
				continue
			}
			markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", noAvailableAccountsMessage(reqModel, routed, apiKey.Group), streamStarted)
			return
		}
		if selection == nil || selection.Account == nil {
			if routed && len(routingAccountIDs) > 0 {
				for _, id := range routingAccountIDs {
					failedAccountIDs[id] = struct{}{}
				}
				continue
			}
			markOpsRoutingCapacityLimited(c)
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", noAvailableAccountsMessage(reqModel, routed, apiKey.Group), streamStarted)
			return
		}
		account := selection.Account
		if len(routingAccountIDs) > 0 && !containsAccountID(routingAccountIDs, account.ID) {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			failedAccountIDs[account.ID] = struct{}{}
			continue
		}
		if routed {
			// 模型路由：上游模型名固定取账号 model_mapping 第一个条目的 value，
			// 与账号绑定模型路由（MVP-005 语义）一致，不再读取候选 model 字段。
			if upstream := account.FirstModelMappingValue(); upstream != "" {
				routingModel = upstream
			}
			quotaErr := service.CheckDynamicTokenRouteCandidate(
				routeCtx, apiKey.GroupID,
				service.DynamicTokenRequestIdentity{UserID: subject.UserID, APIKeyID: apiKey.ID},
				reqModel, account, routingModel,
			)
			if errors.Is(quotaErr, service.ErrDynamicTokenQuotaExceeded) {
				if selection.ReleaseFunc != nil {
					selection.ReleaseFunc()
				}
				failedAccountIDs[account.ID] = struct{}{}
				candidateFailures = appendModelCandidateFailure(candidateFailures, service.ModelCandidateFailure{
					AccountID: account.ID, AccountName: account.Name, Model: routingModel,
					Reason: "token_quota", Message: "Token quota exceeded",
				})
				continue
			}
		}
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai_chat_completions.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		_ = scheduleDecision
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()

		forwardBody := body
		if routingModel != reqModel {
			forwardBody = h.gatewayService.ReplaceModelInBody(forwardBody, routingModel)
		}
		if channelMapping.Mapped {
			forwardBody = h.gatewayService.ReplaceModelInBody(forwardBody, channelMapping.MappedModel)
		}
		writerSizeBeforeForward := c.Writer.Size()
		forwardModel := reqModel
		if routingModel != reqModel {
			forwardModel = routingModel
		}
		if channelMapping.Mapped {
			forwardModel = channelMapping.MappedModel
		}
		logModelForwardStarted(reqLog, "openai_chat_completions.forward_started", account, reqModel, forwardModel, reqStream, len(failedAccountIDs), len(forwardBody))
		result, err := func() (*service.OpenAIForwardResult, error) {
			defer func() {
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
			}()
			return h.gatewayService.ForwardAsChatCompletions(c.Request.Context(), c, account, forwardBody, promptCacheKey, "")
		}()
		cyberBlockKeyChat := ""
		if service.GetOpsCyberPolicy(c) != nil {
			cyberBlockKeyChat = service.CyberSessionBlockKey(apiKey.ID, c, body)
		}
		h.recordCyberPolicyIfMarked(c, apiKey, account, subscription, reqModel, err != nil, cyberBlockKeyChat, channelMapping.ToUsageFields(reqModel, ""), service.HashUsageRequestPayload(body))

		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		forwardFinishedFields := []zap.Field{}
		if result != nil {
			forwardFinishedFields = append(forwardFinishedFields,
				zap.String("upstream_request_id", result.RequestID),
				zap.String("response_id", result.ResponseID),
				zap.String("result_model", result.Model),
				zap.String("result_upstream_model", result.UpstreamModel),
				zap.String("billing_model", result.BillingModel),
				zap.Bool("client_disconnect", result.ClientDisconnect),
				zap.Int("image_count", result.ImageCount),
			)
			if result.FirstTokenMs != nil {
				forwardFinishedFields = append(forwardFinishedFields, zap.Int("first_token_ms", *result.FirstTokenMs))
			}
		}
		logModelForwardFinished(reqLog, "openai_chat_completions.forward_finished", c, forwardStart, account, reqModel, forwardModel, reqStream, len(failedAccountIDs), writerSizeBeforeForward, err, forwardFinishedFields...)
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMs(c, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			if result != nil && result.ImageCount > 0 {
				reqLog.Warn("openai_chat_completions.forward_partial_error_with_image_result",
					zap.Int64("account_id", account.ID),
					zap.Int("image_count", result.ImageCount),
					zap.Error(err),
				)
			} else {
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					if c.Writer.Size() != writerSizeBeforeForward {
						h.handleFailoverExhausted(c, failoverErr, true)
						return
					}
					h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
					h.gatewayService.RecordOpenAIAccountSwitch()
					failedAccountIDs[account.ID] = struct{}{}
					lastFailoverErr = failoverErr
					candidateFailures = appendModelCandidateFailure(candidateFailures, upstreamCandidateFailure(account, forwardModel, failoverErr))
					reqLog.Warn("openai_chat_completions.upstream_failover_switching",
						zap.Int64("account_id", account.ID),
						zap.Int("upstream_status", failoverErr.StatusCode),
					)
					continue
				}
				if c.Request.Context().Err() == nil && c.Writer.Size() == writerSizeBeforeForward {
					h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
					h.gatewayService.RecordOpenAIAccountSwitch()
					failedAccountIDs[account.ID] = struct{}{}
					lastFailoverErr = &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway}
					candidateFailures = appendModelCandidateFailure(candidateFailures, service.ModelCandidateFailure{
						AccountID: account.ID, AccountName: account.Name, Model: forwardModel,
						Reason: "upstream_error", Message: err.Error(),
					})
					continue
				}
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				upstreamErrorAlreadyCommunicated := openAIForwardErrorAlreadyCommunicated(c, writerSizeBeforeForward, err)
				wroteFallback := false
				if !upstreamErrorAlreadyCommunicated {
					wroteFallback = h.ensureForwardErrorResponse(c, streamStarted)
				}
				reqLog.Warn("openai_chat_completions.forward_failed",
					zap.Int64("account_id", account.ID),
					zap.Bool("fallback_error_response_written", wroteFallback),
					zap.Bool("upstream_error_response_already_written", upstreamErrorAlreadyCommunicated),
					zap.Error(err),
				)
				return
			}
		}
		if result != nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}

		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)
		inboundEndpoint := GetInboundEndpoint(c)
		upstreamEndpoint := resolveOpenAIUpstreamEndpoint(c, account)

		cyberBlocked := service.GetOpsCyberPolicy(c) != nil
		quotaRouteAlias := ""
		if routed {
			quotaRouteAlias = reqModel
		}
		h.submitOpenAIUsageRecordTask(c.Request.Context(), result, func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             apiKey,
				User:               apiKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    inboundEndpoint,
				UpstreamEndpoint:   upstreamEndpoint,
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				APIKeyService:      h.apiKeyService,
				ChannelUsageFields: channelMapping.ToUsageFields(reqModel, result.UpstreamModel),
				RouteAlias:         quotaRouteAlias,
				CyberBlocked:       cyberBlocked,
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.chat_completions"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Any("group_id", apiKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("openai_chat_completions.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai_chat_completions.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", len(failedAccountIDs)),
		)
		return
	}
}

// resolveOpenAIUpstreamEndpoint returns the actual upstream endpoint for an
// OpenAI account, used by every OpenAI usage-recording site. APIKey accounts
// whose upstream is forced or probed to not support the Responses API are
// served directly via /v1/chat/completions (the raw chat path) regardless of
// the inbound endpoint; everything else goes through the Responses API.
func resolveOpenAIUpstreamEndpoint(c *gin.Context, account *service.Account) string {
	if account != nil && account.Type == service.AccountTypeAPIKey &&
		!openai_compat.ShouldUseResponsesAPI(account.Extra) {
		return "/v1/chat/completions"
	}
	return GetUpstreamEndpoint(c, account.Platform)
}

// noAvailableAccountsMessage 构造“无可用账号”时的可读错误消息，帮助调用方定位原因：
// 模型名是否被路由/白名单支持、分组下是否有可调度账号。
func noAvailableAccountsMessage(reqModel string, routed bool, group *service.Group) string {
	groupName := ""
	if group != nil {
		groupName = strings.TrimSpace(group.Name)
	}
	switch {
	case routed && groupName != "":
		return fmt.Sprintf("模型 %q 命中了分组 %q 的路由规则，但没有可用的路由账号（账号可能被禁用、限额已耗尽或不可调度），请检查分组账号配置", reqModel, groupName)
	case routed:
		return fmt.Sprintf("模型 %q 命中了路由规则，但没有可用的路由账号（账号可能被禁用、限额已耗尽或不可调度），请检查分组账号配置", reqModel)
	case groupName != "":
		return fmt.Sprintf("模型 %q 在分组 %q 中没有可用账号：请确认模型名是否已配置（路由别名或账号 model_mapping 白名单），以及账号是否可用", reqModel, groupName)
	default:
		return fmt.Sprintf("模型 %q 没有可用账号：请确认模型名是否正确、已配置路由或账号 model_mapping 白名单，以及账号是否可用", reqModel)
	}
}
