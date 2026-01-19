package handlers

import (
	"encoding/json"
	"fmt"
	"goblog/db"
	"goblog/models"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// BackupData represents the complete backup structure
type BackupData struct {
	Version      string               `json:"version"`
	BackupDate   time.Time            `json:"backup_date"`
	Transactions []models.Transaction `json:"transactions"`
	Habits       []models.Habit       `json:"habits"`
	HabitLogs    []models.HabitLog    `json:"habit_logs"`
	Todos        []models.Todo        `json:"todos"`
	Badges       []models.Badge       `json:"badges"`
	FinanceGoals []models.FinanceGoal `json:"finance_goals"`
}

// BackupHandler creates a complete backup of all data
func BackupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := BackupData{
		Version:    "1.0",
		BackupDate: time.Now(),
	}

	// Export Transactions
	rows, err := db.DB.Query("SELECT id, type, amount, category, date, note, created_at FROM transactions")
	if err == nil && rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t models.Transaction
			err := rows.Scan(&t.ID, &t.Type, &t.Amount, &t.Category, &t.Date, &t.Note, &t.CreatedAt)
			if err == nil {
				data.Transactions = append(data.Transactions, t)
			}
		}
	}

	// Export Habits
	hRows, err := db.DB.Query("SELECT id, name, description, frequency, streak, total_days, created_at FROM habits")
	if err == nil && hRows != nil {
		defer hRows.Close()
		for hRows.Next() {
			var h models.Habit
			err := hRows.Scan(&h.ID, &h.Name, &h.Description, &h.Frequency, &h.Streak, &h.TotalDays, &h.CreatedAt)
			if err == nil {
				data.Habits = append(data.Habits, h)
			}
		}
	}

	// Export Habit Logs
	hlRows, err := db.DB.Query("SELECT id, habit_id, date FROM habit_logs")
	if err == nil && hlRows != nil {
		defer hlRows.Close()
		for hlRows.Next() {
			var hl models.HabitLog
			err := hlRows.Scan(&hl.ID, &hl.HabitID, &hl.Date)
			if err == nil {
				data.HabitLogs = append(data.HabitLogs, hl)
			}
		}
	}

	// Export Todos
	tRows, err := db.DB.Query("SELECT id, content, status, due_date, created_at FROM todos")
	if err == nil && tRows != nil {
		defer tRows.Close()
		for tRows.Next() {
			var t models.Todo
			err := tRows.Scan(&t.ID, &t.Content, &t.Status, &t.DueDate, &t.CreatedAt)
			if err == nil {
				data.Todos = append(data.Todos, t)
			}
		}
	}

	// Export Badges
	bRows, err := db.DB.Query("SELECT id, name, description, icon, unlocked, condition_days FROM badges")
	if err == nil && bRows != nil {
		defer bRows.Close()
		for bRows.Next() {
			var b models.Badge
			var unlockedInt int
			var conditionDays int
			err := bRows.Scan(&b.ID, &b.Name, &b.Description, &b.Icon, &unlockedInt, &conditionDays)
			if err == nil {
				b.Unlocked = unlockedInt == 1
				data.Badges = append(data.Badges, b)
			}
		}
	}

	// Export Finance Goals
	fgRows, err := db.DB.Query("SELECT id, type, target_amount, start_date, end_date FROM finance_goals")
	if err == nil && fgRows != nil {
		defer fgRows.Close()
		for fgRows.Next() {
			var fg models.FinanceGoal
			err := fgRows.Scan(&fg.ID, &fg.Type, &fg.TargetAmount, &fg.StartDate, &fg.EndDate)
			if err == nil {
				data.FinanceGoals = append(data.FinanceGoals, fg)
			}
		}
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("life_backup_%s.json", timestamp)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error creating backup", http.StatusInternalServerError)
		return
	}
}

// RestoreHandler restores data from a backup file
func RestoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit file size to 50MB
	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "Error retrieving backup file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if filepath.Ext(header.Filename) != ".json" {
		http.Error(w, "Invalid file type. Please upload a JSON backup file.", http.StatusBadRequest)
		return
	}

	// Read and parse backup data
	body, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading backup file", http.StatusInternalServerError)
		return
	}

	var backupData BackupData
	if err := json.Unmarshal(body, &backupData); err != nil {
		http.Error(w, "Error parsing backup file", http.StatusBadRequest)
		return
	}

	// Create a backup of current data before restoring
	backupCurrentData()

	// Clear existing data
	clearAllData()

	// Restore data
	restoreErrors := []string{}

	// Restore Transactions
	if err := restoreTransactions(backupData.Transactions); err != nil {
		restoreErrors = append(restoreErrors, fmt.Sprintf("Transactions: %v", err))
	}

	// Restore Habits
	if err := restoreHabits(backupData.Habits); err != nil {
		restoreErrors = append(restoreErrors, fmt.Sprintf("Habits: %v", err))
	}

	// Restore Habit Logs
	if err := restoreHabitLogs(backupData.HabitLogs); err != nil {
		restoreErrors = append(restoreErrors, fmt.Sprintf("Habit Logs: %v", err))
	}

	// Restore Todos
	if err := restoreTodos(backupData.Todos); err != nil {
		restoreErrors = append(restoreErrors, fmt.Sprintf("Todos: %v", err))
	}

	// Restore Badges
	if err := restoreBadges(backupData.Badges); err != nil {
		restoreErrors = append(restoreErrors, fmt.Sprintf("Badges: %v", err))
	}

	// Restore Finance Goals
	if err := restoreFinanceGoals(backupData.FinanceGoals); err != nil {
		restoreErrors = append(restoreErrors, fmt.Sprintf("Finance Goals: %v", err))
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if len(restoreErrors) > 0 {
		response := map[string]interface{}{
			"success": false,
			"errors":  restoreErrors,
		}
		json.NewEncoder(w).Encode(response)
	} else {
		response := map[string]interface{}{
			"success":     true,
			"message":     "Data restored successfully",
			"backup_date": backupData.BackupDate,
			"version":     backupData.Version,
			"restored": map[string]int{
				"transactions":  len(backupData.Transactions),
				"habits":        len(backupData.Habits),
				"habit_logs":    len(backupData.HabitLogs),
				"todos":         len(backupData.Todos),
				"badges":        len(backupData.Badges),
				"finance_goals": len(backupData.FinanceGoals),
			},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// backupCurrentData creates a backup of current data before restore
func backupCurrentData() {
	timestamp := time.Now().Format("20060102_150405")
	backupDir := "data/backups"

	// Ensure backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		os.MkdirAll(backupDir, 0755)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("pre_restore_%s.db", timestamp))

	// Copy current database file
	currentDB := "data/app.db"
	if _, err := os.Stat(currentDB); err == nil {
		copyFile(currentDB, backupFile)
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// clearAllData removes all existing data from tables
func clearAllData() {
	tables := []string{
		"habit_logs", "transactions", "todos", "finance_goals", "habits", "badges",
	}

	for _, table := range tables {
		db.DB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}
}

// restoreTransactions restores transaction data
func restoreTransactions(transactions []models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	stmt, err := db.DB.Prepare("INSERT INTO transactions (id, type, amount, category, date, note, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range transactions {
		_, err := stmt.Exec(t.ID, t.Type, t.Amount, t.Category, t.Date, t.Note, t.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// restoreHabits restores habit data
func restoreHabits(habits []models.Habit) error {
	if len(habits) == 0 {
		return nil
	}

	stmt, err := db.DB.Prepare("INSERT INTO habits (id, name, description, frequency, streak, total_days, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, h := range habits {
		_, err := stmt.Exec(h.ID, h.Name, h.Description, h.Frequency, h.Streak, h.TotalDays, h.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// restoreHabitLogs restores habit log data
func restoreHabitLogs(habitLogs []models.HabitLog) error {
	if len(habitLogs) == 0 {
		return nil
	}

	stmt, err := db.DB.Prepare("INSERT INTO habit_logs (id, habit_id, date) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, hl := range habitLogs {
		_, err := stmt.Exec(hl.ID, hl.HabitID, hl.Date)
		if err != nil {
			return err
		}
	}
	return nil
}

// restoreTodos restores todo data
func restoreTodos(todos []models.Todo) error {
	if len(todos) == 0 {
		return nil
	}

	stmt, err := db.DB.Prepare("INSERT INTO todos (id, content, status, due_date, created_at) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range todos {
		_, err := stmt.Exec(t.ID, t.Content, t.Status, t.DueDate, t.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// restoreBadges restores badge data
func restoreBadges(badges []models.Badge) error {
	if len(badges) == 0 {
		return nil
	}

	stmt, err := db.DB.Prepare("INSERT INTO badges (id, name, description, icon, unlocked, condition_days) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range badges {
		unlocked := 0
		if b.Unlocked {
			unlocked = 1
		}
		_, err := stmt.Exec(b.ID, b.Name, b.Description, b.Icon, unlocked, 0) // condition_days not in model
		if err != nil {
			return err
		}
	}
	return nil
}

// restoreFinanceGoals restores finance goal data
func restoreFinanceGoals(financeGoals []models.FinanceGoal) error {
	if len(financeGoals) == 0 {
		return nil
	}

	stmt, err := db.DB.Prepare("INSERT INTO finance_goals (id, type, target_amount, start_date, end_date) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, fg := range financeGoals {
		_, err := stmt.Exec(fg.ID, fg.Type, fg.TargetAmount, fg.StartDate, fg.EndDate)
		if err != nil {
			return err
		}
	}
	return nil
}
