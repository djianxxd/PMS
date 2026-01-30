package handlers

import (
	"database/sql"
	"encoding/json"
	"goblog/auth"
	"goblog/db"
	"goblog/models"
	"log"
	"net/http"
	"strings"
	"time"
)

func DiaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Get user ID from context
		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user session for display
		session, _ := auth.ValidateSession(r)

		// Get diaries from database for current user
		rows, err := db.DB.Query(`
			SELECT id, title, content, weather, mood, date, created_at, updated_at 
			FROM diaries 
			WHERE user_id = ?
			ORDER BY date DESC, created_at DESC
		`, userID)
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
			User          *auth.Session
			IsLoggedIn    bool
		}{
			DiaryGroups:   diaryGroups,
			TotalCount:    totalCount,
			MonthCount:    len(diaryGroups),
			CurrentStreak: calculateCurrentStreak(diaries),
			TodayMood:     getTodayMood(diaries),
			ActivePage:    "diary",
			User:          session,
			IsLoggedIn:    session != nil,
		}

		renderTemplate(w, "diary.html", data)
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
		// 确保正确解析UTF-8编码的表单数据
		if err := r.ParseForm(); err != nil {
			log.Printf("表单解析错误: %v", err)
			http.Error(w, "表单解析错误", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")
		weather := r.FormValue("weather")
		mood := r.FormValue("mood")
		dateStr := r.FormValue("date")

		log.Printf("收到日记数据 - 标题: %s, 内容长度: %d, 天气: %s, 心情: %s, 日期: %s",
			title, len(content), weather, mood, dateStr)

		if title == "" || content == "" {
			log.Printf("验证失败 - 标题或内容为空")
			http.Error(w, "标题和内容不能为空", http.StatusBadRequest)
			return
		}

		// 如果没有心情，使用默认值
		if mood == "" {
			mood = "😊"
			log.Printf("使用默认心情: %s", mood)
		}

		var date time.Time
		if dateStr != "" {
			parsed, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Printf("日期格式错误: %v", err)
				http.Error(w, "日期格式错误", http.StatusBadRequest)
				return
			}
			date = parsed
		} else {
			date = time.Now()
		}

		// Get user ID from context
		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		result, err := db.DB.Exec(`
		INSERT INTO diaries (user_id, title, content, weather, mood, date, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, title, content, weather, mood, date, time.Now(), time.Now())

		if err != nil {
			log.Printf("数据库插入错误: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		id, _ := result.LastInsertId()
		log.Printf("日记创建成功，ID: %d", id)

		http.Redirect(w, r, "/diary", http.StatusSeeOther)
	}
}

func DeleteDiaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get user ID from context
		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "ID不能为空", http.StatusBadRequest)
			return
		}

		_, err := db.DB.Exec("DELETE FROM diaries WHERE id = ? AND user_id = ?", id, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/diary", http.StatusSeeOther)
	}
}

func GetDiaryHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		log.Printf("获取日记失败：ID为空")
		http.Error(w, "ID不能为空", http.StatusBadRequest)
		return
	}

	log.Printf("获取日记，ID: %s", id)

	var diary models.Diary
	err := db.DB.QueryRow(`
		SELECT id, title, content, weather, mood, date, created_at, updated_at 
		FROM diaries 
		WHERE id = ? AND user_id = ?
	`, id, userID).Scan(
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
			log.Printf("日记不存在，ID: %s", id)
			http.Error(w, "日记不存在", http.StatusNotFound)
		} else {
			log.Printf("数据库查询错误: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	log.Printf("找到日记 - 标题: %s, 内容长度: %d", diary.Title, len(diary.Content))

	// Check if this is an AJAX request for editing (expects JSON)
	ajaxHeader := r.Header.Get("X-Requested-With")
	isJsonRequest := ajaxHeader == "XMLHttpRequest" || r.URL.Query().Get("format") == "json"

	if isJsonRequest {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		// Use proper JSON encoding
		response := map[string]interface{}{
			"id":      diary.ID,
			"title":   diary.Title,
			"content": diary.Content,
			"weather": diary.Weather,
			"mood":    diary.Mood,
			"date":    diary.Date.Format("2006-01-02"),
		}
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		err = encoder.Encode(response)
		if err != nil {
			log.Printf("JSON编码错误: %v", err)
		}
		return
	}

	// Convert newlines to <br> for HTML display
	content := strings.ReplaceAll(diary.Content, "\n", "<br>")

	// 确保内容不为空
	if content == "" {
		content = "<em class='text-slate-400'>暂无内容</em>"
	}

	// Weather icons map with fallback
	weatherIcons := map[string]string{
		"sunny":  "☀️ 晴天",
		"cloudy": "☁️ 多云",
		"rainy":  "🌧️ 雨天",
		"snowy":  "❄️ 雪天",
		"windy":  "💨 大风",
	}

	weatherDisplay := weatherIcons[diary.Weather]
	if weatherDisplay == "" {
		weatherDisplay = "🌤️ 其他"
	}

	moodDisplay := diary.Mood
	if moodDisplay == "" {
		moodDisplay = "😊"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
		<div class="p-6">
			<div class="mb-4">
				<h3 class="text-xl font-bold text-slate-800 mb-2">` + diary.Title + `</h3>
				<div class="flex items-center space-x-4 text-sm text-slate-500">
					<span><i class="fas fa-calendar-alt mr-1"></i>` + diary.Date.Format("2006-01-02") + `</span>
					<span>` + weatherDisplay + `</span>
					<span class="text-2xl">` + moodDisplay + `</span>
				</div>
			</div>
			<div class="prose prose-sm max-w-none">
				<div class="text-slate-700 leading-relaxed whitespace-pre-wrap">` + content + `</div>
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

	log.Printf("生成HTML内容长度: %d", len(html))
	w.Write([]byte(html))
}

func UpdateDiaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Get user ID from context
		userID, ok := GetUserIDFromContext(r)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 确保正确解析UTF-8编码的表单数据
		if err := r.ParseForm(); err != nil {
			log.Printf("更新日记表单解析错误: %v", err)
			http.Error(w, "表单解析错误", http.StatusBadRequest)
			return
		}

		id := r.FormValue("id")
		title := r.FormValue("title")
		content := r.FormValue("content")
		weather := r.FormValue("weather")
		mood := r.FormValue("mood")
		dateStr := r.FormValue("date")

		log.Printf("更新日记数据 - ID: %s, 标题: %s, 内容长度: %d, 天气: %s, 心情: %s, 日期: %s",
			id, title, len(content), weather, mood, dateStr)

		if id == "" || title == "" || content == "" {
			log.Printf("更新验证失败 - ID、标题或内容为空")
			http.Error(w, "ID、标题和内容不能为空", http.StatusBadRequest)
			return
		}

		// 如果没有心情，使用默认值
		if mood == "" {
			mood = "😊"
			log.Printf("更新使用默认心情: %s", mood)
		}

		var date time.Time
		if dateStr != "" {
			parsed, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Printf("更新日期格式错误: %v", err)
				http.Error(w, "日期格式错误", http.StatusBadRequest)
				return
			}
			date = parsed
		} else {
			log.Printf("更新日期为空")
			http.Error(w, "日期不能为空", http.StatusBadRequest)
			return
		}

		result, err := db.DB.Exec(`
			UPDATE diaries 
			SET title = ?, content = ?, weather = ?, mood = ?, date = ?, updated_at = ?
			WHERE id = ? AND user_id = ?
		`, title, content, weather, mood, date, time.Now(), id, userID)

		if err != nil {
			log.Printf("数据库更新错误: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		log.Printf("日记更新成功，影响行数: %d", rowsAffected)

		http.Redirect(w, r, "/diary", http.StatusSeeOther)
	}
}
