-- 173_add_user_department.sql
-- Store the current LDAP department on the user profile.

ALTER TABLE users
    ADD COLUMN department VARCHAR(255) NOT NULL DEFAULT '' AFTER username;
