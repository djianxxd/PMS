package handlers

import (
	"database/sql"
	"goblog/db"
	"html/template"
	"log"
	"net/http"
	"time"
)

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles("templates/layout.html", "templates/"+tmpl)
	if err != nil {
		log.Printf("Error parsing template %s: %v", tmpl, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = t.ExecuteTemplate(w, "layout", data)
	if err != nil {
		log.Printf("Error executing template %s: %v", tmpl, err)
	}
}

// DashboardHandler handles the main dashboard
func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ActivePage         string
		MonthlyIncome      float64
		MonthlyExpense     float64
		MaxStreak          int
		TodoCompletionRate int
		ChartMonths        []string
		ChartIncome        []float64
		ChartExpense       []float64
		HabitDoneCount     int
		HabitMissedCount   int
	}{
		ActivePage: "dashboard",
	}

	// Calculate Monthly Income/Expense
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	rows, err := db.DB.Query("SELECT type, amount FROM transactions WHERE date >= ?", startOfMonth)
	if err != nil {
		log.Println(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var tType string
			var amount float64
			rows.Scan(&tType, &amount)
			if tType == "income" {
				data.MonthlyIncome += amount
			} else {
				data.MonthlyExpense += amount
			}
		}
	}

	// Max Streak
	db.DB.QueryRow("SELECT MAX(streak) FROM habits").Scan(&data.MaxStreak)

	// Todo Completion Rate
	var total, completed int
	db.DB.QueryRow("SELECT COUNT(*) FROM todos").Scan(&total)
	db.DB.QueryRow("SELECT COUNT(*) FROM todos WHERE status = 'completed'").Scan(&completed)
	if total > 0 {
		data.TodoCompletionRate = (completed * 100) / total
	}

	// Chart Data (Last 6 months)
	data.ChartMonths = make([]string, 6)
	data.ChartIncome = make([]float64, 6)
	data.ChartExpense = make([]float64, 6)

	// 中文月份名称
	chineseMonths := []string{"一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"}

	for i := 0; i < 6; i++ {
		month := now.AddDate(0, -5+i, 0)
		data.ChartMonths[i] = chineseMonths[month.Month()-1]

		mStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local)

		mEnd := mStart.AddDate(0, 1, 0)

		var inc, exp sql.NullFloat64
		db.DB.QueryRow("SELECT SUM(amount) FROM transactions WHERE type='income' AND date >= ? AND date < ?", mStart, mEnd).Scan(&inc)
		db.DB.QueryRow("SELECT SUM(amount) FROM transactions WHERE type='expense' AND date >= ? AND date < ?", mStart, mEnd).Scan(&exp)

		if inc.Valid {
			data.ChartIncome[i] = inc.Float64
		}
		if exp.Valid {
			data.ChartExpense[i] = exp.Float64
		}
	}

	// Habit Stats (Today)
	// Simple approximation: Habits done today vs Total habits
	var totalHabits int
	db.DB.QueryRow("SELECT COUNT(*) FROM habits").Scan(&totalHabits)

	var doneToday int
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	db.DB.QueryRow("SELECT COUNT(DISTINCT habit_id) FROM habit_logs WHERE date >= ?", startOfDay).Scan(&doneToday)

	data.HabitDoneCount = doneToday
	data.HabitMissedCount = totalHabits - doneToday
	if data.HabitMissedCount < 0 {
		data.HabitMissedCount = 0
	}

	renderTemplate(w, "dashboard.html", data)
}

// BackupPageHandler handles the backup/restore page
func BackupPageHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ActivePage string
	}{
		ActivePage: "backup",
	}
	renderTemplate(w, "backup.html", data)
}
