-- 添加分组支持的模型系列字段
ALTER TABLE groups
ADD COLUMN IF NOT EXISTS supported_model_scopes JSON NULL;
