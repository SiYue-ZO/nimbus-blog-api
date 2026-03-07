DROP TABLE IF EXISTS files;

DROP TABLE IF EXISTS notifications;

DROP TABLE IF EXISTS comment_likes;

DROP TABLE IF EXISTS comments;

DROP TABLE IF EXISTS post_likes;

DROP TABLE IF EXISTS post_tags;

DROP TABLE IF EXISTS post_views;

DROP TABLE IF EXISTS posts;

DROP TABLE IF EXISTS links;

DROP TABLE IF EXISTS site_settings;

DROP TABLE IF EXISTS feedbacks;

DROP TABLE IF EXISTS tags;

DROP TABLE IF EXISTS categories;

DROP TABLE IF EXISTS refresh_token_blacklist;

DROP TABLE IF EXISTS admin_recovery_codes;

DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS admins;

DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS maintain_category_post_count();
DROP FUNCTION IF EXISTS maintain_tag_post_count();

DROP TYPE IF EXISTS link_status;
DROP TYPE IF EXISTS setting_type;
DROP TYPE IF EXISTS comment_status;
DROP TYPE IF EXISTS post_status;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS feedback_status;
DROP TYPE IF EXISTS feedback_type;
DROP TYPE IF EXISTS file_usage;
DROP TYPE IF EXISTS notification_type;

DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS pg_trgm;
