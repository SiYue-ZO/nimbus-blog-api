CREATE TYPE user_status AS ENUM ('active', 'disabled');
CREATE TYPE post_status AS ENUM ('draft', 'published', 'archived');
CREATE TYPE comment_status AS ENUM ('pending', 'approved', 'rejected', 'spam');
CREATE TYPE feedback_type AS ENUM ('general', 'bug', 'feature', 'ui');
CREATE TYPE feedback_status AS ENUM ('pending', 'processing', 'resolved', 'closed');
CREATE TYPE setting_type AS ENUM ('string', 'number', 'boolean', 'json');
CREATE TYPE link_status AS ENUM ('active', 'inactive');
CREATE TYPE file_usage AS ENUM ('post_cover', 'post_content', 'avatar');
CREATE TYPE notification_type AS ENUM ('comment_reply', 'comment_approved', 'admin_message');

-- Create trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Admins table: standalone admin identities
CREATE TABLE IF NOT EXISTS admins (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100) NOT NULL,
    specialization VARCHAR(100) DEFAULT '' NOT NULL,
    must_reset_password BOOLEAN DEFAULT FALSE NOT NULL,
    two_factor_secret VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create trigger for updated_at
CREATE TRIGGER update_admins_updated_at
BEFORE UPDATE ON admins
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Seed initial administrator account
CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO admins (
    username,
    password_hash,
    nickname,
    specialization,
    must_reset_password
) VALUES (
    'nimbus-admin',
    crypt('12345678', gen_salt('bf')),
    'admin',
    '全栈工程师',
    TRUE
)
ON CONFLICT (username) DO NOTHING;

-- Admin recovery codes: 2FA backup codes linked to admin
CREATE TABLE IF NOT EXISTS admin_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT NOT NULL,
    code_hash VARCHAR(255) NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE (admin_id, code_hash),
    FOREIGN KEY (admin_id) REFERENCES admins (id) ON DELETE CASCADE
);

-- Users table: site members and commenters
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    avatar VARCHAR(500) DEFAULT '/avatar.png' NOT NULL,
    bio TEXT DEFAULT '该用户尚未填写个人简介。' NOT NULL,
    status user_status DEFAULT 'active' NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE NOT NULL,
    region VARCHAR(255),
    blog_url VARCHAR(500),
    auth_provider VARCHAR(50),
    auth_openid VARCHAR(255),
    show_full_profile BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Users auth constraints: provider/openid pairing
ALTER TABLE users
    ADD CONSTRAINT users_auth_provider_allowed CHECK (
        auth_provider IS NULL
        OR auth_provider IN ('qq')
    );

ALTER TABLE users
    ADD CONSTRAINT users_auth_openid_pair_unique UNIQUE (auth_provider, auth_openid);

ALTER TABLE users
    ADD CONSTRAINT users_auth_pair_nullness CHECK (
        (
            auth_provider IS NULL
            AND auth_openid IS NULL
        )
        OR (
            auth_provider IS NOT NULL
            AND auth_openid IS NOT NULL
        )
    );

-- Create indexes for users table
CREATE UNIQUE INDEX idx_users_email_not_null
ON users (email)
WHERE email IS NOT NULL;

CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Refresh token blacklist: store revoked/expired refresh tokens
CREATE TABLE IF NOT EXISTS refresh_token_blacklist (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE (token_hash)
);

-- Create indexes for refresh_token_blacklist table
CREATE INDEX IF NOT EXISTS idx_refresh_token_blacklist_user_id
ON refresh_token_blacklist (user_id);

CREATE INDEX IF NOT EXISTS idx_refresh_token_blacklist_expires_at
ON refresh_token_blacklist (expires_at);

-- Categories table: post categories and counts
CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    post_count INTEGER DEFAULT 0 NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create trigger for updated_at
CREATE TRIGGER update_categories_updated_at
BEFORE UPDATE ON categories
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Tags table: post tags and counts
CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    post_count INTEGER DEFAULT 0 NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create trigger for updated_at
CREATE TRIGGER update_tags_updated_at
BEFORE UPDATE ON tags
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Posts table: blog articles and metadata
CREATE TABLE IF NOT EXISTS posts (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    excerpt TEXT NOT NULL,
    content TEXT NOT NULL,
    featured_image VARCHAR(500),
    author_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    status post_status DEFAULT 'draft' NOT NULL,
    read_time VARCHAR(20) NOT NULL,
    views INTEGER DEFAULT 0 NOT NULL,
    likes INTEGER DEFAULT 0 NOT NULL,
    is_featured BOOLEAN DEFAULT FALSE NOT NULL,
    meta_title VARCHAR(255),
    meta_description TEXT,
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (author_id) REFERENCES admins (id) ON DELETE RESTRICT,
    FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE RESTRICT
);

-- Create indexes for posts table
CREATE INDEX IF NOT EXISTS idx_posts_category_published_at
ON posts (category_id, published_at DESC)
WHERE status = 'published';

CREATE INDEX IF NOT EXISTS idx_posts_featured_published_at
ON posts (published_at DESC)
WHERE is_featured
  AND status = 'published';

-- Create trigram search index (requires pg_trgm)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_posts_search_trgm ON posts USING gin (
    (
        title || ' ' || COALESCE(excerpt, '') || ' ' || content
    ) gin_trgm_ops
);

-- Create trigger for updated_at
CREATE TRIGGER update_posts_updated_at
BEFORE UPDATE ON posts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Post-tags join: many-to-many relation
CREATE TABLE IF NOT EXISTS post_tags (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL,
    tag_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE RESTRICT,
    UNIQUE (post_id, tag_id)
);

-- Create indexes for post_tags table
CREATE INDEX IF NOT EXISTS idx_post_tags_tag_id
ON post_tags (tag_id);

-- Maintain categories.post_count based on posts changes
CREATE OR REPLACE FUNCTION maintain_category_post_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.category_id IS NOT NULL THEN
            UPDATE categories
            SET post_count = post_count + 1
            WHERE id = NEW.category_id;
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.category_id IS NOT NULL THEN
            UPDATE categories
            SET post_count = post_count - 1
            WHERE id = OLD.category_id;
        END IF;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.category_id IS DISTINCT FROM NEW.category_id THEN
            IF OLD.category_id IS NOT NULL THEN
                UPDATE categories
                SET post_count = post_count - 1
                WHERE id = OLD.category_id;
            END IF;
            IF NEW.category_id IS NOT NULL THEN
                UPDATE categories
                SET post_count = post_count + 1
                WHERE id = NEW.category_id;
            END IF;
        END IF;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER maintain_posts_category_post_count
AFTER INSERT OR DELETE OR UPDATE OF category_id ON posts
FOR EACH ROW
EXECUTE FUNCTION maintain_category_post_count();

-- Maintain tags.post_count based on post_tags changes
CREATE OR REPLACE FUNCTION maintain_tag_post_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tags
        SET post_count = post_count + 1
        WHERE id = NEW.tag_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE tags
        SET post_count = post_count - 1
        WHERE id = OLD.tag_id;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.tag_id IS DISTINCT FROM NEW.tag_id THEN
            UPDATE tags
            SET post_count = post_count - 1
            WHERE id = OLD.tag_id;

            UPDATE tags
            SET post_count = post_count + 1
            WHERE id = NEW.tag_id;
        END IF;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER maintain_post_tags_count_ins
AFTER INSERT ON post_tags
FOR EACH ROW
EXECUTE FUNCTION maintain_tag_post_count();

CREATE TRIGGER maintain_post_tags_count_del
AFTER DELETE ON post_tags
FOR EACH ROW
EXECUTE FUNCTION maintain_tag_post_count();

CREATE TRIGGER maintain_post_tags_count_upd
AFTER UPDATE OF tag_id ON post_tags
FOR EACH ROW
EXECUTE FUNCTION maintain_tag_post_count();

-- Post likes: track user likes per post
CREATE TABLE IF NOT EXISTS post_likes (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    liked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE (post_id, user_id)
);

-- Create indexes for post_likes table
CREATE INDEX IF NOT EXISTS idx_post_likes_user_id
ON post_likes (user_id);

-- Post views table: track visits for analytics
CREATE TABLE IF NOT EXISTS post_views (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    referer VARCHAR(500),
    viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE
);

-- Comments: article comments with moderation
CREATE TABLE IF NOT EXISTS comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL,
    parent_id BIGINT,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    status comment_status DEFAULT 'pending' NOT NULL,
    likes INTEGER DEFAULT 0 NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (post_id) REFERENCES posts (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES comments (id) ON DELETE CASCADE
);

-- Create trigger for updated_at
CREATE TRIGGER update_comments_updated_at
BEFORE UPDATE ON comments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Comment likes: prevent duplicate likes and count
CREATE TABLE IF NOT EXISTS comment_likes (
    id BIGSERIAL PRIMARY KEY,
    comment_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    liked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (comment_id) REFERENCES comments (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    UNIQUE (comment_id, user_id)
);

-- Site settings: key-value configuration
CREATE TABLE IF NOT EXISTS site_settings (
    id BIGSERIAL PRIMARY KEY,
    setting_key VARCHAR(100) UNIQUE NOT NULL,
    setting_value TEXT NOT NULL,
    setting_type setting_type DEFAULT 'string' NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create trigger for updated_at
CREATE TRIGGER update_site_settings_updated_at
BEFORE UPDATE ON site_settings
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Seed site_settings
INSERT INTO site_settings (setting_key, setting_value, setting_type, description, is_public)
VALUES
    ('site.name', 'Nimbus Blog', 'string', '站点名称', TRUE),
    ('site.title', 'Nimbus Blog - 现代技术博客', 'string', '站点标题', TRUE),
    ('site.description', '专注分享前端、后端与云原生的技术文章与实践', 'string', '站点描述', TRUE),
    ('site.slogan', 'Where Thoughts Leave Their Trace.', 'string', '站点标语', TRUE),
    ('site.hero', E'聚焦现代 Web 技术栈与工程实践。\n记录架构设计、性能优化与开发经验，共同成长。', 'string', '首页 Hero 介绍', TRUE),
    ('site.icp_record', '', 'string', 'ICP 备案号', TRUE),
    ('site.police_record', '', 'string', '公安备案号', TRUE),
    ('site.faq', '[{"title":"如何开始使用这个博客？","content":"这是一个简单的博客系统，您可以浏览文章、查看分类和标签。如果您想要更多功能，请联系管理员。"},{"title":"如何搜索文章？","content":"您可以使用导航栏中的搜索框来搜索文章。支持按标题、内容和标签进行搜索。"},{"title":"如何订阅RSS？","content":"点击导航栏中的''RSS订阅''按钮，或者直接访问 /rss.xml 来获取RSS订阅源。"},{"title":"网站支持哪些浏览器？","content":"本网站支持所有现代浏览器，包括Chrome、Firefox、Safari、Edge等。建议使用最新版本以获得最佳体验。"},{"title":"如何切换主题？","content":"点击导航栏右上角的主题切换按钮，可以在浅色模式和深色模式之间切换。"},{"title":"移动端体验如何？","content":"本网站采用响应式设计，完全适配移动设备。您可以在手机和平板上获得良好的浏览体验。"}]', 'json', '常见问题', TRUE),
    ('profile.name', '博主', 'string', '个人昵称', TRUE),
    ('profile.avatar', '/author.png', 'string', '个人头像', TRUE),
    ('profile.bio', E'我是一名热爱开源与技术分享的开发者。\n关注前端、后端与云原生，记录实践经验与学习心得，欢迎交流。', 'string', '个人简介', TRUE),
    ('profile.tech_stack', '["Go","Fiber","PostgreSQL","Redis","Docker","Nginx","React","Next.js","TypeScript","MinIO"]', 'json', '技术栈', TRUE),
    ('profile.work_experiences', '[{"title":"Web 开发工程师","company":"互联网公司","period":"2019 - 2021","description":"参与 Web 应用开发与维护，积累工程实践。"},{"title":"全栈工程师","company":"技术团队","period":"2021 - 2023","description":"负责前后端开发与部署，推动工程效率提升。"},{"title":"技术顾问","company":"开源社区/企业","period":"2023 - 至今","description":"分享技术经验与最佳实践，参与社区建设。"}]', 'json', '工作经历', TRUE),
    ('profile.project_experiences', '[{"name":"内容管理平台","description":"用于管理文章、分类与标签的 CMS 系统","tech":["React","TypeScript","PostgreSQL"]},{"name":"技术博客站点","description":"基于现代前端框架构建的个人/团队博客","tech":["Next.js","Tailwind CSS","HeroUI"]},{"name":"数据分析工具","description":"用于指标采集与可视化的应用","tech":["Go","Docker","Grafana"]}]', 'json', '项目经历', TRUE),
    ('profile.github_url', 'https://github.com/yourname', 'string', 'GitHub 链接', TRUE),
    ('profile.bilibili_url', 'https://space.bilibili.com/000000000', 'string', 'Bilibili 链接', TRUE),
    ('profile.qq_group_url', 'https://qm.qq.com/q/XXXXXXXXXX', 'string', 'QQ 群链接', TRUE),
    ('profile.email', 'contact@example.com', 'string', '联系邮箱', TRUE);

-- Feedbacks: user feedback submissions
CREATE TABLE IF NOT EXISTS feedbacks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    type feedback_type DEFAULT 'general' NOT NULL,
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status feedback_status DEFAULT 'pending' NOT NULL,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create indexes for feedbacks table
CREATE INDEX IF NOT EXISTS idx_feedbacks_status
ON feedbacks (status);

CREATE INDEX IF NOT EXISTS idx_feedbacks_type
ON feedbacks (type);

CREATE TRIGGER update_feedbacks_updated_at
BEFORE UPDATE ON feedbacks
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Links: friend links for public page
CREATE TABLE IF NOT EXISTS links (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    url VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    logo VARCHAR(500) NOT NULL,
    sort_order INTEGER DEFAULT 0 NOT NULL,
    status link_status DEFAULT 'active' NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create trigger for updated_at
CREATE TRIGGER update_links_updated_at
BEFORE UPDATE ON links
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Notifications: in-app user notifications
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type notification_type NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_read BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- Create indexes for notifications table
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
ON notifications (user_id, is_read, created_at DESC);

-- Files: uploaded file metadata linked to MinIO objects
CREATE TABLE IF NOT EXISTS files (
    id BIGSERIAL PRIMARY KEY,
    object_key VARCHAR(512) UNIQUE NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT DEFAULT 0 NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    usage file_usage NOT NULL,
    resource_id BIGINT,
    uploader_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (uploader_id) REFERENCES admins (id) ON DELETE RESTRICT
);

-- Create indexes for files table
CREATE INDEX IF NOT EXISTS idx_files_usage_resource
ON files (usage, resource_id);

CREATE INDEX IF NOT EXISTS idx_files_resource_id
ON files (resource_id)
WHERE resource_id IS NOT NULL;

