package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrInvalidModelRouting identifies malformed legacy or candidate routing JSON.
var ErrInvalidModelRouting = errors.New("invalid model routing")

// ModelRouteCandidate is one upstream model choice for a requested route alias.
// Legacy entries keep Model empty so the caller can pass the requested model
// through unchanged. Candidate token limits are stored separately.
type ModelRouteCandidate struct {
	Model      string  `json:"model"`
	AccountIDs []int64 `json:"account_ids"`
	Priority   int     `json:"priority"`
	Legacy     bool    `json:"-"`
}

// ModelRoutingConfig maps an exact model or a trailing-star prefix pattern to
// its ordered candidates.
type ModelRoutingConfig map[string][]ModelRouteCandidate

// ModelRoutingJSON is an opaque JSON object used by persistence layers. It
// preserves whether each route uses the legacy or candidate representation.
type ModelRoutingJSON struct {
	raw json.RawMessage
}

func NewModelRoutingJSON(data []byte) ModelRoutingJSON {
	return ModelRoutingJSON{raw: append(json.RawMessage(nil), data...)}
}

func (v ModelRoutingJSON) MarshalJSON() ([]byte, error) {
	if len(bytes.TrimSpace(v.raw)) == 0 {
		return []byte("{}"), nil
	}
	return append([]byte(nil), v.raw...), nil
}

func (v *ModelRoutingJSON) UnmarshalJSON(data []byte) error {
	if v == nil {
		return errors.New("model routing JSON receiver is nil")
	}
	v.raw = append(v.raw[:0], data...)
	return nil
}

func (v ModelRoutingJSON) RawMessage() json.RawMessage {
	return append(json.RawMessage(nil), v.raw...)
}

// ParseModelRoutingConfig accepts both the legacy map[string][]int64 shape and
// the candidate-object shape. Candidate order is stable for equal priorities.
func ParseModelRoutingConfig(data []byte) (ModelRoutingConfig, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("%w: empty JSON", ErrInvalidModelRouting)
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return ModelRoutingConfig{}, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidModelRouting, err)
	}

	config := make(ModelRoutingConfig, len(raw))
	for pattern, value := range raw {
		if err := validateRoutePattern(pattern); err != nil {
			return nil, err
		}
		candidates, err := parseRouteCandidates(pattern, value)
		if err != nil {
			return nil, err
		}
		config[pattern] = candidates
	}
	return config, nil
}

// Match returns a copy of the candidates for an exact route first, then for a
// deterministic trailing-star prefix match. Legacy candidates inherit the
// requested model.
func (c ModelRoutingConfig) Match(requestedModel string) []ModelRouteCandidate {
	if requestedModel == "" {
		return nil
	}
	if candidates, ok := c[requestedModel]; ok {
		return materializeLegacyModel(candidates, requestedModel)
	}

	patterns := make([]string, 0, len(c))
	for pattern := range c {
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(requestedModel, strings.TrimSuffix(pattern, "*")) {
			patterns = append(patterns, pattern)
		}
	}
	sort.Slice(patterns, func(i, j int) bool {
		if len(patterns[i]) != len(patterns[j]) {
			return len(patterns[i]) > len(patterns[j])
		}
		return patterns[i] < patterns[j]
	})
	if len(patterns) == 0 {
		return nil
	}
	return materializeLegacyModel(c[patterns[0]], requestedModel)
}

func parseRouteCandidates(pattern string, value json.RawMessage) ([]ModelRouteCandidate, error) {
	if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return nil, fmt.Errorf("%w: route %q is null", ErrInvalidModelRouting, pattern)
	}

	var legacyIDs []int64
	if err := json.Unmarshal(value, &legacyIDs); err == nil {
		if err := validateAccountIDs(pattern, 0, legacyIDs); err != nil {
			return nil, err
		}
		return []ModelRouteCandidate{{AccountIDs: append([]int64(nil), legacyIDs...), Legacy: true}}, nil
	}

	var candidates []ModelRouteCandidate
	if err := json.Unmarshal(value, &candidates); err != nil {
		return nil, fmt.Errorf("%w: route %q must be an account ID array or candidate array", ErrInvalidModelRouting, pattern)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: route %q has no candidates", ErrInvalidModelRouting, pattern)
	}
	for i := range candidates {
		candidate := &candidates[i]
		candidate.Model = strings.TrimSpace(candidate.Model)
		if candidate.Model == "" {
			return nil, fmt.Errorf("%w: route %q candidate %d has no model", ErrInvalidModelRouting, pattern, i)
		}
		if candidate.Priority < 0 {
			return nil, fmt.Errorf("%w: route %q candidate %d has negative priority", ErrInvalidModelRouting, pattern, i)
		}
		if err := validateAccountIDs(pattern, i, candidate.AccountIDs); err != nil {
			return nil, err
		}
		candidate.AccountIDs = append([]int64(nil), candidate.AccountIDs...)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})
	return candidates, nil
}

func validateRoutePattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("%w: route name is empty", ErrInvalidModelRouting)
	}
	if strings.Count(pattern, "*") > 1 || (strings.Contains(pattern, "*") && !strings.HasSuffix(pattern, "*")) {
		return fmt.Errorf("%w: route %q only supports a trailing wildcard", ErrInvalidModelRouting, pattern)
	}
	return nil
}

func validateAccountIDs(pattern string, candidateIndex int, accountIDs []int64) error {
	if len(accountIDs) == 0 {
		return fmt.Errorf("%w: route %q candidate %d has no accounts", ErrInvalidModelRouting, pattern, candidateIndex)
	}
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			return fmt.Errorf("%w: route %q candidate %d has invalid account ID %d", ErrInvalidModelRouting, pattern, candidateIndex, accountID)
		}
	}
	return nil
}

func materializeLegacyModel(candidates []ModelRouteCandidate, requestedModel string) []ModelRouteCandidate {
	result := make([]ModelRouteCandidate, len(candidates))
	copy(result, candidates)
	for i := range result {
		result[i].AccountIDs = append([]int64(nil), result[i].AccountIDs...)
		if result[i].Legacy {
			result[i].Model = requestedModel
		}
	}
	return result
}
