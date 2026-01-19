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

// TodosHandler renders the todos page
func TodosHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ActivePage    string
		Todos         []models.Todo
		TotalCount    int
		PendingCount  int
		DoneCount     int
		TotalCheckins int
	}{
		ActivePage: "todos",
	}

	rows, err := db.DB.Query("SELECT id, content, status, due_date FROM todos ORDER BY status DESC, due_date ASC")
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
			err := db.DB.QueryRow("SELECT COUNT(*), MAX(checkin_date) FROM todo_checkins WHERE todo_id = ?", t.ID).Scan(&totalCount, &lastCheckin)
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

	content := r.FormValue("content")
	dueDateStr := r.FormValue("due_date")

	log.Printf("📝 添加待办事项 - 内容: '%s', 截止时间: '%s'", content, dueDateStr)

	// 验证内容不为空
	if strings.TrimSpace(content) == "" {
		log.Printf("❌ 待办事项内容为空")
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
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
				log.Printf("✅ 日期解析成功: %s (格式: %s)", dueDate.Format("2006-01-02 15:04:05"), format)
				break
			}
		}

		if dueDateToInsert == nil {
			log.Printf("⚠️ 无法解析日期格式，将不设置截止时间: %s", dueDateStr)
		}
	} else {
		log.Printf("ℹ️ 未设置截止时间")
	}

	// 插入到数据库
	result, err := db.DB.Exec("INSERT INTO todos (content, due_date) VALUES (?, ?)", content, dueDateToInsert)
	if err != nil {
		log.Printf("❌ 插入待办事项失败: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取插入的ID进行验证
	if id, err := result.LastInsertId(); err == nil {
		log.Printf("✅ 成功插入待办事项，ID: %d", id)

		// 验证插入的数据
		var verifyContent string
		var verifyDueDate sql.NullTime
		err := db.DB.QueryRow("SELECT content, due_date FROM todos WHERE id = ?", id).Scan(&verifyContent, &verifyDueDate)
		if err == nil {
			if verifyDueDate.Valid {
				log.Printf("✅ 验证成功: 内容='%s', 截止时间=%s", verifyContent, verifyDueDate.Time.Format("2006-01-02 15:04:05"))
			} else {
				log.Printf("✅ 验证成功: 内容='%s', 无截止时间", verifyContent)
			}
		}
	}

	log.Printf("🔄 重定向到待办事项页面")
	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}

// ToggleTodoHandler toggles todo status
func ToggleTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))

	// Get current status
	var status string
	err := db.DB.QueryRow("SELECT status FROM todos WHERE id = ?", id).Scan(&status)
	if err != nil {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	newStatus := "completed"
	if status == "completed" {
		newStatus = "pending"
	}

	_, err = db.DB.Exec("UPDATE todos SET status = ? WHERE id = ?", newStatus, id)
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

	id, _ := strconv.Atoi(r.FormValue("id"))
	now := time.Now()

	// 直接记录打卡，不做每天限制
	_, err := db.DB.Exec("INSERT INTO todo_checkins (todo_id, checkin_date) VALUES (?, ?)", id, now)
	if err != nil {
		log.Printf("Error inserting checkin: %v", err)
	} else {
		log.Printf("✅ Successfully checked in todo %d at %s", id, now.Format("2006-01-02 15:04:05"))
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}

// TodoCheckinsHandler shows detailed checkin history for a todo
func TodoCheckinsHandler(w http.ResponseWriter, r *http.Request) {
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
	err = db.DB.QueryRow("SELECT id, content, status, due_date FROM todos WHERE id = ?", todoID).Scan(
		&data.Todo.ID, &data.Todo.Content, &data.Todo.Status, &data.Todo.DueDate)
	if err != nil {
		log.Printf("Error fetching todo: %v", err)
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	// 获取打卡记录
	rows, err := db.DB.Query("SELECT id, checkin_date FROM todo_checkins WHERE todo_id = ? ORDER BY checkin_date DESC LIMIT 50", todoID)
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

// DeleteTodoHandler deletes a todo
func DeleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))

	// 先删除相关的打卡记录
	_, err := db.DB.Exec("DELETE FROM todo_checkins WHERE todo_id = ?", id)
	if err != nil {
		log.Printf("Error deleting todo checkins: %v", err)
	}

	// 删除todo
	_, err = db.DB.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		log.Printf("Error deleting todo: %v", err)
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}
