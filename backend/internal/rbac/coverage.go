package rbac

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type CoverageReport struct {
	Total, Controlled, Excluded int
	Missing                     []string
}

func ValidateRouteCoverage(actual gin.RoutesInfo, registry *Registry) CoverageReport {
	report := CoverageReport{Total: len(actual)}
	for _, route := range actual {
		declaration, ok := registry.Lookup(route.Method, route.Path)
		if !ok {
			report.Missing = append(report.Missing, route.Method+" "+route.Path)
			continue
		}
		if declaration.Classification == RouteControlled {
			report.Controlled++
		} else {
			report.Excluded++
		}
	}
	sort.Strings(report.Missing)
	return report
}

func (r CoverageReport) Err() error {
	if len(r.Missing) == 0 && r.Total == r.Controlled+r.Excluded {
		return nil
	}
	return fmt.Errorf("RBAC route coverage incomplete: total=%d controlled=%d excluded=%d missing=[%s]",
		r.Total, r.Controlled, r.Excluded, strings.Join(r.Missing, ", "))
}

// RegisterKnownExclusions classifies only the project's established non-user
// trust boundaries. A new route outside these prefixes remains missing and
// fails readiness/CI coverage.
func RegisterKnownExclusions(actual gin.RoutesInfo, registry *Registry) error {
	for _, route := range actual {
		if _, exists := registry.Lookup(route.Method, route.Path); exists {
			continue
		}
		reason := exclusionReason(route.Path)
		if reason == "" {
			continue
		}
		if err := registry.RegisterExcluded(route.Method, route.Path, reason); err != nil {
			return err
		}
	}
	return nil
}

func exclusionReason(path string) string {
	switch {
	case path == "/health" || path == "/api/v1/status" || path == "/api/v1/health" ||
		path == "/setup/status":
		return "public health/readiness endpoint"
	case path == "/api/event_logging/batch":
		return "best-effort client telemetry sink"
	case strings.HasPrefix(path, "/api/v1/auth/"):
		return "public or temporary-token authentication flow"
	case path == "/api/v1/settings/public" || path == "/api/v1/settings/email-unsubscribe":
		return "public settings or signed email action"
	case strings.HasPrefix(path, "/api/v1/pages/:slug/images/"):
		return "public page asset with handler visibility and path validation"
	case strings.HasPrefix(path, "/api/v1/payment/public/"):
		return "signed resume-token or persisted payment-state recovery"
	case strings.HasPrefix(path, "/api/v1/payment/webhook/"):
		return "payment provider signature trust boundary"
	case strings.HasPrefix(path, "/api/v1/integrations/"):
		return "external integration key trust boundary"
	case strings.HasPrefix(path, "/v1/"), strings.HasPrefix(path, "/v1beta/"),
		strings.HasPrefix(path, "/anthropic/"), strings.HasPrefix(path, "/antigravity/"),
		path == "/responses", strings.HasPrefix(path, "/responses/"),
		strings.HasPrefix(path, "/backend-api/codex/"),
		path == "/chat/completions", path == "/embeddings",
		strings.HasPrefix(path, "/images/"):
		return "model gateway API-key and subscription trust boundary"
	default:
		return ""
	}
}
