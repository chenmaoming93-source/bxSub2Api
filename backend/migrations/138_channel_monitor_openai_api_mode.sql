-- Migration: 137_channel_monitor_openai_api_mode
-- Adds OpenAI API mode to channel monitors and request templates.

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

ALTER TABLE channel_monitor_request_templates
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

ALTER TABLE channel_monitors
    ADD CONSTRAINT channel_monitors_api_mode_check
    CHECK (api_mode IN ('chat_completions', 'responses'));

ALTER TABLE channel_monitor_request_templates
    ADD CONSTRAINT channel_monitor_request_templates_api_mode_check
    CHECK (api_mode IN ('chat_completions', 'responses'));

CREATE INDEX IF NOT EXISTS idx_channel_monitors_provider_api_mode
    ON channel_monitors (provider, api_mode);

CREATE INDEX IF NOT EXISTS idx_channel_monitor_templates_provider_api_mode
    ON channel_monitor_request_templates (provider, api_mode);
