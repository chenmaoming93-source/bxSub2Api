package rbac

// BaselineSummary freezes the approved migration inventory. Runtime route
// closure is implemented later by the Registry; this baseline keeps changes to
// the source inventory visible during that migration.
type BaselineSummary struct {
	Total      int
	Controlled int
	Excluded   int
}

var ApprovedRouteBaseline = BaselineSummary{
	Total:      522,
	Controlled: 427,
	Excluded:   95,
}

type ExclusionDefinition struct {
	Category       string
	Count          int
	Authentication string
	Reason         string
}

var ApprovedExclusions = []ExclusionDefinition{
	{Category: "public_auth", Count: 54, Authentication: "public, temporary token, OAuth state, or signed resume token", Reason: "登录和公开流程没有站内用户角色主体"},
	{Category: "payment_webhook", Count: 6, Authentication: "payment provider signature", Reason: "服务商服务器回调使用独立签名信任边界"},
	{Category: "external_integration", Count: 3, Authentication: "external integration key", Reason: "外部系统主体不属于站内用户 RBAC"},
	{Category: "model_gateway", Count: 32, Authentication: "API key and group/subscription constraints", Reason: "模型调用方使用独立 API Key 授权模型"},
}
