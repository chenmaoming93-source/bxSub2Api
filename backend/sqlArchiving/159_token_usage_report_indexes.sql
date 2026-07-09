-- Bounded token-usage report and default-target lookup indexes.
CREATE INDEX idx_model_token_usage_date_tokens
    ON model_token_daily_usages (usage_date, used_tokens);

CREATE INDEX idx_group_candidate_token_report
    ON group_candidate_token_daily_usages (group_id, route_alias, usage_date, upstream_model);
CREATE INDEX idx_group_candidate_token_default
    ON group_candidate_token_daily_usages (usage_date, group_id, route_alias);

CREATE INDEX idx_user_model_token_report
    ON user_model_token_daily_usages (user_id, usage_date, model);
CREATE INDEX idx_user_model_token_default
    ON user_model_token_daily_usages (usage_date, user_id, model);

