package handlers

import (
	"database/sql"
	"goblog/db"
	"goblog/models"
	"log"
	"net/http"
	"strconv"
	"time"
)

// TodosHandler renders the todos page
func TodosHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		ActivePage string
		Todos      []models.Todo
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
			data.Todos = append(data.Todos, t)
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

	var dueDate time.Time
	if dueDateStr != "" {
		dueDate, _ = time.Parse("2006-01-02T15:04", dueDateStr)
	}

	_, err := db.DB.Exec("INSERT INTO todos (content, due_date) VALUES (?, ?)", content, dueDate)
	if err != nil {
		log.Println("Error adding todo:", err)
	}

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

// DeleteTodoHandler deletes a todo
func DeleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/todos", http.StatusSeeOther)
		return
	}

	id, _ := strconv.Atoi(r.FormValue("id"))

	_, err := db.DB.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		log.Println("Error deleting todo:", err)
	}

	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}
