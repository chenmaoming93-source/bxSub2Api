-- 148_add_ops_error_logs_user_time_index_notx.sql
-- 用户侧“错误请求”按 user_id 时间倒序分页所需的索引。
-- 非事务迁移（_notx）：CREATE INDEX 不可在事务内执行。
CREATE INDEX idx_ops_error_logs_user_time
  ON ops_error_logs (user_id, created_at DESC);
