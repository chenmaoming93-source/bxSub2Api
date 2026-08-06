package domain

import "strings"

// ModelAttributeItem 描述一条模型基本属性的「值部分」。
// Description 为中文描述（可选）；Value 为动态类型的属性值，
// 由前端解析后原样提交，后端信任前端，不做类型改写。
type ModelAttributeItem struct {
	Description string `json:"description,omitempty"`
	Value       any    `json:"value"`
}

// ModelAttributes 是模型账户的「模型基本属性」集合，以 JSON map 形式整体持久化
// 在 accounts.model_attributes 列：属性名（英文）为 key，描述与值为 value。
// 与账户 1:1 关联，供管理端创建/编辑账户时维护与回显，未来可对外提供查询。
type ModelAttributes map[string]ModelAttributeItem

// Normalize 对模型属性做最小防御规整：丢弃 key 去首尾空白后为空的条目，
// 其余条目（description / value）原样保留，不做类型解析、枚举校验或内容改写。
// 入参为 nil 时返回 nil；非 nil 输入始终返回非 nil map（可能为空 map，
// 用于表达「显式清空」语义）。
func (m ModelAttributes) Normalize() ModelAttributes {
	if m == nil {
		return nil
	}
	out := make(ModelAttributes, len(m))
	for key, item := range m {
		k := strings.TrimSpace(key)
		if k == "" {
			continue
		}
		out[k] = item
	}
	if len(out) == 0 {
		return ModelAttributes{}
	}
	return out
}
