package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// validateAccountModelMapping enforces the write-time invariant that a
// non-OAuth account can expose at most one upstream model name.
func validateAccountModelMapping(accountType string, credentials map[string]any) error {
	if accountType == AccountTypeOAuth || credentials == nil {
		return nil
	}

	raw, exists := credentials["model_mapping"]
	if !exists || raw == nil {
		return nil
	}

	countKeys := func(keys []string) error {
		count := 0
		for _, key := range keys {
			if strings.TrimSpace(key) == "" {
				continue
			}
			count++
			if count > 1 {
				return infraerrors.BadRequest(
					"ACCOUNT_MULTIPLE_UPSTREAM_MODELS",
					"model account can contain at most one upstream model",
				)
			}
		}
		return nil
	}

	switch mapping := raw.(type) {
	case map[string]any:
		keys := make([]string, 0, len(mapping))
		for key := range mapping {
			keys = append(keys, key)
		}
		return countKeys(keys)
	case map[string]string:
		keys := make([]string, 0, len(mapping))
		for key := range mapping {
			keys = append(keys, key)
		}
		return countKeys(keys)
	default:
		return infraerrors.BadRequest(
			"ACCOUNT_INVALID_MODEL_MAPPING",
			"credentials.model_mapping must be an object",
		)
	}
}
