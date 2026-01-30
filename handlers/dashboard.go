package handlers

import (
	"database/sql"
	"goblog/auth"
	"goblog/db"
	"html/template"
	"log"
	"net/http"
	"time"
)

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	// 添加模板函数
	funcMap := template.FuncMap{
		"mul": func(a, b interface{}) float64 {
			var aVal, bVal float64
			switch v := a.(type) {
			case int:
				aVal = float64(v)
			case float64:
				aVal = v
			}
			switch v := b.(type) {
			case int:
				bVal = float64(v)
			case float64:
				bVal = v
			}
			return aVal * bVal
		},
		"add": func(a, b interface{}) int {
			var aVal, bVal int
			switch v := a.(type) {
			case int:
				aVal = v
			}
			switch v := b.(type) {
			case int:
				bVal = v
			}
			return aVal + bVal
		},
		"substr": func(s string, start, length int) string {
			if start < 0 {
				start = 0
			}
			if start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
	}

	t := template.New("").Funcs(funcMap)
	t, err := t.ParseFiles("templates/layout.html", "templates/"+tmpl)
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
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user session for display
	session, _ := auth.ValidateSession(r)

	data := struct {
		ActivePage         string
		MonthlyIncome      float64
		MonthlyExpense     float64
		MaxStreak          int
		TodoCompletionRate int
		ChartMonths        []string
		ChartIncome        []float64
		ChartExpense       []float64
		TotalCount         int
		HabitDoneCount     int
		HabitMissedCount   int
		User               *auth.Session
		IsLoggedIn         bool
	}{
		ActivePage: "dashboard",
		User:       session,
		IsLoggedIn: session != nil,
	}

	// Calculate Monthly Income/Expense (从实际数据计算，但由于数据库为空，结果会是0)
	now := time.Now()

	// 首先检查数据库中是否有任何交易记录
	var totalCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE user_id = ?", userID).Scan(&totalCount)

	if totalCount == 0 {

		data.MonthlyIncome = 0
		data.MonthlyExpense = 0
	} else {

		// 查询当前用户的记录
		err := db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='income' AND user_id = ?", userID).Scan(&data.MonthlyIncome)
		if err != nil {
			data.MonthlyIncome = 0
		}

		err = db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='expense' AND user_id = ?", userID).Scan(&data.MonthlyExpense)
		if err != nil {
			data.MonthlyExpense = 0
		}

		err = db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='expense'").Scan(&data.MonthlyExpense)
		if err != nil {

			data.MonthlyExpense = 0
		} else {

		}

		// 如果找到了数据，现在尝试按月份查询
		if data.MonthlyIncome > 0 || data.MonthlyExpense > 0 {

			startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)

			var monthlyIncome, monthlyExpense float64
			db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='income' AND date >= ? AND user_id = ?", startOfMonth, userID).Scan(&monthlyIncome)
			db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='expense' AND date >= ? AND user_id = ?", startOfMonth, userID).Scan(&monthlyExpense)

			// 如果本月有数据就用本月的，否则用总数据
			if monthlyIncome > 0 || monthlyExpense > 0 {
				data.MonthlyIncome = monthlyIncome
				data.MonthlyExpense = monthlyExpense

			} else {

			}
		}
	}

	// Max Streak (从当前用户数据计算)
	var maxStreak sql.NullInt64
	db.DB.QueryRow("SELECT MAX(streak) FROM habits WHERE user_id = ?", userID).Scan(&maxStreak)
	if maxStreak.Valid {
		data.MaxStreak = int(maxStreak.Int64)
	} else {
		data.MaxStreak = 0
	}

	// Todo Completion Rate for current user
	var total, completed int
	db.DB.QueryRow("SELECT COUNT(*) FROM todos WHERE user_id = ?", userID).Scan(&total)
	db.DB.QueryRow("SELECT COUNT(*) FROM todos WHERE status = 'completed' AND user_id = ?", userID).Scan(&completed)
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
		db.DB.QueryRow("SELECT SUM(amount) FROM transactions WHERE type='income' AND date >= ? AND date < ? AND user_id = ?", mStart, mEnd, userID).Scan(&inc)
		db.DB.QueryRow("SELECT SUM(amount) FROM transactions WHERE type='expense' AND date >= ? AND date < ? AND user_id = ?", mStart, mEnd, userID).Scan(&exp)

		if inc.Valid {
			data.ChartIncome[i] = inc.Float64
		}
		if exp.Valid {
			data.ChartExpense[i] = exp.Float64
		}
	}

	// Habit Stats (Today) for current user
	// Simple approximation: Habits done today vs Total habits
	var totalHabits int
	db.DB.QueryRow("SELECT COUNT(*) FROM habits WHERE user_id = ?", userID).Scan(&totalHabits)

	var doneToday int
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	db.DB.QueryRow("SELECT COUNT(DISTINCT hl.habit_id) FROM habit_logs hl INNER JOIN habits h ON hl.habit_id = h.id WHERE hl.date >= ? AND h.user_id = ?", startOfDay, userID).Scan(&doneToday)

	data.HabitDoneCount = doneToday
	data.HabitMissedCount = totalHabits - doneToday
	data.TotalCount = totalHabits
	if data.HabitMissedCount < 0 {
		data.HabitMissedCount = 0
	}

	renderTemplate(w, "dashboard.html", data)
}
