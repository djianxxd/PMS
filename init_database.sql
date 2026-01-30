-- goblog 数据库初始化脚本
-- 使用 root 用户运行此脚本

-- 1. 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS goblog CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 2. 选择数据库
USE goblog;

-- 3. 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 4. 创建分类表
CREATE TABLE IF NOT EXISTS categories (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    icon VARCHAR(50),
    color VARCHAR(50),
    is_default INT DEFAULT 0,
    is_custom INT DEFAULT 0,
    sort_order INT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 5. 创建交易表
CREATE TABLE IF NOT EXISTS transactions (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    type VARCHAR(50),
    category_id INT,
    category VARCHAR(255),
    amount DECIMAL(10,2),
    date DATETIME,
    note TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(category_id) REFERENCES categories(id),
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- 6. 创建财务目标表
CREATE TABLE IF NOT EXISTS finance_goals (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    type VARCHAR(50),
    target_amount DECIMAL(10,2),
    start_date DATETIME,
    end_date DATETIME,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- 7. 创建习惯表
CREATE TABLE IF NOT EXISTS habits (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    name VARCHAR(255),
    description TEXT,
    frequency VARCHAR(50),
    streak INT DEFAULT 0,
    total_days INT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- 8. 创建习惯记录表
CREATE TABLE IF NOT EXISTS habit_logs (
    id INT PRIMARY KEY AUTO_INCREMENT,
    habit_id INT,
    date DATETIME,
    FOREIGN KEY(habit_id) REFERENCES habits(id)
);

-- 9. 创建待办事项表
CREATE TABLE IF NOT EXISTS todos (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    content TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    due_date DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- 10. 创建待办事项检查表
CREATE TABLE IF NOT EXISTS todo_checkins (
    id INT PRIMARY KEY AUTO_INCREMENT,
    todo_id INT,
    checkin_date DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(todo_id) REFERENCES todos(id)
);

-- 11. 创建徽章表
CREATE TABLE IF NOT EXISTS badges (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    name VARCHAR(255),
    description TEXT,
    icon VARCHAR(50),
    unlocked INT DEFAULT 0,
    condition_days INT,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- 12. 创建日记表
CREATE TABLE IF NOT EXISTS diaries (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(255),
    content TEXT,
    weather VARCHAR(50),
    mood VARCHAR(50),
    date DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- 13. 启用外键约束
SET FOREIGN_KEY_CHECKS = 1;

-- 14. 插入默认收入分类
INSERT IGNORE INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES
('工资收入', 'income', '💰', '#10B981', 1, 0, 1),
('奖金福利', 'income', '🎁', '#10B981', 1, 0, 2),
('投资理财', 'income', '📈', '#10B981', 1, 0, 3),
('副业兼职', 'income', '💼', '#10B981', 1, 0, 4),
('经营收入', 'income', '🏪', '#10B981', 1, 0, 5),
('其他收入', 'income', '💵', '#10B981', 1, 0, 6),
('自定义输入', 'income', '✏️', '#6B7280', 1, 0, 999);

-- 15. 插入默认支出分类
INSERT IGNORE INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES
('餐饮美食', 'expense', '🍽️', '#EF4444', 1, 0, 1),
('超市购物', 'expense', '🛒', '#EF4444', 1, 0, 2),
('交通出行', 'expense', '🚗', '#EF4444', 1, 0, 3),
('休闲娱乐', 'expense', '🎮', '#EF4444', 1, 0, 4),
('房租房贷', 'expense', '🏠', '#EF4444', 1, 0, 5),
('水电物业', 'expense', '💡', '#EF4444', 1, 0, 6),
('医疗保健', 'expense', '🏥', '#EF4444', 1, 0, 7),
('教育学习', 'expense', '📚', '#EF4444', 1, 0, 8),
('人情往来', 'expense', '🎁', '#EF4444', 1, 0, 9),
('运动健身', 'expense', '🏃', '#EF4444', 1, 0, 10),
('美容护肤', 'expense', '💄', '#EF4444', 1, 0, 11),
('服饰鞋包', 'expense', '👔', '#EF4444', 1, 0, 12),
('通讯费用', 'expense', '📱', '#EF4444', 1, 0, 13),
('其他支出', 'expense', '📝', '#EF4444', 1, 0, 14),
('自定义输入', 'expense', '✏️', '#6B7280', 1, 0, 999);

-- 16. 创建普通用户并授予权限
-- 注意：请将 'your_password' 替换为实际的密码
CREATE USER IF NOT EXISTS 'goblog_user'@'localhost' IDENTIFIED BY 'your_password';
GRANT SELECT, INSERT, UPDATE, DELETE ON goblog.* TO 'goblog_user'@'localhost';
FLUSH PRIVILEGES;

-- 17. 完成信息
SELECT '数据库初始化完成！' AS message;
SELECT '请使用以下配置连接数据库：' AS message;
SELECT '用户: goblog_user' AS message;
SELECT '密码: your_password (请替换为实际密码)' AS message;
SELECT '数据库: goblog' AS message;