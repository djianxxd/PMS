package main

import (
	"fmt"
	"goblog/db"
	"goblog/handlers"
	"log"
	"net/http"
	"time"
)

func main() {
	// Initialize Database
	db.InitDB()
	defer db.DB.Close()

	// Test route
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
		<h2>数据库测试</h2>
		<a href="/test/insert">插入测试数据</a> | 
		<a href="/">查看仪表板</a>
		`)
	})

	http.HandleFunc("/test/insert", func(w http.ResponseWriter, r *http.Request) {
		// 插入测试数据
		now := time.Now()

		_, err := db.DB.Exec(
			"INSERT INTO transactions (type, category_id, category, amount, date, note) VALUES (?, ?, ?, ?, ?, ?)",
			"income", 1, "工资收入", 8000.00, now, "测试收入",
		)
		if err != nil {
			fmt.Fprintf(w, "插入收入失败: %v", err)
			return
		}

		_, err = db.DB.Exec(
			"INSERT INTO transactions (type, category_id, category, amount, date, note) VALUES (?, ?, ?, ?, ?, ?)",
			"expense", 1, "餐饮美食", 150.50, now, "测试支出",
		)
		if err != nil {
			fmt.Fprintf(w, "插入支出失败: %v", err)
			return
		}

		fmt.Fprintf(w, `
		<h2>✅ 测试数据插入成功！</h2>
		<p>收入: ¥8000.00</p>
		<p>支出: ¥150.50</p>
		<a href="/">查看仪表板</a>
		`)
	})

	// Routes
	http.HandleFunc("/", handlers.DashboardHandler)

	http.HandleFunc("/finance", handlers.FinanceHandler)
	http.HandleFunc("/finance/add", handlers.AddTransactionHandler)
	http.HandleFunc("/finance/delete", handlers.DeleteTransactionHandler)

	// Category management
	http.HandleFunc("/api/categories", handlers.GetCategoriesHandler)

	http.HandleFunc("/habits", handlers.HabitsHandler)
	http.HandleFunc("/habits/add", handlers.AddHabitHandler)
	http.HandleFunc("/habits/delete", handlers.DeleteHabitHandler)
	http.HandleFunc("/habits/checkin", handlers.CheckinHabitHandler)

	http.HandleFunc("/todos", handlers.TodosHandler)
	http.HandleFunc("/todos/add", handlers.AddTodoHandler)
	http.HandleFunc("/todos/toggle", handlers.ToggleTodoHandler)
	http.HandleFunc("/todos/delete", handlers.DeleteTodoHandler)

	http.HandleFunc("/export", handlers.ExportHandler)

	http.HandleFunc("/backup", handlers.BackupHandler)
	http.HandleFunc("/restore", handlers.RestoreHandler)
	http.HandleFunc("/backup/page", handlers.BackupPageHandler)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
