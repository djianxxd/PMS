package handlers

import (
	"database/sql"
	"goblog/auth"
	"goblog/db"
	"goblog/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TodosHandler renders todos page
func TodosHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user session
	session, _ := auth.ValidateSession(r)

	data := struct {
		ActivePage    string
		Todos         []models.Todo
		TotalCount    int
		PendingCount  int
		DoneCount     int
		TotalCheckins int
		User          *auth.Session
		IsLoggedIn    bool
	}{
		ActivePage: "todos",
		User:       session,
		IsLoggedIn: session != nil,
	}

	rows, err := db.DB.Query("SELECT id, content, status, due_date FROM todos WHERE user_id = ? ORDER BY status DESC, due_date ASC", userID)
	if err != nil {
		log.Println(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var t models.Todo
			var dueDate sql.NullTime
			rows.Scan(&t.ID, &t.Content, &t.Status, &dueDate)
			if dueDate.Valid {
				t.DueDate = dueDate.Time
			}

			// 获取总打卡次数和最近打卡时间
			var totalCount int
			var lastCheckin sql.NullTime
			err := db.DB.QueryRow("SELECT COUNT(*), MAX(checkin_date) FROM todo_checkins tc INNER JOIN todos t ON tc.todo_id = t.id WHERE tc.todo_id = ? AND t.user_id = ?", t.ID, userID).Scan(&totalCount, &lastCheckin)
			if err == nil {
				t.CheckinCount = totalCount
				if lastCheckin.Valid {
					t.LastCheckin = lastCheckin.Time
				}
				data.TotalCheckins += totalCount
			}

			data.Todos = append(data.Todos, t)
			data.TotalCount++
			if t.Status == "pending" {
				data.PendingCount++
			} else if t.Status == "completed" {
				data.DoneCount++
			}
		}
	}

	renderTemplate(w, "todos.html", data)
}

// AddTodoHandler adds a new todo
func AddTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	content := r.FormValue("content")
	dueDateStr := r.FormValue("due_date")

	log.Printf("📝 添加待办事项 - 内容: '%s', 截止时间: '%s'", content, dueDateStr)

	// 验证内容不为空
	if strings.TrimSpace(content) == "" {

		http.Error(w, "任务内容不能为空", http.StatusBadRequest)
		return
	}

	var dueDate time.Time
	var dueDateToInsert interface{} = nil // 使用nil来处理空日期

	if dueDateStr != "" {
		// 尝试多种日期格式解析
		formats := []string{
			"2006-01-02T15:04",    // HTML datetime-local 格式
			"2006-01-02 15:04:05", // 标准格式
			"2006-01-02T15:04:05", // 带秒的格式
			"2006-01-02",          // 只有日期
		}

		for _, format := range formats {
			if parsed, err := time.Parse(format, dueDateStr); err == nil {
				dueDate = parsed
				dueDateToInsert = parsed

				break
			}
		}

		if dueDateToInsert == nil {
			log.Printf("⚠️ 无法解析日期格式，将不设置截止时间: %s", dueDateStr)
		}
	} else {
		log.Printf("ℹ️ 未设置截止时间")
	}

	// 日期验证：如果设置了截止时间，必须是未来时间
	if dueDateToInsert != nil {
		now := time.Now()
		if dueDate.Before(now) {
			// 严格验证：截止时间不能早于当前时间

			http.Error(w, "截止时间必须是未来时间", http.StatusBadRequest)
			return
		} else {
			// 检查是否设置得过近（比如只差几秒，可能误操作）
			duration := dueDate.Sub(now)
			if duration < time.Minute {
				log.Printf("⚠️ 截止时间设置过近: 截止时间=%s, 当前时间=%s, 相差=%s", dueDate.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"), duration)
				http.Error(w, "截止时间设置过近，请选择一个合理的未来时间", http.StatusBadRequest)
				return
			} else {

			}
		}
	} else {
		log.Printf("ℹ️ 未设置截止时间")
	}

	// 插入到数据库
	_, err := db.DB.Exec("INSERT INTO todos (user_id, content, due_date) VALUES (?, ?, ?)", userID, content, dueDateToInsert)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 重定向到待办事项页面
	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}

// ToggleTodoHandler toggles todo status
func ToggleTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))

	// Get current status
	var status string
	err := db.DB.QueryRow("SELECT status FROM todos WHERE id = ? AND user_id = ?", id, userID).Scan(&status)
	if err != nil {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	newStatus := "completed"
	if status == "completed" {
		newStatus = "pending"
	}

	_, err = db.DB.Exec("UPDATE todos SET status = ? WHERE id = ? AND user_id = ?", newStatus, id, userID)
	if err != nil {
		log.Println("Error toggling todo:", err)
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}

// CheckinTodoHandler handles todo check-ins
func CheckinTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))
	now := time.Now()

	// Verify todo belongs to user
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM todos WHERE id = ? AND user_id = ?", id, userID).Scan(&count)
	if err != nil || count == 0 {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	// 直接记录打卡，不做每天限制
	_, err = db.DB.Exec("INSERT INTO todo_checkins (todo_id, checkin_date) VALUES (?, ?)", id, now)
	if err != nil {
		log.Printf("Error inserting checkin: %v", err)
	} else {

	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}

// DeleteTodoHandler deletes a todo
func DeleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))

	// 先删除相关的打卡记录（通过todo_id关联，确保是用户自己的）
	_, err := db.DB.Exec("DELETE tc FROM todo_checkins tc INNER JOIN todos t ON tc.todo_id = t.id WHERE tc.todo_id = ? AND t.user_id = ?", id, userID)
	if err != nil {
		log.Printf("Error deleting todo checkins: %v", err)
	}

	// 删除todo
	_, err = db.DB.Exec("DELETE FROM todos WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		log.Printf("Error deleting todo: %v", err)
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}

// TodoCheckinsHandler shows detailed checkin history for a todo
func TodoCheckinsHandler(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	todoID, err := strconv.Atoi(id)
	if err != nil {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	// Verify todo belongs to user
	var count int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM todos WHERE id = ? AND user_id = ?", todoID, userID).Scan(&count)
	if err != nil || count == 0 {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	data := struct {
		ActivePage string
		Todo       models.Todo
		Checkins   []struct {
			ID          int       `json:"id"`
			CheckinDate time.Time `json:"checkin_date"`
		}
	}{
		ActivePage: "todos",
	}

	// 获取todo信息
	err = db.DB.QueryRow("SELECT id, content, status, due_date FROM todos WHERE id = ? AND user_id = ?", todoID, userID).Scan(
		&data.Todo.ID, &data.Todo.Content, &data.Todo.Status, &data.Todo.DueDate)

	// 获取打卡记录
	rows, err := db.DB.Query("SELECT id, checkin_date FROM todo_checkins tc INNER JOIN todos t ON tc.todo_id = t.id WHERE tc.todo_id = ? AND t.user_id = ? ORDER BY checkin_date DESC LIMIT 50", todoID, userID)
	if err != nil {
		log.Printf("Error fetching checkins: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var checkin struct {
				ID          int       `json:"id"`
				CheckinDate time.Time `json:"checkin_date"`
			}
			rows.Scan(&checkin.ID, &checkin.CheckinDate)
			data.Checkins = append(data.Checkins, checkin)
		}
	}

	renderTemplate(w, "todo_checkins.html", data)
}
