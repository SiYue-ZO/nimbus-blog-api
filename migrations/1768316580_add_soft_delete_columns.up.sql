-- Add soft delete column deleted_at to core tables
ALTER TABLE admins ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE categories ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE tags ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE posts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE site_settings ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE feedbacks ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
ALTER TABLE links ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- Replace UNIQUE constraints with partial unique indexes that ignore soft-deleted rows
-- admins.username
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'admins_username_key'
  ) THEN
    ALTER TABLE admins DROP CONSTRAINT admins_username_key;
  END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_admins_username_active ON admins (username) WHERE deleted_at IS NULL;

-- users (auth_provider, auth_openid) unique pair
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'users_auth_openid_pair_unique'
  ) THEN
    ALTER TABLE users DROP CONSTRAINT users_auth_openid_pair_unique;
  END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_auth_pair_active ON users (auth_provider, auth_openid) WHERE deleted_at IS NULL;

-- users.email unique (email can be reused after soft delete)
DROP INDEX IF EXISTS idx_users_email_not_null;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_not_null_active
ON users (email)
WHERE email IS NOT NULL
  AND deleted_at IS NULL;

-- categories.name, categories.slug
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'categories_name_key'
  ) THEN
    ALTER TABLE categories DROP CONSTRAINT categories_name_key;
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'categories_slug_key'
  ) THEN
    ALTER TABLE categories DROP CONSTRAINT categories_slug_key;
  END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name_active ON categories (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug_active ON categories (slug) WHERE deleted_at IS NULL;

-- tags.name, tags.slug
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'tags_name_key'
  ) THEN
    ALTER TABLE tags DROP CONSTRAINT tags_name_key;
  END IF;
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'tags_slug_key'
  ) THEN
    ALTER TABLE tags DROP CONSTRAINT tags_slug_key;
  END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_name_active ON tags (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_slug_active ON tags (slug) WHERE deleted_at IS NULL;

-- posts.slug
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'posts_slug_key'
  ) THEN
    ALTER TABLE posts DROP CONSTRAINT posts_slug_key;
  END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_slug_active ON posts (slug) WHERE deleted_at IS NULL;

-- site_settings.setting_key
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'site_settings_setting_key_key'
  ) THEN
    ALTER TABLE site_settings DROP CONSTRAINT site_settings_setting_key_key;
  END IF;
END $$;
DROP INDEX IF EXISTS idx_site_settings_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_site_settings_key_active
ON site_settings (setting_key)
WHERE deleted_at IS NULL;
