package handlers

import (
	"goblog/auth"
	"goblog/db"
	"goblog/models"
	"log"
	"net/http"
	"strconv"
	"time"
)

// HabitsHandler renders the habits page
func HabitsHandler(w http.ResponseWriter, r *http.Request) {
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
		Habits         []models.Habit
		Badges         []models.Badge
		TotalHabits    int
		DoneToday      int
		MaxStreak      int
		UnlockedBadges int
		TotalBadges    int
		User           *auth.Session
		IsLoggedIn     bool
	}{
		ActivePage: "habits",
		User:       session,
		IsLoggedIn: session != nil,
	}

	// Fetch Habits for current user
	rows, err := db.DB.Query("SELECT id, name, description, frequency, streak, total_days FROM habits WHERE user_id = ?", userID)
	if err != nil {
		log.Println(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var h models.Habit
			rows.Scan(&h.ID, &h.Name, &h.Description, &h.Frequency, &h.Streak, &h.TotalDays)

			// Check if habit is already checked today
			startOfDay := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
			var count int
			db.DB.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = ? AND date >= ?", h.ID, startOfDay).Scan(&count)
			h.TodayChecked = count > 0

			if h.TodayChecked {
				data.DoneToday++
			}
			if h.Streak > data.MaxStreak {
				data.MaxStreak = h.Streak
			}

			data.Habits = append(data.Habits, h)
			data.TotalHabits++
		}
	}

	// Fetch Badges for current user
	bRows, err := db.DB.Query("SELECT id, name, description, icon, unlocked FROM badges WHERE user_id = ?", userID)
	if err != nil {
		log.Println(err)
	} else {
		defer bRows.Close()
		for bRows.Next() {
			var b models.Badge
			var unlockedInt int
			bRows.Scan(&b.ID, &b.Name, &b.Description, &b.Icon, &unlockedInt)
			b.Unlocked = unlockedInt == 1
			data.Badges = append(data.Badges, b)
			data.TotalBadges++
			if b.Unlocked {
				data.UnlockedBadges++
			}
		}
	}

	renderTemplate(w, "habits.html", data)
}

// AddHabitHandler adds a new habit
func AddHabitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/habits", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	frequency := r.FormValue("frequency")

	_, err := db.DB.Exec("INSERT INTO habits (user_id, name, description, frequency) VALUES (?, ?, ?, ?)", userID, name, description, frequency)
	if err != nil {
		log.Println("Error adding habit:", err)
	}

	http.Redirect(w, r, "/habits", http.StatusSeeOther)
}

// DeleteHabitHandler deletes a habit
func DeleteHabitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/habits", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))

	// Delete logs first (foreign key)
	_, err := db.DB.Exec("DELETE FROM habit_logs WHERE habit_id = ?", id)
	if err != nil {
		log.Println("Error deleting habit logs:", err)
	}

	_, err = db.DB.Exec("DELETE FROM habits WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		log.Println("Error deleting habit:", err)
	}

	http.Redirect(w, r, "/habits", http.StatusSeeOther)
}

// CheckinHabitHandler handles habit check-ins
func CheckinHabitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/habits", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	habitID, _ := strconv.Atoi(r.FormValue("habit_id"))
	now := time.Now()

	// Check if already checked in today
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM habit_logs hl INNER JOIN habits h ON hl.habit_id = h.id WHERE hl.habit_id = ? AND hl.date >= ? AND h.user_id = ?", habitID, startOfDay, userID).Scan(&count)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/habits", http.StatusSeeOther)
		return
	}
	if count > 0 {
		http.Redirect(w, r, "/habits", http.StatusSeeOther)
		return
	}

	// Record Log
	_, err = db.DB.Exec("INSERT INTO habit_logs (habit_id, date) VALUES (?, ?)", habitID, now)
	if err != nil {
		log.Println("Error log habit:", err)
		http.Redirect(w, r, "/habits", http.StatusSeeOther)
		return
	}

	// Update Streak and Total
	yesterday := startOfDay.AddDate(0, 0, -1)
	var yesterdayCount int
	db.DB.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = ? AND date >= ? AND date < ?", habitID, yesterday, startOfDay).Scan(&yesterdayCount)

	var streak int
	var totalDays int

	db.DB.QueryRow("SELECT streak, total_days FROM habits WHERE id = ?", habitID).Scan(&streak, &totalDays)

	if yesterdayCount > 0 {
		streak++
	} else {
		streak = 1
	}
	totalDays++

	_, err = db.DB.Exec("UPDATE habits SET streak = ?, total_days = ? WHERE id = ?", streak, totalDays, habitID)
	if err != nil {
		log.Println("Error updating habit:", err)
	}

	// Check Badges for current user
	checkBadges(userID, totalDays, streak)

	http.Redirect(w, r, "/habits", http.StatusSeeOther)
}

func checkBadges(userID, totalDays, streak int) {
	rows, err := db.DB.Query("SELECT id, condition_days, unlocked FROM badges WHERE user_id = ? AND unlocked = 0", userID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, days int
		var unlocked int
		rows.Scan(&id, &days, &unlocked)

		if totalDays >= days || streak >= days {
			db.DB.Exec("UPDATE badges SET unlocked = 1 WHERE id = ? AND user_id = ?", id, userID)
		}
	}
}
