package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

const gatewayRequestBodyCaptureLimit = 64 * 1024

type gatewayRequestBodyCapture struct {
	limit int
	body  bytes.Buffer
	total int64
}

func (c *gatewayRequestBodyCapture) Write(p []byte) (int, error) {
	n := len(p)
	c.total += int64(n)
	if c.limit > 0 && c.body.Len() < c.limit {
		remaining := c.limit - c.body.Len()
		if len(p) > remaining {
			_, _ = c.body.Write(p[:remaining])
		} else {
			_, _ = c.body.Write(p)
		}
	}
	return n, nil
}

func (c *gatewayRequestBodyCapture) snapshot() ([]byte, int64, bool) {
	if c == nil {
		return nil, 0, false
	}
	body := append([]byte(nil), c.body.Bytes()...)
	return body, c.total, c.total > int64(c.limit)
}

type teeReadCloser struct {
	io.Reader
	io.Closer
}

func wrapGatewayRequestBody(c *gin.Context) *gatewayRequestBodyCapture {
	if c == nil || c.Request == nil || c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil
	}
	capture := &gatewayRequestBodyCapture{limit: gatewayRequestBodyCaptureLimit}
	original := c.Request.Body
	primeLimit := int64(gatewayRequestBodyCaptureLimit + 1)
	if c.Request.ContentLength >= 0 && c.Request.ContentLength < primeLimit {
		primeLimit = c.Request.ContentLength
	}
	if primeLimit <= 0 {
		return capture
	}
	prefix, _ := io.ReadAll(io.LimitReader(original, primeLimit))
	if len(prefix) > 0 {
		_, _ = capture.Write(prefix)
	}
	c.Request.Body = &teeReadCloser{
		Reader: io.MultiReader(bytes.NewReader(prefix), io.TeeReader(original, capture)),
		Closer: original,
	}
	return capture
}

var writeGatewayFailureRequest = logger.WriteRequestFailure

func safeLogGatewayFailureRequest(c *gin.Context, capture *gatewayRequestBodyCapture) {
	defer func() {
		_ = recover()
	}()
	logGatewayFailureRequest(c, capture)
}

func logGatewayFailureRequest(c *gin.Context, capture *gatewayRequestBodyCapture) {
	if c == nil || c.Request == nil {
		return
	}
	status := c.Writer.Status()
	if status < http.StatusBadRequest || status == http.StatusTooManyRequests {
		return
	}

	body, readBytes, truncated := capture.snapshot()
	if c.Request.ContentLength > readBytes {
		readBytes = c.Request.ContentLength
	}
	contentType := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Type")))
	contentEncoding := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Encoding")))

	formattedBody, bodyFormat, omittedReason := formatGatewayFailureBody(body, contentType, contentEncoding, truncated)
	requestID := c.Writer.Header().Get("X-Request-Id")
	if requestID == "" {
		requestID = c.Writer.Header().Get("x-request-id")
	}

	var userID, groupID int64
	if apiKey := getOpsAPIKey(c); apiKey != nil {
		if apiKey.User != nil {
			userID = apiKey.User.ID
		}
		if apiKey.GroupID != nil {
			groupID = *apiKey.GroupID
		}
	}

	endpoint := GetInboundEndpoint(c)
	if strings.TrimSpace(endpoint) == "" && c.Request.URL != nil {
		endpoint = c.Request.URL.Path
	}
	writeGatewayFailureRequest(logger.RequestFailureEvent{
		RequestID:       requestID,
		Endpoint:        endpoint,
		Method:          c.Request.Method,
		Status:          status,
		UserID:          userID,
		GroupID:         groupID,
		ContentType:     contentType,
		RequestBody:     formattedBody,
		RequestBodySize: readBytes,
		Truncated:       truncated,
		BodyFormat:      bodyFormat,
		BodyOmitted:     omittedReason,
	})
}

func formatGatewayFailureBody(body []byte, contentType, contentEncoding string, truncated bool) (string, string, string) {
	if len(body) == 0 {
		return "", "unavailable", "request_body_unavailable"
	}
	if contentEncoding != "" && contentEncoding != "identity" {
		return "", "unavailable", "encoded_request_body"
	}
	if isBinaryGatewayContentType(contentType) {
		return "", "unavailable", "binary_request_body"
	}
	if !utf8.Valid(body) {
		return "", "unavailable", "non_utf8_request_body"
	}

	text := string(body)
	if isJSONGatewayContentType(contentType) || looksLikeJSONBody(text) {
		var decoded any
		if err := json.Unmarshal(body, &decoded); err == nil {
			sanitized, marshalErr := json.Marshal(redactGatewayJSONValue(decoded))
			if marshalErr == nil {
				return string(sanitized), "json", ""
			}
		}
		format := "invalid_json_redacted"
		if truncated {
			format = "truncated_json_redacted_prefix"
		}
		return redactGatewayJSONFragments(text), format, ""
	}
	if strings.HasPrefix(contentType, "text/") || contentType == "" {
		return redactGatewayText(text), "text", ""
	}
	return "", "unavailable", "unsupported_content_type"
}

func isJSONGatewayContentType(contentType string) bool {
	return contentType == "application/json" ||
		strings.HasSuffix(contentType, "+json") ||
		strings.HasPrefix(contentType, "application/json;")
}

func isBinaryGatewayContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "multipart/") ||
		strings.HasPrefix(contentType, "application/octet-stream") ||
		strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "audio/") ||
		strings.HasPrefix(contentType, "video/")
}

func looksLikeJSONBody(body string) bool {
	body = strings.TrimSpace(body)
	return strings.HasPrefix(body, "{") || strings.HasPrefix(body, "[")
}

func redactGatewayJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if isSensitiveGatewayField(key) {
				out[key] = "***REDACTED***"
				continue
			}
			out[key] = redactGatewayJSONValue(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = redactGatewayJSONValue(child)
		}
		return out
	default:
		return value
	}
}

func isSensitiveGatewayField(field string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(field))
	switch normalized {
	case "authorization", "xapikey", "apikey", "token", "accesstoken", "refreshtoken",
		"password", "passwd", "secret", "clientsecret", "secretkey", "privatekey",
		"credential", "credentials", "cookie", "setcookie":
		return true
	}
	return strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "credential") ||
		strings.HasSuffix(normalized, "token")
}

// redactGatewayJSONFragments safely handles incomplete or invalid JSON while
// preserving the readable non-sensitive prefix for diagnostics.
func redactGatewayJSONFragments(input string) string {
	var out strings.Builder
	last := 0
	for i := 0; i < len(input); {
		if input[i] != '"' {
			i++
			continue
		}
		keyEnd, key, ok := scanJSONString(input, i)
		if !ok {
			break
		}
		cursor := skipJSONWhitespace(input, keyEnd)
		if cursor >= len(input) || input[cursor] != ':' {
			i = keyEnd
			continue
		}
		valueStart := skipJSONWhitespace(input, cursor+1)
		if !isSensitiveGatewayField(key) {
			i = valueStart
			continue
		}
		valueEnd := scanJSONValueEnd(input, valueStart)
		out.WriteString(input[last:valueStart])
		out.WriteString(`"***REDACTED***"`)
		last = valueEnd
		i = valueEnd
	}
	out.WriteString(input[last:])
	return out.String()
}

func scanJSONString(input string, start int) (int, string, bool) {
	if start >= len(input) || input[start] != '"' {
		return start, "", false
	}
	escaped := false
	for i := start + 1; i < len(input); i++ {
		switch {
		case escaped:
			escaped = false
		case input[i] == '\\':
			escaped = true
		case input[i] == '"':
			token := input[start : i+1]
			decoded, err := strconv.Unquote(token)
			if err != nil {
				return i + 1, token[1 : len(token)-1], true
			}
			return i + 1, decoded, true
		}
	}
	return len(input), input[start+1:], false
}

func skipJSONWhitespace(input string, start int) int {
	for start < len(input) {
		switch input[start] {
		case ' ', '\t', '\r', '\n':
			start++
		default:
			return start
		}
	}
	return start
}

func scanJSONValueEnd(input string, start int) int {
	if start >= len(input) {
		return start
	}
	switch input[start] {
	case '"':
		end, _, ok := scanJSONString(input, start)
		if !ok {
			return len(input)
		}
		return end
	case '{', '[':
		depth := 0
		inString := false
		escaped := false
		for i := start; i < len(input); i++ {
			ch := input[i]
			if inString {
				if escaped {
					escaped = false
					continue
				}
				if ch == '\\' {
					escaped = true
					continue
				}
				if ch == '"' {
					inString = false
				}
				continue
			}
			switch ch {
			case '"':
				inString = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
		return len(input)
	default:
		i := start
		for i < len(input) {
			switch input[i] {
			case ',', '}', ']', ' ', '\t', '\r', '\n':
				return i
			default:
				i++
			}
		}
		return i
	}
}

func redactGatewayText(input string) string {
	input = gatewayTextSecretPattern.ReplaceAllString(input, "${1}${2}***REDACTED***")
	return gatewayBearerPattern.ReplaceAllString(input, "${1}***REDACTED***")
}

var (
	gatewayTextSecretPattern = regexp.MustCompile(`(?i)((?:authorization|x-api-key|api[_-]?key|apikey|access[_-]?token|refresh[_-]?token|password|passwd|client[_-]?secret|secret[_-]?key|token)\s*[=:]\s*)(?:"[^"]*"|[^\s,;&]+)`)
	gatewayBearerPattern     = regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/\-=]+`)
)
