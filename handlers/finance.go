package handlers

import (
	"database/sql"
	"goblog/db"
	"goblog/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FinanceHandler renders the finance page
func FinanceHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ActivePage     string
		Transactions   []models.Transaction
		Goals          []models.FinanceGoal
		Categories     []models.Category
		MonthlyIncome  float64
		MonthlyExpense float64
	}{
		ActivePage: "finance",
	}

	// Fetch Transactions
	rows, err := db.DB.Query("SELECT t.id, t.type, t.amount, t.category_id, t.category, t.date, t.note FROM transactions t ORDER BY date DESC LIMIT 50")
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

	// Fetch Goals
	gRows, err := db.DB.Query("SELECT id, type, target_amount, start_date, end_date FROM finance_goals")
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

			var current float64
			err = db.DB.QueryRow("SELECT SUM(amount) FROM transactions WHERE type='expense' AND date >= ? AND date <= ?", g.StartDate, g.EndDate).Scan(&current)
			if err == nil {
				g.CurrentAmount = current
			}
			data.Goals = append(data.Goals, g)
		}
	}

	// Fetch Categories for the form
	catRows, err := db.DB.Query("SELECT id, name, type, icon, color, is_default, is_custom, sort_order, created_at FROM categories ORDER BY type, sort_order ASC")
	if err != nil {
		log.Println("Error fetching categories:", err)
	} else {
		defer catRows.Close()
		for catRows.Next() {
			var cat models.Category
			var isDefault, isCustom int
			err := catRows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Icon, &cat.Color, &isDefault, &isCustom, &cat.SortOrder, &cat.CreatedAt)
			if err != nil {
				log.Println("Error scanning category:", err)
				continue
			}
			cat.IsDefault = isDefault == 1
			cat.IsCustom = isCustom == 1
			data.Categories = append(data.Categories, cat)
		}
	}

	// Calculate monthly statistics for finance page
	log.Printf("📊 计算收支管理页面统计")

	// 首先检查数据库中是否有交易记录
	var totalCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&totalCount)
	log.Printf("数据库总交易记录数: %d", totalCount)

	if totalCount > 0 {
		// 查询本月统计（使用与dashboard相同的逻辑）
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		log.Printf("查询本月统计，起始时间: %s", startOfMonth.Format("2006-01-02 15:04:05"))

		// 查询本月收入
		err := db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='income' AND date >= ?", startOfMonth).Scan(&data.MonthlyIncome)
		if err != nil {
			log.Printf("❌ 查询本月收入失败: %v", err)
			data.MonthlyIncome = 0
		} else {
			log.Printf("✅ 本月收入查询成功: ¥%.2f", data.MonthlyIncome)
		}

		// 查询本月支出
		err = db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='expense' AND date >= ?", startOfMonth).Scan(&data.MonthlyExpense)
		if err != nil {
			log.Printf("❌ 查询本月支出失败: %v", err)
			data.MonthlyExpense = 0
		} else {
			log.Printf("✅ 本月支出查询成功: ¥%.2f", data.MonthlyExpense)
		}

		// 如果本月没有数据，查询全部数据
		if data.MonthlyIncome == 0 && data.MonthlyExpense == 0 {
			log.Printf("⚠️ 本月无数据，查询全部数据")
			db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='income'").Scan(&data.MonthlyIncome)
			db.DB.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type='expense'").Scan(&data.MonthlyExpense)
			log.Printf("✅ 全部数据统计 - 收入:¥%.2f, 支出:¥%.2f", data.MonthlyIncome, data.MonthlyExpense)
		}
	} else {
		log.Printf("❌ 数据库中没有交易记录，保持显示0")
		data.MonthlyIncome = 0
		data.MonthlyExpense = 0
	}

	log.Printf("📈 收支管理页面最终统计: 本月收入=¥%.2f, 本月支出=¥%.2f", data.MonthlyIncome, data.MonthlyExpense)

	renderTemplate(w, "finance.html", data)
}

// AddTransactionHandler handles adding a new transaction
func AddTransactionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/finance", http.StatusSeeOther)
		return
	}

	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	categoryIDStr := r.FormValue("category_id")
	customCategory := r.FormValue("custom_category")
	note := r.FormValue("note")
	tType := r.FormValue("type")
	date := time.Now()

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
		// Create new custom category
		result, err := db.DB.Exec(
			"INSERT INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES (?, ?, ?, ?, 0, 1, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM categories WHERE type = ?))",
			category, tType, "🏷️", "#6B7280", tType,
		)
		if err == nil {
			if id, _ := result.LastInsertId(); id > 0 {
				categoryID = sql.NullInt64{Int64: id, Valid: true}
			}
		}
	}

	log.Printf("插入交易记录: type=%s, category=%s, amount=%.2f", tType, category, amount)

	log.Printf("插入交易记录: type=%s, category=%s, amount=%.2f, date=%s", tType, category, amount, date.Format("2006-01-02 15:04:05"))

	// 使用显式的SQL插入，确保所有字段都正确
	result, err := db.DB.Exec(
		"INSERT INTO transactions (type, category_id, category, amount, date, note, created_at) VALUES (?, ?, ?, ?, ?, ?, datetime('now'))",
		tType, categoryID, category, amount, date, note)
	if err != nil {
		log.Printf("Error adding transaction: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取插入的记录ID来验证
	lastID, err := result.LastInsertId()
	if err != nil {
		log.Printf("获取插入ID失败: %v", err)
	} else {
		log.Printf("✅ 成功插入交易记录，ID: %d", lastID)

		// 立即验证插入的数据
		var verifyType string
		var verifyAmount float64
		var verifyDate time.Time
		var verifyCategory string
		err := db.DB.QueryRow("SELECT type, category, amount, date FROM transactions WHERE id = ?", lastID).Scan(&verifyType, &verifyCategory, &verifyAmount, &verifyDate)
		if err != nil {
			log.Printf("❌ 验证插入记录失败: %v", err)
		} else {
			log.Printf("✅ 验证记录: type=%s, category=%s, amount=%.2f, date=%s",
				verifyType, verifyCategory, verifyAmount, verifyDate.Format("2006-01-02 15:04:05"))
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

	id, _ := strconv.Atoi(r.FormValue("id"))

	_, err := db.DB.Exec("DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		log.Println("Error deleting transaction:", err)
	}

	http.Redirect(w, r, "/finance", http.StatusSeeOther)
}
