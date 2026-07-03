-- Add openai_ws_mode flag to usage_logs to persist exact OpenAI WS transport type.
ALTER TABLE usage_logs ADD COLUMN openai_ws_mode BOOLEAN NOT NULL DEFAULT FALSE;
