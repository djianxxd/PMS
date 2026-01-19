package main

import (
	"fmt"
	"goblog/db"
	"goblog/handlers"
	"log"
	"net/http"
)

func main() {
	// Initialize Database
	db.InitDB()
	defer db.DB.Close()

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
