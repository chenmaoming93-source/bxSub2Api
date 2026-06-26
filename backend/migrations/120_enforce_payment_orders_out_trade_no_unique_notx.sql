-- Build the payment order uniqueness guarantee for GoldenDB/MySQL mode.
-- The migration runner performs an explicit duplicate out_trade_no precheck and
-- drops any stale invalid paymentorder_out_trade_no_unique index before retrying.
-- MySQL-compatible databases do not support PostgreSQL partial unique indexes,
-- so use a generated NULLIF key. Multiple NULL values are allowed by the unique
-- index while non-empty out_trade_no values remain unique.
ALTER TABLE payment_orders
    ADD COLUMN out_trade_no_unique_key VARCHAR(191)
    GENERATED ALWAYS AS (NULLIF(out_trade_no, '')) STORED;

CREATE UNIQUE INDEX paymentorder_out_trade_no_unique
    ON payment_orders (out_trade_no_unique_key);

DROP INDEX IF EXISTS paymentorder_out_trade_no ON payment_orders;
