-- Add model basic attributes to accounts as a JSON map.
-- 目的：为 accounts 表新增 model_attributes JSON 列，保存模型账户的模型基本属性
-- （如上下文窗口大小、是否支持推理、是否支持图片输入等），结构为
-- {属性名(英文): {description: 中文描述, value: 动态类型}}，与账户 1:1 关联。
-- 该列可空（NULL = 未配置）；前端提交空对象 {} 表示显式空配置。
-- 方言：MySQL 8 / GoldenDB（MySQL 兼容模式）。
ALTER TABLE accounts
    ADD COLUMN model_attributes JSON NULL
    COMMENT 'Model basic attributes map: {attrName: {description, value}}.';
