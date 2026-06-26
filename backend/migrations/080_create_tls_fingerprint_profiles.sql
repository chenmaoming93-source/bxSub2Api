-- Create tls_fingerprint_profiles table for managing TLS fingerprint templates.
-- Each profile contains ClientHello parameters to simulate specific client TLS handshake characteristics.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS tls_fingerprint_profiles (
    id           BIGINT NOT NULL AUTO_INCREMENT    PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,
    description  TEXT,
    enable_grease BOOLEAN     NOT NULL DEFAULT false,
    cipher_suites        JSON,
    curves               JSON,
    point_formats        JSON,
    signature_algorithms JSON,
    alpn_protocols       JSON,
    supported_versions   JSON,
    key_share_groups     JSON,
    psk_modes            JSON,
    extensions           JSON,
    created_at   DATETIME(6)  NOT NULL DEFAULT NOW(),
    updated_at   DATETIME(6)  NOT NULL DEFAULT NOW()
);
