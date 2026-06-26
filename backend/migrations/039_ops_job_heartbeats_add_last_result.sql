-- Add last_result to ops_job_heartbeats for UI job details.

ALTER TABLE ops_job_heartbeats
    ADD COLUMN last_result TEXT;
