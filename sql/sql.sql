-- 管理员用户表
CREATE TABLE admin_users (
    admin_id INTEGER PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    full_name VARCHAR(100),
    position VARCHAR(100),
    telegram_id BIGINT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 普通用户表
CREATE TABLE regular_users (
    user_id INTEGER PRIMARY KEY,
    user_group VARCHAR(50) NOT NULL,
    telegram_id BIGINT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 工单表
CREATE TABLE tickets (
    ticket_id INTEGER PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'open',
    priority VARCHAR(20) DEFAULT 'normal',
    created_by INTEGER,
    assigned_to INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES regular_users(user_id),
    FOREIGN KEY (assigned_to) REFERENCES admin_users(admin_id)
);

-- 工单评论表
CREATE TABLE ticket_comments (
    comment_id INTEGER PRIMARY KEY,
    ticket_id INTEGER,
    user_id INTEGER,
    admin_id INTEGER,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (ticket_id) REFERENCES tickets(ticket_id),
    FOREIGN KEY (user_id) REFERENCES regular_users(user_id),
    FOREIGN KEY (admin_id) REFERENCES admin_users(admin_id)
);

-- 工单历史记录表
CREATE TABLE ticket_history (
    history_id INTEGER PRIMARY KEY,
    ticket_id INTEGER,
    user_id INTEGER,
    admin_id INTEGER,
    action VARCHAR(50) NOT NULL,
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (ticket_id) REFERENCES tickets(ticket_id),
    FOREIGN KEY (user_id) REFERENCES regular_users(user_id),
    FOREIGN KEY (admin_id) REFERENCES admin_users(admin_id)
);