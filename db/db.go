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
	seedBadges()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT,
			amount REAL,
			category TEXT,
			date DATETIME,
			note TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
		`CREATE TABLE IF NOT EXISTS badges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			description TEXT,
			icon TEXT,
			unlocked INTEGER DEFAULT 0,
			condition_days INTEGER
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
