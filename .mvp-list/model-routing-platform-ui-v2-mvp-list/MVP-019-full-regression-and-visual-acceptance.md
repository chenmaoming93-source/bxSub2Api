# MVP-019：完成全站回归与 UI V2 视觉验收

- Protocol: `mvp-list/v1`
- State: `PLANNED`
- Estimate: `1 个开发日`
- Estimate rationale: 聚焦最终集成、自动测试、视觉和路由验收，作为独立收口交付。
- Dependencies: `MVP-009, MVP-010, MVP-013, MVP-014, MVP-016, MVP-017, MVP-018`

## 预期成果

所有纳入范围的页面完成功能回归、视觉验收、权限验证和旧代码保留检查。

## 背景

参考原型：

```text
assets/final-admin-dashboard-prototype-v2.png
assets/final-admin-dashboard-scene-modal.png
assets/final-admin-dashboard-department-modal.png
assets/final-admin-usage-prototype.png
assets/final-ops-prototype.png
assets/final-personal-dashboard-prototype-v2.png
```

## 范围内

- Part 1～10 功能回归。
- UI V2 组件回归。
- 亮色、暗色、桌面和移动端检查。
- 所有详情弹窗交互检查。
- 停用路由和菜单检查。
- 旧页面、旧组件存在性检查。
- 记录截图、命令和已知问题。

## 范围外

- 不在此 MVP 中新增业务功能。
- 不清理旧代码。
- 不改变既定接口口径。

## 实现说明

视觉验收以六张原型图为基线，重点检查布局、颜色、信息密度、排行方向、弹窗形式和图表主题。

## 验收标准

- [ ] 主要页面功能和权限回归通过。
- [ ] UI V2 风格在目标页面保持一致。
- [ ] 亮色、暗色和响应式无明显布局错误。
- [ ] 停用路由不可访问。
- [ ] 旧文件仍存在。
- [ ] 所有测试和构建通过。

## 验证计划

- `pnpm --dir frontend typecheck`
- `pnpm --dir frontend test:run`
- `pnpm --dir frontend build`
- 启动项目进行管理员、普通用户和停用 URL 人工验收。

## 完成证据

| 类型 | 命令或路径 | 结果 |
|---|---|---|

## 执行记录

