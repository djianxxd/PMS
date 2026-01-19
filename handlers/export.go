package handlers

import (
	"encoding/json"
	"goblog/db"
	"goblog/models"
	"net/http"
)

// ExportHandler exports all data as JSON
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Transactions []models.Transaction
		Habits       []models.Habit
		Todos        []models.Todo
	}{}

	// Transactions
	rows, _ := db.DB.Query("SELECT id, type, amount, category, date, note FROM transactions")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t models.Transaction
			rows.Scan(&t.ID, &t.Type, &t.Amount, &t.Category, &t.Date, &t.Note)
			data.Transactions = append(data.Transactions, t)
		}
	}

	// Habits
	hRows, _ := db.DB.Query("SELECT id, name, description, frequency, streak, total_days FROM habits")
	if hRows != nil {
		defer hRows.Close()
		for hRows.Next() {
			var h models.Habit
			hRows.Scan(&h.ID, &h.Name, &h.Description, &h.Frequency, &h.Streak, &h.TotalDays)
			data.Habits = append(data.Habits, h)
		}
	}

	// Todos
	tRows, _ := db.DB.Query("SELECT id, content, status FROM todos")
	if tRows != nil {
		defer tRows.Close()
		for tRows.Next() {
			var t models.Todo
			tRows.Scan(&t.ID, &t.Content, &t.Status)
			data.Todos = append(data.Todos, t)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=life_data.json")
	json.NewEncoder(w).Encode(data)
}
