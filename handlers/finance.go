package handlers

import (
	"goblog/db"
	"goblog/models"
	"log"
	"net/http"
	"strconv"
	"time"
)

// FinanceHandler renders the finance page
func FinanceHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ActivePage     string
		Transactions   []models.Transaction
		Goals          []models.FinanceGoal
		CalcPercentage func(float64, float64) float64
	}{
		ActivePage: "finance",
		CalcPercentage: func(curr, target float64) float64 {
			if target == 0 {
				return 0
			}
			p := (curr / target) * 100
			if p > 100 {
				return 100
			}
			return p
		},
	}

	// Fetch Transactions
	rows, err := db.DB.Query("SELECT id, type, amount, category, date, note FROM transactions ORDER BY date DESC LIMIT 50")
	if err != nil {
		log.Println(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var t models.Transaction
			rows.Scan(&t.ID, &t.Type, &t.Amount, &t.Category, &t.Date, &t.Note)
			data.Transactions = append(data.Transactions, t)
		}
	}

	// Fetch Goals
	gRows, err := db.DB.Query("SELECT id, type, target_amount, start_date, end_date FROM finance_goals")
	if err != nil {
		log.Println(err)
	} else {
		defer gRows.Close()
		for gRows.Next() {
			var g models.FinanceGoal
			gRows.Scan(&g.ID, &g.Type, &g.TargetAmount, &g.StartDate, &g.EndDate)

			var current float64
			err := db.DB.QueryRow("SELECT SUM(amount) FROM transactions WHERE type='expense' AND date >= ? AND date <= ?", g.StartDate, g.EndDate).Scan(&current)
			if err == nil {
				g.CurrentAmount = current
			}
			data.Goals = append(data.Goals, g)
		}
	}

	renderTemplate(w, "finance.html", data)
}

// AddTransactionHandler handles adding a new transaction
func AddTransactionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/finance", http.StatusSeeOther)
		return
	}

	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	category := r.FormValue("category")
	note := r.FormValue("note")
	tType := r.FormValue("type")
	date := time.Now()

	_, err := db.DB.Exec("INSERT INTO transactions (type, amount, category, date, note) VALUES (?, ?, ?, ?, ?)",
		tType, amount, category, date, note)
	if err != nil {
		log.Println("Error adding transaction:", err)
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
