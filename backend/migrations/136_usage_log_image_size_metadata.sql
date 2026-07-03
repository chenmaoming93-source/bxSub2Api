-- Add generated-image billing size audit metadata.
-- `image_size` remains the canonical billing tier used for cost calculation.

ALTER TABLE usage_logs
    ADD COLUMN image_input_size VARCHAR(32);

ALTER TABLE usage_logs
    ADD COLUMN image_output_size VARCHAR(32);

ALTER TABLE usage_logs
    ADD COLUMN image_size_source VARCHAR(16);

ALTER TABLE usage_logs
    ADD COLUMN image_size_breakdown JSON;

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_image_size_source_check
    CHECK (
        image_size_source IS NULL
        OR image_size_source IN ('output', 'input', 'default', 'legacy')
    );

ALTER TABLE usage_logs
    ADD CONSTRAINT usage_logs_image_billing_size_check
    CHECK (
        image_count <= 0
        OR (
            image_size IS NOT NULL
            AND image_size IN ('1K', '2K', '4K', 'mixed')
        )
    );
