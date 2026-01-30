package db

import (
	"database/sql"
	"fmt"
	"log"

	"goblog/config"

	_ "github.com/go-sql-driver/mysql" // Import MySQL driver
)

var DB *sql.DB

func InitDB() error {
	var err error

	// MySQL connection parameters from config
	dbUser := config.AppConfig.MySQL.User
	dbPassword := config.AppConfig.MySQL.Password
	dbHost := config.AppConfig.MySQL.Host
	dbPort := config.AppConfig.MySQL.Port
	dbName := config.AppConfig.MySQL.Database

	// First try to connect to MySQL server without specifying database
	serverDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort)

	serverDB, err := sql.Open("mysql", serverDSN)
	if err != nil {
		log.Printf("Failed to connect to MySQL server: %v\n", err)
		return fmt.Errorf("连接 MySQL 服务器失败: %w\n请确保 MySQL 服务已启动并且数据库配置正确", err)
	}
	defer serverDB.Close()

	// Test server connection
	err = serverDB.Ping()
	if err != nil {
		log.Printf("Failed to ping MySQL server: %v\n", err)
		return fmt.Errorf("ping MySQL 服务器失败: %w\n请确保 MySQL 服务已启动并且数据库配置正确", err)
	}

	// Check if database exists
	var dbExists bool
	err = serverDB.QueryRow("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?", dbName).Scan(&dbExists)
	if err != nil {
		log.Printf("Failed to check if database exists: %v\n", err)
		return fmt.Errorf("检查数据库存在性失败: %w", err)
	}

	// Create database if it doesn't exist
	if !dbExists {
		log.Printf("Database %s does not exist, creating...", dbName)
		_, err = serverDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName))
		if err != nil {
			log.Printf("Failed to create database: %v\n", err)
			return fmt.Errorf("创建数据库失败: %w\n\n解决方案:\n1. 请确保 MySQL 用户有创建数据库的权限，或者\n2. 手动在 MySQL 中创建数据库: CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci\n3. 然后使用该数据库的普通用户权限重新配置", err, dbName)
		}
		log.Printf("Database %s created successfully", dbName)
	}

	// Now connect to the specific database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("Failed to connect to database: %v\n", err)
		return fmt.Errorf("连接数据库失败: %w\n请确保 MySQL 服务已启动并且数据库配置正确", err)
	}

	// Test database connection
	err = DB.Ping()
	if err != nil {
		log.Printf("Failed to ping database: %v\n", err)
		return fmt.Errorf("ping 数据库失败: %w\n请确保 MySQL 服务已启动并且数据库已创建", err)
	}

	log.Println("Successfully connected to MySQL database")

	err = createTables()
	if err != nil {
		log.Printf("Error creating tables: %v", err)
		return fmt.Errorf("创建数据库表失败: %w\n请确保 MySQL 用户有创建表的权限", err)
	}
	migrateDatabase()
	seedBadges()
	seedCategories()
	seedSampleData()

	// 验证分类是否成功初始化
	verifyCategories()

	return nil
}

func createTables() error {
	// First create users table since other tables reference it
	_, err := DB.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INT PRIMARY KEY AUTO_INCREMENT,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		log.Printf("Error creating users table: %v", err)
		return fmt.Errorf("创建用户表失败: %w\n请确保 MySQL 用户有创建表的权限", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			icon VARCHAR(50),
			color VARCHAR(50),
			is_default INT DEFAULT 0,
			is_custom INT DEFAULT 0,
			sort_order INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS transactions (
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
		);`,
		`CREATE TABLE IF NOT EXISTS finance_goals (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			type VARCHAR(50),
			target_amount DECIMAL(10,2),
			start_date DATETIME,
			end_date DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS habits (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			name VARCHAR(255),
			description TEXT,
			frequency VARCHAR(50),
			streak INT DEFAULT 0,
			total_days INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS habit_logs (
			id INT PRIMARY KEY AUTO_INCREMENT,
			habit_id INT,
			date DATETIME,
			FOREIGN KEY(habit_id) REFERENCES habits(id)
		);`,
		`CREATE TABLE IF NOT EXISTS todos (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			content TEXT,
			status VARCHAR(50) DEFAULT 'pending',
			due_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS todo_checkins (
			id INT PRIMARY KEY AUTO_INCREMENT,
			todo_id INT,
			checkin_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(todo_id) REFERENCES todos(id)
		);`,
		`CREATE TABLE IF NOT EXISTS badges (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			name VARCHAR(255),
			description TEXT,
			icon VARCHAR(50),
			unlocked INT DEFAULT 0,
			condition_days INT,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS diaries (
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
		);`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Printf("Error creating table: %s, %v", query, err)
			return fmt.Errorf("创建表失败: %w\n请确保 MySQL 用户有创建表的权限", err)
		}
	}

	// Enable foreign key constraints in MySQL
	_, err = DB.Exec("SET FOREIGN_KEY_CHECKS = 1")
	if err != nil {
		log.Printf("Error enabling foreign key constraints: %v", err)
		return fmt.Errorf("启用外键约束失败: %w", err)
	}

	return nil
}

// CreateUserBadges creates badges for a specific user
func CreateUserBadges(userID int) {
	// Check if badges already exist for this user
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM badges WHERE user_id = ?", userID).Scan(&count)
	if count > 0 {
		return
	}

	badges := []struct {
		Name        string
		Description string
		Icon        string
		Days        int
	}{
		{"初出茅庐", "完成第一次打卡", "🌱", 1},
		{"坚持不懈", "累计打卡7天", "🔥", 7},
		{"习惯养成", "累计打卡21天", "⭐", 21},
		{"自律大师", "累计打卡100天", "👑", 100},
	}

	for _, b := range badges {
		_, err := DB.Exec("INSERT INTO badges (user_id, name, description, icon, condition_days) VALUES (?, ?, ?, ?, ?)", userID, b.Name, b.Description, b.Icon, b.Days)
		if err != nil {
			log.Println("Error creating user badges:", err)
		}
	}
}

func seedBadges() {
	// Badges are now created per user, so this function is deprecated
	// We'll create badges when users register instead
	log.Println("Badges will be created per user upon registration")
}

func seedCategories() {
	// Check if categories table exists
	var tableExists bool
	err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		AND table_name = 'categories'
	`).Scan(&tableExists)

	if err != nil {
		log.Println("Warning: Error checking if categories table exists:", err)
		return
	}

	if !tableExists {
		log.Println("Warning: categories table does not exist, skipping seedCategories")
		return
	}

	// Check if categories exist
	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		log.Println("Warning: Error checking if categories exist:", err)
		return
	}

	if count > 0 {
		return
	}

	// Default income categories - 更合理的分类
	incomeCategories := []struct {
		Name      string
		Icon      string
		Color     string
		SortOrder int
	}{
		{"工资收入", "💰", "#10B981", 1},
		{"奖金福利", "🎁", "#10B981", 2},
		{"投资理财", "📈", "#10B981", 3},
		{"副业兼职", "💼", "#10B981", 4},
		{"经营收入", "🏪", "#10B981", 5},
		{"其他收入", "💵", "#10B981", 6},
		{"自定义输入", "✏️", "#6B7280", 999},
	}

	// Default expense categories - 更详细的分类
	expenseCategories := []struct {
		Name      string
		Icon      string
		Color     string
		SortOrder int
	}{
		{"餐饮美食", "🍽️", "#EF4444", 1},
		{"超市购物", "🛒", "#EF4444", 2},
		{"交通出行", "🚗", "#EF4444", 3},
		{"休闲娱乐", "🎮", "#EF4444", 4},
		{"房租房贷", "🏠", "#EF4444", 5},
		{"水电物业", "💡", "#EF4444", 6},
		{"医疗保健", "🏥", "#EF4444", 7},
		{"教育学习", "📚", "#EF4444", 8},
		{"人情往来", "🎁", "#EF4444", 9},
		{"运动健身", "🏃", "#EF4444", 10},
		{"美容护肤", "💄", "#EF4444", 11},
		{"服饰鞋包", "👔", "#EF4444", 12},
		{"通讯费用", "📱", "#EF4444", 13},
		{"其他支出", "📝", "#EF4444", 14},
		{"自定义输入", "✏️", "#6B7280", 999},
	}

	// Insert income categories
	for _, cat := range incomeCategories {
		_, err := DB.Exec(
			"INSERT INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES (?, ?, ?, ?, 1, 0, ?)",
			cat.Name, "income", cat.Icon, cat.Color, cat.SortOrder,
		)
		if err != nil {
			log.Println("Error seeding income categories:", err)
		}
	}

	// Insert expense categories
	for _, cat := range expenseCategories {
		_, err := DB.Exec(
			"INSERT INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES (?, ?, ?, ?, 1, 0, ?)",
			cat.Name, "expense", cat.Icon, cat.Color, cat.SortOrder,
		)
		if err != nil {
			log.Println("Error seeding expense categories:", err)
		}
	}
}

func seedSampleData() {
	// 不添加示例数据，保持数据库为空
	log.Println("数据库已初始化，无示例数据 - 等待用户输入")
}

func migrateDatabase() {
	// Check if badges table has user_id column
	var hasUserIDColumn bool
	err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		AND table_name = 'badges'
		AND column_name = 'user_id'
	`).Scan(&hasUserIDColumn)

	if err != nil {
		log.Printf("Error checking if badges table has user_id column: %v", err)
	} else if !hasUserIDColumn {
		log.Println("Adding user_id column to badges table...")
		// First, we need to drop existing badges table since we can't easily add a non-null foreign key to existing table
		// Note: This will delete all existing badges data
		_, err := DB.Exec("DROP TABLE IF EXISTS badges")
		if err != nil {
			log.Printf("Error dropping badges table: %v", err)
		} else {
			log.Println("Badges table dropped, will be recreated with user_id column")
		}
	}

	log.Println("Database migration completed for MySQL")
}

func verifyCategories() {
	// Check if categories table exists
	var tableExists bool
	err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
		AND table_name = 'categories'
	`).Scan(&tableExists)

	if err != nil {
		log.Println("Warning: Error checking if categories table exists:", err)
		return
	}

	if !tableExists {
		log.Println("Warning: categories table does not exist, skipping verification")
		return
	}

	// 检查分类数量
	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		log.Println("Error checking categories:", err)
		return
	}

	log.Printf("数据库已初始化，包含 %d 个分类", count)

	// 如果没有分类，强制重新初始化
	if count == 0 {
		log.Println("No categories found, reinitializing...")
		seedCategories()
	}
}

// ClearAllData clears all data from all tables
func ClearAllData() error {
	// Disable foreign key constraints temporarily
	_, err := DB.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		return err
	}
	defer DB.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Clear data from all tables
	tables := []string{
		"todo_checkins",
		"todos",
		"habit_logs",
		"habits",
		"transactions",
		"finance_goals",
		"diaries",
		"categories",
		"badges",
		"users",
	}

	for _, table := range tables {
		if table == "badges" {
			// Drop badges table instead of truncating to ensure new structure with user_id column
			_, err := DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
			if err != nil {
				log.Printf("Error dropping table %s: %v", table, err)
			}
		} else {
			_, err := DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
			if err != nil {
				log.Printf("Error truncating table %s: %v", table, err)
				// Continue with other tables even if one fails
			}
		}
	}

	log.Println("All data cleared successfully")
	return nil
}
