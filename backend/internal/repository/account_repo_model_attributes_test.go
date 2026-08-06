package repository

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestAccountEntityToService_ModelAttributes 验证读回映射：
// NULL 列读回 nil；非 nil map 保留 key/description/value（含类型）原样。
func TestAccountEntityToService_ModelAttributes(t *testing.T) {
	t.Run("nil column maps to nil", func(t *testing.T) {
		out := accountEntityToService(&dbent.Account{ID: 1, ModelAttributes: nil})
		require.NotNil(t, out)
		require.Nil(t, out.ModelAttributes)
	})

	t.Run("map round-trips with typed values preserved", func(t *testing.T) {
		attrs := domain.ModelAttributes{
			"context_window":  {Description: "上下文窗口总大小（token）", Value: 200000},
			"supports_vision": {Description: "支持图片输入", Value: true},
			"raw_string_true": {Description: "信任前端原样存储", Value: "true"},
			"modalities":      {Description: "模态列表", Value: []string{"text", "image"}},
		}
		out := accountEntityToService(&dbent.Account{ID: 7, ModelAttributes: attrs})
		require.NotNil(t, out)
		require.NotNil(t, out.ModelAttributes)
		require.Len(t, out.ModelAttributes, 4)
		require.Equal(t, 200000, out.ModelAttributes["context_window"].Value)
		require.Equal(t, true, out.ModelAttributes["supports_vision"].Value)
		require.Equal(t, "true", out.ModelAttributes["raw_string_true"].Value)
		require.Equal(t, []string{"text", "image"}, out.ModelAttributes["modalities"].Value)
		require.Equal(t, "上下文窗口总大小（token）", out.ModelAttributes["context_window"].Description)
	})

	t.Run("copy does not alias the source map", func(t *testing.T) {
		attrs := domain.ModelAttributes{"k": {Description: "d", Value: 1}}
		m := &dbent.Account{ID: 9, ModelAttributes: attrs}
		out := accountEntityToService(m)
		out.ModelAttributes["k"] = domain.ModelAttributeItem{Description: "changed", Value: 2}
		require.Equal(t, 1, m.ModelAttributes["k"].Value)
	})
}
