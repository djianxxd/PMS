package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goblog/auth"
	"goblog/db"
	"goblog/models"
)

// AddTransactionHandler handles adding a new transaction
func AddTransactionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/finance", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		log.Printf("Failed to get user ID from context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	tType := r.FormValue("type")
	log.Printf("准备插入交易 - User ID: %d, Type: %s, Amount: %.2f", userID, tType, amount)
	categoryIDStr := r.FormValue("category_id")
	customCategory := r.FormValue("custom_category")
	note := r.FormValue("note")
	date := time.Now().Format("2006-01-02 15:04:05")

	var categoryID sql.NullInt64
	var category string

	// Handle category selection
	if categoryIDStr != "" && categoryIDStr != "custom" {
		// Existing category selected
		if id, err := strconv.Atoi(categoryIDStr); err == nil {
			var catName string
			err := db.DB.QueryRow("SELECT name FROM categories WHERE id = ?", id).Scan(&catName)
			if err == nil {
				categoryID = sql.NullInt64{Int64: int64(id), Valid: true}
				category = catName
			}
		}
	} else if customCategory != "" && strings.TrimSpace(customCategory) != "" {
		// Custom category entered
		category = strings.TrimSpace(customCategory)
	} else {
		// No category specified
		category = ""
	}

	// 验证内容不为空
	if strings.TrimSpace(tType) == "" {
		http.Error(w, "交易类型不能为空", http.StatusBadRequest)
		return
	}
	if amount <= 0 {
		http.Error(w, "交易金额必须大于0", http.StatusBadRequest)
		return
	}

	// 使用显式的SQL插入，确保所有字段都正确
	log.Printf("插入交易 - User ID: %d, Type: %s, Amount: %.2f, CategoryID: %v, Category: %s", userID, tType, amount, categoryID, category)

	result, err := db.DB.Exec("INSERT INTO transactions (user_id, type, category_id, category, amount, date, note, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, NOW())", userID, tType, categoryID, category, amount, date, note)
	if err != nil {
		log.Printf("Error adding transaction: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取插入的记录ID来验证
	lastID, _ := result.LastInsertId()
	if lastID > 0 {
		log.Printf("Transaction inserted with ID: %d", lastID)

		// 验证插入的记录
		var verifyType string
		var verifyAmount float64
		var verifyCategory string
		var verifyDate time.Time
		err := db.DB.QueryRow("SELECT type, category, amount, date FROM transactions WHERE id = ?", lastID).Scan(&verifyType, &verifyCategory, &verifyAmount, &verifyDate)
		if err != nil {
			log.Printf("Error verifying transaction: %v", err)
		} else {
			log.Printf("Verification successful - Type: %s, Amount: %.2f, Category: %s", verifyType, verifyAmount, verifyCategory)
		}
	}

	http.Redirect(w, r, "/finance", http.StatusSeeOther)
}

// DeleteTransactionHandler handles deleting a transaction
func DeleteTransactionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/finance", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))

	_, err := db.DB.Exec("DELETE FROM transactions WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		log.Println("Error deleting transaction:", err)
	}

	http.Redirect(w, r, "/finance", http.StatusSeeOther)
}

// FinanceHandler renders the finance page
func FinanceHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user session for display
	session, _ := auth.ValidateSession(r)

	data := struct {
		ActivePage     string
		Transactions   []models.Transaction
		Goals          []models.FinanceGoal
		Categories     []models.Category
		MonthlyIncome  float64
		MonthlyExpense float64
		User           *auth.Session
		IsLoggedIn     bool
	}{
		ActivePage: "finance",
		User:       session,
		IsLoggedIn: session != nil,
	}

	// Fetch Transactions for current user
	rows, err := db.DB.Query("SELECT t.id, t.type, t.amount, t.category_id, t.category, t.date, t.note FROM transactions t WHERE t.user_id = ? ORDER BY date DESC LIMIT 50", userID)
	if err != nil {
		log.Println("Error fetching transactions:", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var t models.Transaction
			var categoryID sql.NullInt64
			err := rows.Scan(&t.ID, &t.Type, &t.Amount, &categoryID, &t.Category, &t.Date, &t.Note)
			if err != nil {
				log.Println("Error scanning transaction:", err)
				continue
			}

			// If category_id exists but category is empty, fetch category name
			if categoryID.Valid && t.Category == "" {
				err := db.DB.QueryRow("SELECT name FROM categories WHERE id = ?", categoryID.Int64).Scan(&t.Category)
				if err != nil {
					log.Printf("Error fetching category name for ID %d: %v", categoryID.Int64, err)
					t.Category = "未知分类"
				}
			}

			data.Transactions = append(data.Transactions, t)
		}
	}

	// Fetch Goals for current user
	gRows, err := db.DB.Query("SELECT id, type, target_amount, start_date, end_date FROM finance_goals WHERE user_id = ?", userID)
	if err != nil {
		log.Println("Error fetching goals:", err)
	} else {
		defer gRows.Close()
		for gRows.Next() {
			var g models.FinanceGoal
			err := gRows.Scan(&g.ID, &g.Type, &g.TargetAmount, &g.StartDate, &g.EndDate)
			if err != nil {
				log.Println("Error scanning goal:", err)
				continue
			}

			data.Goals = append(data.Goals, g)
		}
	}

	// Fetch Categories (all users can see the same categories)
	rows, err = db.DB.Query("SELECT id, name, type, icon, color, is_default, is_custom, sort_order FROM categories ORDER BY type, sort_order ASC, id ASC")
	if err != nil {
		log.Println("Error fetching categories:", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var c models.Category
			err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Icon, &c.Color, &c.IsDefault, &c.IsCustom, &c.SortOrder)
			if err != nil {
				log.Println("Error scanning category:", err)
				continue
			}
			data.Categories = append(data.Categories, c)
		}
	}

	// Calculate Monthly Income/Expense
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// 查询本月收入和支出
	err = db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='income' AND date >= ? AND user_id = ?", startOfMonth, userID).Scan(&data.MonthlyIncome)
	if err != nil {
		log.Println("Error calculating monthly income:", err)
		data.MonthlyIncome = 0
	}

	err = db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='expense' AND date >= ? AND user_id = ?", startOfMonth, userID).Scan(&data.MonthlyExpense)
	if err != nil {
		log.Println("Error calculating monthly expense:", err)
		data.MonthlyExpense = 0
	}

	renderTemplate(w, "finance.html", data)
}
