# 模型使用观测台 v2 原型

## 文件

- `design-spec.md`: 修订设计、功能差异、交互和验收标准。
- `prototype.html`: 可交互原型，链路下钻区位于观测总览同一页面下方。

## 本地查看

```powershell
cd dashboard-demo/revision-v2
python -m http.server 4173 --bind 127.0.0.1
```

打开：

- 总览：http://127.0.0.1:4173/prototype.html
- 链路定位：http://127.0.0.1:4173/prototype.html#drilldown

## 状态与原型范围

- 状态：`CONFIRMED`（2026-11-14）。
- 只确认页面结构和交互，不接真实后端。
- 使用确定性模拟数据。
- Dark mode、筛选选择器、模式切换和详情抽屉可直接操作。
- 原型用于视觉和主路径确认；生产组件的 loading / empty / error / partial / unavailable 五态、键盘导航、焦点约束和 `aria` 契约由 `frontend/` 实现及自动化测试验证。
- v1 `../index.html` 已标记为 `DEPRECATED`，不得继续以其环境选择器或旧交互作为实现依据。

## 原型截图

- `screenshots/overview-light-wide.png`
- `screenshots/drilldown-inline-page-wide.png`
- `screenshots/time-range-custom-1440.png`
- `screenshots/concurrency-capacity-card-wide.png`
- `screenshots/overview-mobile-390.png`
- `screenshots/time-range-custom-mobile-500.png`
- `screenshots/overview-1280-light.png`
- `screenshots/overview-768-light.png`
- `screenshots/overview-1280-dark.png`
- `screenshots/heat-range-concurrency-tooltip-wide.png`
- `screenshots/heat-account-concurrency-tooltip-wide.png`

## 已确认

`design-spec.md` 第 10 节已确认：单页信息架构、热力图与链路联动、删除环境选择器、保留 7 KPI、四列主链路和设计说明仅存在于原型。
