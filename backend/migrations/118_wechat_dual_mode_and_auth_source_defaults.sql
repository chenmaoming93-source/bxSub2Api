INSERT IGNORE INTO settings (`key`, value)
SELECT new_settings.`key`, new_settings.value
FROM (
    SELECT
        'wechat_connect_open_enabled' AS `key`,
        CASE
            WHEN legacy.wechat_enabled IS NULL THEN ''
            WHEN COALESCE(legacy.wechat_enabled, 'false') <> 'true' THEN 'false'
            WHEN LOWER(TRIM(COALESCE(legacy.wechat_mode, 'open'))) = 'mp' THEN 'false'
            ELSE 'true'
        END AS value
    FROM (
        SELECT
            MAX(CASE WHEN `key` = 'wechat_connect_enabled' THEN value END) AS wechat_enabled,
            MAX(CASE WHEN `key` = 'wechat_connect_mode' THEN value END) AS wechat_mode
        FROM settings
        WHERE `key` IN ('wechat_connect_enabled', 'wechat_connect_mode')
    ) AS legacy

    UNION ALL

    SELECT
        'wechat_connect_mp_enabled' AS `key`,
        CASE
            WHEN legacy.wechat_enabled IS NULL THEN ''
            WHEN COALESCE(legacy.wechat_enabled, 'false') <> 'true' THEN 'false'
            WHEN LOWER(TRIM(COALESCE(legacy.wechat_mode, 'open'))) = 'mp' THEN 'true'
            ELSE 'false'
        END AS value
    FROM (
        SELECT
            MAX(CASE WHEN `key` = 'wechat_connect_enabled' THEN value END) AS wechat_enabled,
            MAX(CASE WHEN `key` = 'wechat_connect_mode' THEN value END) AS wechat_mode
        FROM settings
        WHERE `key` IN ('wechat_connect_enabled', 'wechat_connect_mode')
    ) AS legacy

    UNION ALL SELECT 'auth_source_default_email_grant_on_signup', 'false'
    UNION ALL SELECT 'auth_source_default_linuxdo_grant_on_signup', 'false'
    UNION ALL SELECT 'auth_source_default_oidc_grant_on_signup', 'false'
    UNION ALL SELECT 'auth_source_default_wechat_grant_on_signup', 'false'
) AS new_settings;
