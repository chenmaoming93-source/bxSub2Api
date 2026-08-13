package domain

import (
	"reflect"
	"testing"
)

func TestModelAttributes_Normalize(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		t.Parallel()
		var m ModelAttributes
		if got := m.Normalize(); got != nil {
			t.Fatalf("Normalize() of nil map = %v, want nil", got)
		}
	})

	t.Run("drops blank-key entries", func(t *testing.T) {
		t.Parallel()
		m := ModelAttributes{
			"context_window": {Description: "上下文窗口总大小（token）", Value: 200000},
			"   ":            {Description: "should be dropped", Value: true},
			"":               {Description: "should be dropped too", Value: "x"},
		}
		got := m.Normalize()
		if len(got) != 1 {
			t.Fatalf("Normalize() len = %d, want 1 (got %#v)", len(got), got)
		}
		if _, ok := got["context_window"]; !ok {
			t.Fatalf("Normalize() lost valid key, got %#v", got)
		}
	})

	t.Run("trims key whitespace", func(t *testing.T) {
		t.Parallel()
		m := ModelAttributes{
			"  supports_vision  ": {Description: "支持图片输入", Value: true},
		}
		got := m.Normalize()
		if _, ok := got["supports_vision"]; !ok {
			t.Fatalf("Normalize() did not trim key, got %#v", got)
		}
		if _, ok := got["  supports_vision  "]; ok {
			t.Fatalf("Normalize() kept untrimmed key, got %#v", got)
		}
	})

	t.Run("preserves description and value as-is", func(t *testing.T) {
		t.Parallel()
		rawValue := "true" // 字符串 "true"，后端不得改写为布尔
		m := ModelAttributes{
			"supports_reasoning": {Description: "  是否支持推理模式  ", Value: rawValue},
		}
		got := m.Normalize()
		item, ok := got["supports_reasoning"]
		if !ok {
			t.Fatalf("Normalize() lost key, got %#v", got)
		}
		if item.Description != "  是否支持推理模式  " {
			t.Fatalf("Normalize() changed description to %q", item.Description)
		}
		if item.Value != rawValue {
			t.Fatalf("Normalize() changed value: got %#v (%T), want %#v (%T)", item.Value, item.Value, rawValue, rawValue)
		}
	})

	t.Run("preserves typed values without re-parsing", func(t *testing.T) {
		t.Parallel()
		m := ModelAttributes{
			"max_output_tokens": {Value: 64000},
			"modalities":        {Value: []string{"text", "image"}},
			"supports_vision":   {Value: true},
		}
		got := m.Normalize()
		if !reflect.DeepEqual(got["max_output_tokens"].Value, 64000) {
			t.Fatalf("int value changed: %#v", got["max_output_tokens"].Value)
		}
		if !reflect.DeepEqual(got["modalities"].Value, []string{"text", "image"}) {
			t.Fatalf("array value changed: %#v", got["modalities"].Value)
		}
		if !reflect.DeepEqual(got["supports_vision"].Value, true) {
			t.Fatalf("bool value changed: %#v", got["supports_vision"].Value)
		}
	})

	t.Run("non-nil input with only blank keys returns empty map", func(t *testing.T) {
		t.Parallel()
		m := ModelAttributes{
			" ":   {Value: 1},
			"\t":  {Value: 2},
			"   ": {Value: 3},
		}
		got := m.Normalize()
		if got == nil {
			t.Fatal("Normalize() returned nil for non-nil input, want empty map")
		}
		if len(got) != 0 {
			t.Fatalf("Normalize() len = %d, want 0 (got %#v)", len(got), got)
		}
	})

	t.Run("does not mutate the input map", func(t *testing.T) {
		t.Parallel()
		m := ModelAttributes{
			"  key  ": {Description: "desc", Value: "v"},
			"":       {Value: "drop"},
		}
		_ = m.Normalize()
		if _, ok := m["  key  "]; !ok {
			t.Fatal("Normalize() mutated input map (removed untrimmed key)")
		}
		if _, ok := m[""]; !ok {
			t.Fatal("Normalize() mutated input map (removed blank key)")
		}
	})
}
