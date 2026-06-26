-- 创建公告表
CREATE TABLE IF NOT EXISTS announcements (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    targeting JSON NULL,
    starts_at DATETIME(6) DEFAULT NULL,
    ends_at DATETIME(6) DEFAULT NULL,
    created_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

-- 公告已读表
CREATE TABLE IF NOT EXISTS announcement_reads (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    announcement_id BIGINT NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE(announcement_id, user_id)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_announcements_status ON announcements(status);
CREATE INDEX IF NOT EXISTS idx_announcements_starts_at ON announcements(starts_at);
CREATE INDEX IF NOT EXISTS idx_announcements_ends_at ON announcements(ends_at);
CREATE INDEX IF NOT EXISTS idx_announcements_created_at ON announcements(created_at);

CREATE INDEX IF NOT EXISTS idx_announcement_reads_announcement_id ON announcement_reads(announcement_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_user_id ON announcement_reads(user_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_read_at ON announcement_reads(read_at);
