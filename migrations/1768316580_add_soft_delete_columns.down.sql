-- Revert soft delete changes: drop partial unique indexes, restore constraints, drop deleted_at columns

-- Drop partial unique indexes
DROP INDEX IF EXISTS idx_admins_username_active;
DROP INDEX IF EXISTS idx_users_auth_pair_active;
DROP INDEX IF EXISTS idx_users_email_not_null_active;
DROP INDEX IF EXISTS idx_categories_name_active;
DROP INDEX IF EXISTS idx_categories_slug_active;
DROP INDEX IF EXISTS idx_tags_name_active;
DROP INDEX IF EXISTS idx_tags_slug_active;
DROP INDEX IF EXISTS idx_posts_slug_active;
DROP INDEX IF EXISTS idx_site_settings_key_active;
DROP INDEX IF EXISTS idx_site_settings_key;

-- Restore original UNIQUE constraints
ALTER TABLE admins ADD CONSTRAINT admins_username_key UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_auth_openid_pair_unique UNIQUE (auth_provider, auth_openid);
ALTER TABLE categories ADD CONSTRAINT categories_name_key UNIQUE (name);
ALTER TABLE categories ADD CONSTRAINT categories_slug_key UNIQUE (slug);
ALTER TABLE tags ADD CONSTRAINT tags_name_key UNIQUE (name);
ALTER TABLE tags ADD CONSTRAINT tags_slug_key UNIQUE (slug);
ALTER TABLE posts ADD CONSTRAINT posts_slug_key UNIQUE (slug);
ALTER TABLE site_settings ADD CONSTRAINT site_settings_setting_key_key UNIQUE (setting_key);

-- Restore original users.email unique index (email unique only when not null)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_not_null
ON users (email)
WHERE email IS NOT NULL;

-- Drop deleted_at columns
ALTER TABLE admins DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE categories DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE tags DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE posts DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE comments DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE site_settings DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE feedbacks DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE links DROP COLUMN IF EXISTS deleted_at;
