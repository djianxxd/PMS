package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"goblog/auth"
	"goblog/config"
	"goblog/db"
	"html/template"
	"log"
	"mime/multipart"
	"net/http"
	"time"
)

// AdminAuthMiddleware 管理后台认证中间件
func AdminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := auth.ValidateSession(r)
		if err != nil {
			// 未登录，跳转到登录页面
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// 检查是否是管理员
		if session.Username != "admin" {
			// 不是管理员，拒绝访问
			http.Error(w, "无权访问管理后台", http.StatusForbidden)
			return
		}

		// 是管理员，继续处理
		next.ServeHTTP(w, r)
	}
}

// renderAdminTemplate 渲染管理后台模板
func renderAdminTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles("templates/" + tmpl)
	if err != nil {
		log.Printf("Error parsing admin template %s: %v", tmpl, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		log.Printf("Error executing admin template %s: %v", tmpl, err)
	}
}

// AdminDashboardHandler 管理后台仪表盘
func AdminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	// 获取统计数据
	stats, err := getAdminStats()
	if err != nil {
		log.Printf("获取统计数据失败: %v", err)
		// 使用默认值
		stats = getDefaultStats()
	}

	// 添加服务器信息
	stats["ServerPort"] = config.AppConfig.Server.Port
	stats["DatabaseStatus"] = "正常"
	stats["MySQLVersion"] = "5.7+"
	stats["Uptime"] = "刚刚启动"

	renderAdminTemplate(w, "admin/dashboard.html", stats)
}

// AdminUsersHandler 用户管理页面
func AdminUsersHandler(w http.ResponseWriter, r *http.Request) {
	// 获取用户列表
	users, err := GetAllUsers()
	if err != nil {
		log.Printf("获取用户列表失败: %v", err)
		users = []map[string]interface{}{}
	}

	data := map[string]interface{}{
		"Users": users,
	}

	renderAdminTemplate(w, "admin/users.html", data)
}

// getAdminStats 获取管理后台统计数据
func getAdminStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取用户数量
	var userCount int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("获取用户数量失败: %w", err)
	}
	stats["UserCount"] = userCount

	// 获取交易记录数量
	var transactionCount int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&transactionCount)
	if err != nil {
		return nil, fmt.Errorf("获取交易记录数量失败: %w", err)
	}
	stats["TransactionCount"] = transactionCount

	// 获取待办事项数量
	var todoCount int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM todos").Scan(&todoCount)
	if err != nil {
		return nil, fmt.Errorf("获取待办事项数量失败: %w", err)
	}
	stats["TodoCount"] = todoCount

	// 获取日记数量
	var diaryCount int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM diaries").Scan(&diaryCount)
	if err != nil {
		return nil, fmt.Errorf("获取日记数量失败: %w", err)
	}
	stats["DiaryCount"] = diaryCount

	return stats, nil
}

// getDefaultStats 获取默认统计数据
func getDefaultStats() map[string]interface{} {
	return map[string]interface{}{
		"UserCount":         0,
		"TransactionCount":  0,
		"TodoCount":         0,
		"DiaryCount":        0,
	}
}

// GetAllUsers 获取所有用户
func GetAllUsers() ([]map[string]interface{}, error) {
	rows, err := db.DB.Query("SELECT id, username, email, created_at FROM users ORDER BY id DESC")
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var username, email string
		var createdAt time.Time
		err := rows.Scan(&id, &username, &email, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("扫描用户失败: %w", err)
		}
		
		user := map[string]interface{}{
			"ID":        id,
			"Username":  username,
			"Email":     email,
			"CreatedAt": createdAt.Format("2006-01-02 15:04:05"),
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历用户失败: %w", err)
	}

	return users, nil
}

// AdminEditUserHandler 处理用户编辑
func AdminEditUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取表单数据
	userIDStr := r.FormValue("user_id")
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// 转换用户ID
	userID := 0
	_, err := fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil || userID <= 0 {
		http.Error(w, "无效的用户ID", http.StatusBadRequest)
		return
	}

	// 验证数据
	if username == "" || email == "" {
		http.Error(w, "用户名和邮箱不能为空", http.StatusBadRequest)
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// 更新用户信息
	var result sql.Result
	if password != "" {
		// 需要更新密码
		hashedPassword, err := auth.HashPassword(password)
		if err != nil {
			log.Printf("密码加密失败: %v", err)
			http.Error(w, "内部服务器错误", http.StatusInternalServerError)
			return
		}

		// 更新所有字段
		result, err = tx.Exec(
			"UPDATE users SET username = ?, email = ?, password = ? WHERE id = ?",
			username, email, hashedPassword, userID,
		)
	} else {
		// 只更新用户名和邮箱
		result, err = tx.Exec(
			"UPDATE users SET username = ?, email = ? WHERE id = ?",
			username, email, userID,
		)
	}

	if err != nil {
		log.Printf("更新用户失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 检查是否更新成功
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("获取影响行数失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "用户不存在", http.StatusNotFound)
		return
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 重定向回用户列表
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// AdminDataHandler 数据管理页面
func AdminDataHandler(w http.ResponseWriter, r *http.Request) {
	renderAdminTemplate(w, "admin/data.html", nil)
}

// AdminExportDataHandler 处理数据导出
func AdminExportDataHandler(w http.ResponseWriter, r *http.Request) {
	// 获取导出类型
	exportType := r.URL.Query().Get("type")
	if exportType == "" {
		exportType = "all"
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=goblog_export_%s_%s.json", exportType, time.Now().Format("20060102_150405")))

	// 导出数据
	data, err := exportDatabaseData(exportType)
	if err != nil {
		log.Printf("导出数据失败: %v", err)
		http.Error(w, "导出数据失败", http.StatusInternalServerError)
		return
	}

	// 返回JSON数据
	json.NewEncoder(w).Encode(data)
}

// AdminImportDataHandler 处理数据导入
func AdminImportDataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析表单
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		http.Error(w, "文件过大", http.StatusBadRequest)
		return
	}

	// 获取文件
	file, _, err := r.FormFile("data_file")
	if err != nil {
		http.Error(w, "获取文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 获取导入选项
	importOption := r.FormValue("import_option")
	if importOption == "" {
		importOption = "replace"
	}

	// 导入数据
	err = importDatabaseData(file, importOption)
	if err != nil {
		log.Printf("导入数据失败: %v", err)
		http.Error(w, "导入数据失败", http.StatusInternalServerError)
		return
	}

	// 重定向回数据管理页面
	http.Redirect(w, r, "/admin/data", http.StatusSeeOther)
}

// exportDatabaseData 导出数据库数据
func exportDatabaseData(exportType string) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	data["export_type"] = exportType
	data["export_time"] = time.Now().Format("2006-01-02 15:04:05")
	data["version"] = "1.0"

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 根据导出类型导出数据
	switch exportType {
	case "all":
		// 导出所有数据
		if err := exportAllData(tx, data); err != nil {
			return nil, err
		}
	case "users":
		// 只导出用户数据
		if err := exportUserData(tx, data); err != nil {
			return nil, err
		}
	case "finance":
		// 只导出财务数据
		if err := exportFinanceData(tx, data); err != nil {
			return nil, err
		}
	case "habits":
		// 只导出习惯数据
		if err := exportHabitsData(tx, data); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("无效的导出类型: %s", exportType)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return data, nil
}

// importDatabaseData 导入数据库数据
func importDatabaseData(file multipart.File, importOption string) error {
	// 解析JSON文件
	var importData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&importData); err != nil {
		return fmt.Errorf("解析数据文件失败: %w", err)
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 根据导入选项处理数据
	if importOption == "replace" {
		// 清空现有数据
		if err := clearDatabaseData(tx); err != nil {
			return err
		}
	}

	// 导入数据
	if err := importDataToDatabase(tx, importData); err != nil {
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// 导出辅助函数
func exportAllData(tx *sql.Tx, data map[string]interface{}) error {
	if err := exportUserData(tx, data); err != nil {
		return err
	}
	if err := exportFinanceData(tx, data); err != nil {
		return err
	}
	if err := exportHabitsData(tx, data); err != nil {
		return err
	}
	if err := exportTodosData(tx, data); err != nil {
		return err
	}
	if err := exportDiaryData(tx, data); err != nil {
		return err
	}
	return nil
}

func exportUserData(tx *sql.Tx, data map[string]interface{}) error {
	// 导出用户数据
	users, err := getAllUsersFromDB(tx)
	if err != nil {
		return err
	}
	data["users"] = users

	// 导出徽章数据
	badges, err := getAllBadgesFromDB(tx)
	if err != nil {
		return err
	}
	data["badges"] = badges

	return nil
}

func exportFinanceData(tx *sql.Tx, data map[string]interface{}) error {
	// 导出分类数据
	categories, err := getAllCategoriesFromDB(tx)
	if err != nil {
		return err
	}
	data["categories"] = categories

	// 导出交易数据
	transactions, err := getAllTransactionsFromDB(tx)
	if err != nil {
		return err
	}
	data["transactions"] = transactions

	return nil
}

func exportHabitsData(tx *sql.Tx, data map[string]interface{}) error {
	// 导出习惯数据
	habits, err := getAllHabitsFromDB(tx)
	if err != nil {
		return err
	}
	data["habits"] = habits

	// 导出习惯日志数据
	habitLogs, err := getAllHabitLogsFromDB(tx)
	if err != nil {
		return err
	}
	data["habit_logs"] = habitLogs

	return nil
}

func exportTodosData(tx *sql.Tx, data map[string]interface{}) error {
	// 导出待办事项数据
	todos, err := getAllTodosFromDB(tx)
	if err != nil {
		return err
	}
	data["todos"] = todos

	// 导出待办事项打卡数据
	todoCheckins, err := getAllTodoCheckinsFromDB(tx)
	if err != nil {
		return err
	}
	data["todo_checkins"] = todoCheckins

	return nil
}

func exportDiaryData(tx *sql.Tx, data map[string]interface{}) error {
	// 导出日记数据
	diaries, err := getAllDiariesFromDB(tx)
	if err != nil {
		return err
	}
	data["diaries"] = diaries

	return nil
}

// 导入辅助函数
func clearDatabaseData(tx *sql.Tx) error {
	// 按顺序删除数据
	tables := []string{"badges", "diaries", "todo_checkins", "todos", "habit_logs", "habits", "transactions", "users"}
	for _, table := range tables {
		_, err := tx.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("清空表 %s 失败: %w", table, err)
		}
	}
	return nil
}

// importDataToDatabase 导入数据库数据
func importDataToDatabase(tx *sql.Tx, data map[string]interface{}) error {
	// 导入用户数据
	if usersData, ok := data["users"].([]interface{}); ok {
		for _, userItem := range usersData {
			if userMap, ok := userItem.(map[string]interface{}); ok {
				username, _ := userMap["username"].(string)
				email, _ := userMap["email"].(string)
				password, _ := userMap["password"].(string)
				
				// 检查用户是否已存在
				var exists bool
				err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
				if err != nil {
					return fmt.Errorf("检查用户是否存在失败: %w", err)
				}
				
				if !exists {
					// 插入新用户
					_, err := tx.Exec(
						"INSERT INTO users (username, email, password, created_at) VALUES (?, ?, ?, ?)",
						username, email, password, time.Now(),
					)
					if err != nil {
						return fmt.Errorf("插入用户失败: %w", err)
					}
				}
			}
		}
	}
	
	// 导入徽章数据
	if badgesData, ok := data["badges"].([]interface{}); ok {
		for _, badgeItem := range badgesData {
			if badgeMap, ok := badgeItem.(map[string]interface{}); ok {
				userID, _ := badgeMap["user_id"].(float64)
				name, _ := badgeMap["name"].(string)
				description, _ := badgeMap["description"].(string)
				icon, _ := badgeMap["icon"].(string)
				unlocked, _ := badgeMap["unlocked"].(float64)
				conditionDays, _ := badgeMap["condition_days"].(float64)
				
				// 检查徽章是否已存在
				var exists bool
				err := tx.QueryRow(
					"SELECT EXISTS(SELECT 1 FROM badges WHERE user_id = ? AND name = ?)",
					int(userID), name,
				).Scan(&exists)
				if err != nil {
					return fmt.Errorf("检查徽章是否存在失败: %w", err)
				}
				
				if !exists {
					// 插入新徽章
					_, err := tx.Exec(
						"INSERT INTO badges (user_id, name, description, icon, unlocked, condition_days) VALUES (?, ?, ?, ?, ?, ?)",
						int(userID), name, description, icon, int(unlocked), int(conditionDays),
					)
					if err != nil {
						return fmt.Errorf("插入徽章失败: %w", err)
					}
				}
			}
		}
	}
	
	// 导入分类数据
	if categoriesData, ok := data["categories"].([]interface{}); ok {
		for _, categoryItem := range categoriesData {
			if categoryMap, ok := categoryItem.(map[string]interface{}); ok {
				name, _ := categoryMap["name"].(string)
				categoryType, _ := categoryMap["type"].(string)
				icon, _ := categoryMap["icon"].(string)
				color, _ := categoryMap["color"].(string)
				isDefault, _ := categoryMap["is_default"].(float64)
				isCustom, _ := categoryMap["is_custom"].(float64)
				sortOrder, _ := categoryMap["sort_order"].(float64)
				
				// 检查分类是否已存在
				var exists bool
				err := tx.QueryRow(
					"SELECT EXISTS(SELECT 1 FROM categories WHERE name = ? AND type = ?)",
					name, categoryType,
				).Scan(&exists)
				if err != nil {
					return fmt.Errorf("检查分类是否存在失败: %w", err)
				}
				
				if !exists {
					// 插入新分类
					_, err := tx.Exec(
						"INSERT INTO categories (name, type, icon, color, is_default, is_custom, sort_order, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
						name, categoryType, icon, color, int(isDefault), int(isCustom), int(sortOrder), time.Now(),
					)
					if err != nil {
						return fmt.Errorf("插入分类失败: %w", err)
					}
				}
			}
		}
	}
	
	// 导入交易数据
	if transactionsData, ok := data["transactions"].([]interface{}); ok {
		for _, transactionItem := range transactionsData {
			if transactionMap, ok := transactionItem.(map[string]interface{}); ok {
				userID, _ := transactionMap["user_id"].(float64)
				transactionType, _ := transactionMap["type"].(string)
				categoryID, _ := transactionMap["category_id"].(float64)
				category, _ := transactionMap["category"].(string)
				amount, _ := transactionMap["amount"].(float64)
				note, _ := transactionMap["note"].(string)
				
				// 插入交易记录
				_, err := tx.Exec(
					"INSERT INTO transactions (user_id, type, category_id, category, amount, date, note, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
					int(userID), transactionType, int(categoryID), category, amount, time.Now(), note, time.Now(),
				)
				if err != nil {
					return fmt.Errorf("插入交易记录失败: %w", err)
				}
			}
		}
	}
	
	// 导入习惯数据
	if habitsData, ok := data["habits"].([]interface{}); ok {
		for _, habitItem := range habitsData {
			if habitMap, ok := habitItem.(map[string]interface{}); ok {
				userID, _ := habitMap["user_id"].(float64)
				name, _ := habitMap["name"].(string)
				description, _ := habitMap["description"].(string)
				frequency, _ := habitMap["frequency"].(string)
				
				// 插入习惯
				_, err := tx.Exec(
					"INSERT INTO habits (user_id, name, description, frequency, streak, total_days, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
					int(userID), name, description, frequency, 0, 0, time.Now(),
				)
				if err != nil {
					return fmt.Errorf("插入习惯失败: %w", err)
				}
			}
		}
	}
	
	// 导入待办事项数据
	if todosData, ok := data["todos"].([]interface{}); ok {
		for _, todoItem := range todosData {
			if todoMap, ok := todoItem.(map[string]interface{}); ok {
				userID, _ := todoMap["user_id"].(float64)
				content, _ := todoMap["content"].(string)
				status, _ := todoMap["status"].(string)
				
				// 插入待办事项
				_, err := tx.Exec(
					"INSERT INTO todos (user_id, content, status, due_date, created_at) VALUES (?, ?, ?, ?, ?)",
					int(userID), content, status, time.Now(), time.Now(),
				)
				if err != nil {
					return fmt.Errorf("插入待办事项失败: %w", err)
				}
			}
		}
	}
	
	// 导入日记数据
	if diariesData, ok := data["diaries"].([]interface{}); ok {
		for _, diaryItem := range diariesData {
			if diaryMap, ok := diaryItem.(map[string]interface{}); ok {
				userID, _ := diaryMap["user_id"].(float64)
				title, _ := diaryMap["title"].(string)
				content, _ := diaryMap["content"].(string)
				weather, _ := diaryMap["weather"].(string)
				mood, _ := diaryMap["mood"].(string)
				
				// 插入日记
				_, err := tx.Exec(
					"INSERT INTO diaries (user_id, title, content, weather, mood, date, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
					int(userID), title, content, weather, mood, time.Now(), time.Now(), time.Now(),
				)
				if err != nil {
					return fmt.Errorf("插入日记失败: %w", err)
				}
			}
		}
	}
	
	return nil
}

// 数据库查询辅助函数
func getAllUsersFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, username, email, password, created_at FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var username, email, password string
		var createdAt time.Time
		if err := rows.Scan(&id, &username, &email, &password, &createdAt); err != nil {
			return nil, err
		}
		user := map[string]interface{}{
			"id":         id,
			"username":   username,
			"email":      email,
			"password":   password,
			"created_at": createdAt,
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func getAllBadgesFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, user_id, name, description, icon, unlocked, condition_days FROM badges")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var badges []map[string]interface{}
	for rows.Next() {
		var id, userID, unlocked, conditionDays int
		var name, description, icon string
		if err := rows.Scan(&id, &userID, &name, &description, &icon, &unlocked, &conditionDays); err != nil {
			return nil, err
		}
		badge := map[string]interface{}{
			"id":            id,
			"user_id":       userID,
			"name":          name,
			"description":   description,
			"icon":          icon,
			"unlocked":      unlocked,
			"condition_days": conditionDays,
		}
		badges = append(badges, badge)
	}
	return badges, rows.Err()
}

func getAllCategoriesFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, name, type, icon, color, is_default, is_custom, sort_order, created_at FROM categories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []map[string]interface{}
	for rows.Next() {
		var id, isDefault, isCustom, sortOrder int
		var name, categoryType, icon, color string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &categoryType, &icon, &color, &isDefault, &isCustom, &sortOrder, &createdAt); err != nil {
			return nil, err
		}
		category := map[string]interface{}{
			"id":         id,
			"name":       name,
			"type":       categoryType,
			"icon":       icon,
			"color":      color,
			"is_default": isDefault,
			"is_custom":  isCustom,
			"sort_order": sortOrder,
			"created_at": createdAt,
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func getAllTransactionsFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, user_id, type, category_id, category, amount, date, note, created_at FROM transactions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []map[string]interface{}
	for rows.Next() {
		var id, userID, categoryID int
		var transactionType, category, note string
		var amount float64
		var date, createdAt time.Time
		if err := rows.Scan(&id, &userID, &transactionType, &categoryID, &category, &amount, &date, &note, &createdAt); err != nil {
			return nil, err
		}
		transaction := map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"type":       transactionType,
			"category_id": categoryID,
			"category":   category,
			"amount":     amount,
			"date":       date,
			"note":       note,
			"created_at": createdAt,
		}
		transactions = append(transactions, transaction)
	}
	return transactions, rows.Err()
}

func getAllHabitsFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, user_id, name, description, frequency, streak, total_days, created_at FROM habits")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []map[string]interface{}
	for rows.Next() {
		var id, userID, streak, totalDays int
		var name, description, frequency string
		var createdAt time.Time
		if err := rows.Scan(&id, &userID, &name, &description, &frequency, &streak, &totalDays, &createdAt); err != nil {
			return nil, err
		}
		habit := map[string]interface{}{
			"id":          id,
			"user_id":     userID,
			"name":        name,
			"description": description,
			"frequency":   frequency,
			"streak":      streak,
			"total_days":  totalDays,
			"created_at":  createdAt,
		}
		habits = append(habits, habit)
	}
	return habits, rows.Err()
}

func getAllHabitLogsFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, habit_id, date FROM habit_logs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, habitID int
		var date time.Time
		if err := rows.Scan(&id, &habitID, &date); err != nil {
			return nil, err
		}
		log := map[string]interface{}{
			"id":       id,
			"habit_id": habitID,
			"date":     date,
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func getAllTodosFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, user_id, content, status, due_date, created_at FROM todos")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []map[string]interface{}
	for rows.Next() {
		var id, userID int
		var content, status string
		var dueDate, createdAt time.Time
		if err := rows.Scan(&id, &userID, &content, &status, &dueDate, &createdAt); err != nil {
			return nil, err
		}
		todo := map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"content":    content,
			"status":     status,
			"due_date":   dueDate,
			"created_at": createdAt,
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

func getAllTodoCheckinsFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, todo_id, checkin_date, created_at FROM todo_checkins")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkins []map[string]interface{}
	for rows.Next() {
		var id, todoID int
		var checkinDate, createdAt time.Time
		if err := rows.Scan(&id, &todoID, &checkinDate, &createdAt); err != nil {
			return nil, err
		}
		checkin := map[string]interface{}{
			"id":          id,
			"todo_id":     todoID,
			"checkin_date": checkinDate,
			"created_at":   createdAt,
		}
		checkins = append(checkins, checkin)
	}
	return checkins, rows.Err()
}

func getAllDiariesFromDB(tx *sql.Tx) ([]map[string]interface{}, error) {
	rows, err := tx.Query("SELECT id, user_id, title, content, weather, mood, date, created_at, updated_at FROM diaries")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diaries []map[string]interface{}
	for rows.Next() {
		var id, userID int
		var title, content, weather, mood string
		var date, createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &userID, &title, &content, &weather, &mood, &date, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		diary := map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"title":      title,
			"content":    content,
			"weather":    weather,
			"mood":       mood,
			"date":       date,
			"created_at": createdAt,
			"updated_at": updatedAt,
		}
		diaries = append(diaries, diary)
	}
	return diaries, rows.Err()
}

// AdminDeleteUserHandler 处理用户删除
func AdminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取用户ID
	userIDStr := r.FormValue("user_id")
	userID := 0
	_, err := fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil || userID <= 0 {
		http.Error(w, "无效的用户ID", http.StatusBadRequest)
		return
	}

	// 开始事务
	tx, err := db.DB.Begin()
	if err != nil {
		log.Printf("开始事务失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// 删除用户相关数据
	// 1. 删除用户的徽章
	_, err = tx.Exec("DELETE FROM badges WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("删除用户徽章失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 2. 删除用户的日记
	_, err = tx.Exec("DELETE FROM diaries WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("删除用户日记失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 3. 删除用户的待办事项
	_, err = tx.Exec("DELETE FROM todo_checkins WHERE todo_id IN (SELECT id FROM todos WHERE user_id = ?)", userID)
	if err != nil {
		log.Printf("删除用户待办事项记录失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM todos WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("删除用户待办事项失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 4. 删除用户的习惯
	_, err = tx.Exec("DELETE FROM habit_logs WHERE habit_id IN (SELECT id FROM habits WHERE user_id = ?)", userID)
	if err != nil {
		log.Printf("删除用户习惯记录失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM habits WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("删除用户习惯失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 5. 删除用户的交易记录
	_, err = tx.Exec("DELETE FROM transactions WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("删除用户交易记录失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 6. 最后删除用户本身
	result, err := tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		log.Printf("删除用户失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 检查是否删除成功
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("获取影响行数失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "用户不存在", http.StatusNotFound)
		return
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}

	// 重定向回用户列表
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}
