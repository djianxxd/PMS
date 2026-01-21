package handlers

import (
	"database/sql"
	"fmt"
	"goblog/db"
	"goblog/models"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

func DiaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Get diaries from database
		rows, err := db.DB.Query(`
			SELECT id, title, content, weather, mood, date, created_at, updated_at 
			FROM diaries 
			ORDER BY date DESC, created_at DESC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var diaries []models.Diary
		for rows.Next() {
			var diary models.Diary
			err := rows.Scan(
				&diary.ID,
				&diary.Title,
				&diary.Content,
				&diary.Weather,
				&diary.Mood,
				&diary.Date,
				&diary.CreatedAt,
				&diary.UpdatedAt,
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			diaries = append(diaries, diary)
		}

		// Group diaries by month
		diaryGroups := make(map[string][]models.Diary)
		for _, diary := range diaries {
			monthKey := diary.Date.Format("2006年01月")
			diaryGroups[monthKey] = append(diaryGroups[monthKey], diary)
		}

		tmpl, err := template.ParseFiles("templates/layout.html", "templates/diary.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Calculate total diaries count
		totalCount := 0
		for _, diaries := range diaryGroups {
			totalCount += len(diaries)
		}

		data := struct {
			DiaryGroups   map[string][]models.Diary
			TotalCount    int
			MonthCount    int
			CurrentStreak int
			TodayMood     string
			ActivePage    string
		}{
			DiaryGroups:   diaryGroups,
			TotalCount:    totalCount,
			MonthCount:    len(diaryGroups),
			CurrentStreak: calculateCurrentStreak(diaries),
			TodayMood:     getTodayMood(diaries),
			ActivePage:    "diary",
		}

		err = tmpl.ExecuteTemplate(w, "layout", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// calculateCurrentStreak calculates the current streak of consecutive diary days
func calculateCurrentStreak(diaries []models.Diary) int {
	if len(diaries) == 0 {
		return 0
	}

	// Get unique dates and sort them
	dateMap := make(map[string]bool)
	for _, diary := range diaries {
		dateStr := diary.Date.Format("2006-01-02")
		dateMap[dateStr] = true
	}

	// Convert to slice and sort
	var dates []string
	for dateStr := range dateMap {
		dates = append(dates, dateStr)
	}

	// Sort dates
	for i := 0; i < len(dates); i++ {
		for j := i + 1; j < len(dates); j++ {
			if dates[i] < dates[j] {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}

	// Calculate current streak from today backwards
	today := time.Now().Format("2006-01-02")
	streak := 0

	// Check if we have a diary for today
	hasToday := false
	for _, dateStr := range dates {
		if dateStr == today {
			hasToday = true
			break
		}
	}

	if !hasToday && len(dates) > 0 {
		// Check if we have yesterday's diary
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		hasYesterday := false
		for _, dateStr := range dates {
			if dateStr == yesterday {
				hasYesterday = true
				break
			}
		}
		if !hasYesterday {
			return 0 // Streak is broken if we don't have today or yesterday
		}
	}

	// Calculate consecutive days
	currentDate := time.Now()
	if !hasToday {
		currentDate = currentDate.AddDate(0, 0, -1) // Start from yesterday if no today
	}

	for i := 0; i < len(dates)*2; i++ { // Check reasonable range
		dateStr := currentDate.Format("2006-01-02")

		if dateMap[dateStr] {
			streak++
			currentDate = currentDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	return streak
}

// getTodayMood returns the mood from today's diary or the most recent diary
func getTodayMood(diaries []models.Diary) string {
	if len(diaries) == 0 {
		return "😊" // Default mood
	}

	today := time.Now().Format("2006-01-02")

	// Look for today's diary
	for _, diary := range diaries {
		if diary.Date.Format("2006-01-02") == today && diary.Mood != "" {
			return diary.Mood
		}
	}

	// If no today's diary, return the most recent mood
	if len(diaries) > 0 && diaries[0].Mood != "" {
		return diaries[0].Mood
	}

	return "😊" // Default mood
}

func AddDiaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		content := r.FormValue("content")
		weather := r.FormValue("weather")
		mood := r.FormValue("mood")
		dateStr := r.FormValue("date")

		if title == "" || content == "" {
			http.Error(w, "标题和内容不能为空", http.StatusBadRequest)
			return
		}

		var date time.Time
		if dateStr != "" {
			parsed, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				http.Error(w, "日期格式错误", http.StatusBadRequest)
				return
			}
			date = parsed
		} else {
			date = time.Now()
		}

		_, err := db.DB.Exec(`
			INSERT INTO diaries (title, content, weather, mood, date, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, title, content, weather, mood, date, time.Now(), time.Now())

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/diary", http.StatusSeeOther)
	}
}

func DeleteDiaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "ID不能为空", http.StatusBadRequest)
			return
		}

		_, err := db.DB.Exec("DELETE FROM diaries WHERE id = ?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/diary", http.StatusSeeOther)
	}
}

func GetDiaryHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID不能为空", http.StatusBadRequest)
		return
	}

	var diary models.Diary
	err := db.DB.QueryRow(`
		SELECT id, title, content, weather, mood, date, created_at, updated_at 
		FROM diaries 
		WHERE id = ?
	`, id).Scan(
		&diary.ID,
		&diary.Title,
		&diary.Content,
		&diary.Weather,
		&diary.Mood,
		&diary.Date,
		&diary.CreatedAt,
		&diary.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "日记不存在", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check if this is an AJAX request for editing (expects JSON)
	ajaxHeader := r.Header.Get("X-Requested-With")
	if ajaxHeader == "XMLHttpRequest" || r.URL.Query().Get("format") == "json" {
		w.Header().Set("Content-Type", "application/json")
		// Simple JSON response
		jsonData := `{
			"id": ` + fmt.Sprintf("%d", diary.ID) + `,
			"title": "` + diary.Title + `",
			"content": "` + strings.ReplaceAll(diary.Content, `"`, `\"`) + `",
			"weather": "` + diary.Weather + `",
			"mood": "` + diary.Mood + `",
			"date": "` + diary.Date.Format("2006-01-02") + `"
		}`
		w.Write([]byte(jsonData))
		return
	}

	// Convert newlines to <br> for HTML display
	content := strings.ReplaceAll(diary.Content, "\n", "<br>")

	// Weather icons map
	weatherIcons := map[string]string{
		"sunny":  "☀️ 晴天",
		"cloudy": "☁️ 多云",
		"rainy":  "🌧️ 雨天",
		"snowy":  "❄️ 雪天",
		"windy":  "💨 大风",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
		<div class="p-6">
			<div class="mb-4">
				<h3 class="text-xl font-bold text-slate-800 mb-2">` + diary.Title + `</h3>
				<div class="flex items-center space-x-4 text-sm text-slate-500">
					<span><i class="fas fa-calendar-alt mr-1"></i>` + diary.Date.Format("2006-01-02") + `</span>
					<span>` + weatherIcons[diary.Weather] + `</span>
					<span class="text-2xl">` + diary.Mood + `</span>
				</div>
			</div>
			<div class="prose prose-sm max-w-none">
				<p class="text-slate-700 leading-relaxed">` + content + `</p>
			</div>
			<div class="mt-4 text-xs text-slate-400">
				创建时间: ` + diary.CreatedAt.Format("2006-01-02 15:04") + `
				` + func() string {
		if !diary.UpdatedAt.Equal(diary.CreatedAt) {
			return `<br>更新时间: ` + diary.UpdatedAt.Format("2006-01-02 15:04")
		}
		return ""
	}() + `
			</div>
		</div>
	`
	w.Write([]byte(html))
}

func UpdateDiaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		id := r.FormValue("id")
		title := r.FormValue("title")
		content := r.FormValue("content")
		weather := r.FormValue("weather")
		mood := r.FormValue("mood")
		dateStr := r.FormValue("date")

		if id == "" || title == "" || content == "" {
			http.Error(w, "ID、标题和内容不能为空", http.StatusBadRequest)
			return
		}

		var date time.Time
		if dateStr != "" {
			parsed, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				http.Error(w, "日期格式错误", http.StatusBadRequest)
				return
			}
			date = parsed
		} else {
			http.Error(w, "日期不能为空", http.StatusBadRequest)
			return
		}

		_, err := db.DB.Exec(`
			UPDATE diaries 
			SET title = ?, content = ?, weather = ?, mood = ?, date = ?, updated_at = ?
			WHERE id = ?
		`, title, content, weather, mood, date, time.Now(), id)

		if err != nil {
			log.Printf("Error updating diary: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/diary", http.StatusSeeOther)
	}
}
