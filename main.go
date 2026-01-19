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

	// Static Files
	// Assuming you might want to serve static assets if any, though currently using CDN
	// fs := http.FileServer(http.Dir("static"))
	// http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routes
	http.HandleFunc("/", handlers.DashboardHandler)

	http.HandleFunc("/finance", handlers.FinanceHandler)
	http.HandleFunc("/finance/add", handlers.AddTransactionHandler)

	http.HandleFunc("/habits", handlers.HabitsHandler)
	http.HandleFunc("/habits/add", handlers.AddHabitHandler)
	http.HandleFunc("/habits/checkin", handlers.CheckinHabitHandler)

	http.HandleFunc("/todos", handlers.TodosHandler)
	http.HandleFunc("/todos/add", handlers.AddTodoHandler)
	http.HandleFunc("/todos/toggle", handlers.ToggleTodoHandler)
	http.HandleFunc("/todos/delete", handlers.DeleteTodoHandler)

	http.HandleFunc("/export", handlers.ExportHandler)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
