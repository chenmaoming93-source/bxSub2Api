package handler

import (
	"strings"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) SetSecurityCheckDependencies(checker *service.SecurityCheckService, provider *service.SecurityConfigProvider, collector *service.SecurityCheckCollector) {
	h.securityCheckService = checker
	h.securityConfigProvider = provider
	h.securityCheckCollector = collector
}

func (h *OpenAIGatewayHandler) SetSecurityCheckDependencies(checker *service.SecurityCheckService, provider *service.SecurityConfigProvider, collector *service.SecurityCheckCollector) {
	h.securityCheckService = checker
	h.securityConfigProvider = provider
	h.securityCheckCollector = collector
}

func (h *GatewayHandler) checkSecurityCheck(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte) *service.SecurityCheckResult {
	if h == nil {
		return nil
	}
	return runSecurityCheck(c, reqLog, h.securityCheckService, h.securityConfigProvider, h.securityCheckCollector, apiKey, subject, protocol, model, body)
}

func (h *OpenAIGatewayHandler) checkSecurityCheck(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte) *service.SecurityCheckResult {
	if h == nil {
		return nil
	}
	return runSecurityCheck(c, reqLog, h.securityCheckService, h.securityConfigProvider, h.securityCheckCollector, apiKey, subject, protocol, model, body)
}

func runSecurityCheck(c *gin.Context, reqLog *zap.Logger, checker *service.SecurityCheckService, provider *service.SecurityConfigProvider, collector *service.SecurityCheckCollector, apiKey *service.APIKey, subject middleware2.AuthSubject, protocol, model string, body []byte) *service.SecurityCheckResult {
	if c == nil || c.Request == nil || checker == nil || provider == nil || apiKey == nil || apiKey.GroupID == nil {
		return nil
	}
	groupID := *apiKey.GroupID
	config, err := provider.Get(c.Request.Context(), groupID)
	if err != nil {
		if reqLog != nil {
			reqLog.Warn("security_check.config_failed", zap.Int64("group_id", groupID), zap.Error(err))
		}
		return nil
	}
	result := checker.Check(c.Request.Context(), protocol, body, config)
	if collector != nil {
		metadata := service.SecurityCheckLogMetadata{
			EventID:   contentModerationRequestID(c.Request.Context()),
			RequestID: contentModerationRequestID(c.Request.Context()),
			UserID:    subject.UserID,
			APIKeyID:  apiKey.ID,
			GroupID:   groupID,
			Model:     strings.TrimSpace(model),
			Protocol:  protocol,
			Endpoint:  GetInboundEndpoint(c),
		}
		if apiKey.Name != "" {
			metadata.APIKeyName = apiKey.Name
		}
		if apiKey.Group != nil {
			metadata.GroupName = apiKey.Group.Name
			metadata.Provider = apiKey.Group.Platform
		}
		collector.Enqueue(config, result, metadata, body)
	}
	if reqLog != nil {
		fields := []zap.Field{
			zap.Int64("group_id", groupID),
			zap.Int64("user_id", subject.UserID),
			zap.String("protocol", protocol),
			zap.String("model", strings.TrimSpace(model)),
			zap.String("status", string(result.Status)),
			zap.String("decision", string(result.Decision)),
			zap.Bool("is_unsafe", result.IsUnsafe),
			zap.Int("body_bytes", len(body)),
		}
		if result.Err != nil {
			fields = append(fields, zap.Error(result.Err))
		}
		if result.Decision == "block" {
			reqLog.Warn("security_check.blocked", fields...)
		} else if result.Decision == "warn" {
			reqLog.Warn("security_check.warn", fields...)
		} else if result.Status == "timeout" || result.Status == "error" {
			reqLog.Warn("security_check.failed", fields...)
		} else {
			reqLog.Info("security_check.completed", fields...)
		}
	}
	return &result
}

func securityCheckBlocked(result *service.SecurityCheckResult) bool {
	return result != nil && result.Decision == "block"
}

const securityCheckErrorCode = "security_policy_violation"

const securityCheckErrorMessage = "request blocked by security policy"
