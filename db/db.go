package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // Import pure Go sqlite driver
)

var DB *sql.DB

func InitDB() {
	var err error

	// Ensure data directory exists
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		os.Mkdir("data", 0755)
	}

	dbPath := filepath.Join("data", "app.db")
	DB, err = sql.Open("sqlite", dbPath)

	if err != nil {
		log.Fatal(err)
	}

	createTables()
	migrateDatabase()
	seedBadges()
	seedCategories()
	seedSampleData()

	// 验证分类是否成功初始化
	verifyCategories()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			icon TEXT,
			color TEXT,
			is_default INTEGER DEFAULT 0,
			is_custom INTEGER DEFAULT 0,
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT,
			category_id INTEGER,
			category TEXT,
			amount REAL,
			date DATETIME,
			note TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(category_id) REFERENCES categories(id)
		);`,
		`CREATE TABLE IF NOT EXISTS finance_goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT,
			target_amount REAL,
			start_date DATETIME,
			end_date DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS habits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			frequency TEXT,
			streak INTEGER DEFAULT 0,
			total_days INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS habit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			habit_id INTEGER,
			date DATETIME,
			FOREIGN KEY(habit_id) REFERENCES habits(id)
		);`,
		`CREATE TABLE IF NOT EXISTS todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT,
			status TEXT DEFAULT 'pending',
			due_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS todo_checkins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			todo_id INTEGER,
			checkin_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(todo_id) REFERENCES todos(id)
		);`,
		`CREATE TABLE IF NOT EXISTS badges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			icon TEXT,
			unlocked INTEGER DEFAULT 0,
			condition_days INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS diaries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			content TEXT,
			weather TEXT,
			mood TEXT,
			date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Printf("Error creating table: %s, %v", query, err)
		}
	}
}

func seedBadges() {
	// Check if badges exist
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM badges").Scan(&count)
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
		_, err := DB.Exec("INSERT INTO badges (name, description, icon, condition_days) VALUES (?, ?, ?, ?)", b.Name, b.Description, b.Icon, b.Days)
		if err != nil {
			log.Println("Error seeding badges:", err)
		}
	}
}

func seedCategories() {
	// Check if categories exist
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
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
	log.Println("Database initialized without sample data - ready for user input")
}

func migrateDatabase() {
	// Add category_id column to transactions table if it doesn't exist
	var columnExists bool
	err := DB.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('transactions') 
		WHERE name = 'category_id'
	`).Scan(&columnExists)

	if err == nil && !columnExists {
		log.Println("Migrating database: adding category_id column to transactions table")
		_, err = DB.Exec("ALTER TABLE transactions ADD COLUMN category_id INTEGER")
		if err != nil {
			log.Println("Error adding category_id column:", err)
		}
	}
}

func verifyCategories() {
	// 检查分类数量
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		log.Println("Error checking categories:", err)
		return
	}

	log.Printf("Database initialized with %d categories", count)

	// 如果没有分类，强制重新初始化
	if count == 0 {
		log.Println("No categories found, reinitializing...")
		seedCategories()
	}
}
