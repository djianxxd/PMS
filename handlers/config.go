package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"goblog/config"
	"goblog/db"
)

// ConfigHandler 处理配置页面
func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// 合并配置数据
		data := struct {
			MySQLHost     string
			MySQLPort     string
			MySQLUser     string
			MySQLPassword string
			MySQLDatabase string
			ServerPort    string
			AdminUsername string
			AdminPassword string
			Error         string
			Success       string
		}{
			MySQLHost:     config.AppConfig.MySQL.Host,
			MySQLPort:     config.AppConfig.MySQL.Port,
			MySQLUser:     config.AppConfig.MySQL.User,
			MySQLPassword: config.AppConfig.MySQL.Password,
			MySQLDatabase: config.AppConfig.MySQL.Database,
			ServerPort:    config.AppConfig.Server.Port,
			AdminUsername: config.AppConfig.Admin.Username,
			AdminPassword: config.AppConfig.Admin.Password,
		}
		
		// 直接解析和执行 config.html 模板
		t, err := template.ParseFiles("templates/config.html")
		if err != nil {
			log.Printf("Error parsing template config.html: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = t.Execute(w, data)
		if err != nil {
			log.Printf("Error executing template config.html: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	if r.Method == http.MethodPost {
		// 检查是否是测试请求
		if r.URL.Path == "/config/test" {
			testDatabaseConnection(w, r)
			return
		}

		// 读取表单数据
		mysqlHost := r.FormValue("mysql_host")
		mysqlPort := r.FormValue("mysql_port")
		mysqlUser := r.FormValue("mysql_user")
		mysqlPassword := r.FormValue("mysql_password")
		mysqlDatabase := r.FormValue("mysql_database")
		serverPort := r.FormValue("server_port")
		adminUsername := r.FormValue("admin_username")
		adminPassword := r.FormValue("admin_password")

		// 更新配置
		config.AppConfig.MySQL.Host = mysqlHost
		config.AppConfig.MySQL.Port = mysqlPort
		config.AppConfig.MySQL.User = mysqlUser
		config.AppConfig.MySQL.Password = mysqlPassword
		config.AppConfig.MySQL.Database = mysqlDatabase
		config.AppConfig.Server.Port = serverPort
		config.AppConfig.Admin.Username = adminUsername
		config.AppConfig.Admin.Password = adminPassword

		// 测试数据库连接
		success, created, err := testDatabaseConnectionInternal(mysqlHost, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase)
		if !success {
			// 合并配置数据
			data := struct {
				MySQLHost     string
				MySQLPort     string
				MySQLUser     string
				MySQLPassword string
				MySQLDatabase string
				ServerPort    string
				AdminUsername string
				AdminPassword string
				Error         string
				Success       string
			}{
				MySQLHost:     mysqlHost,
				MySQLPort:     mysqlPort,
				MySQLUser:     mysqlUser,
				MySQLPassword: mysqlPassword,
				MySQLDatabase: mysqlDatabase,
				ServerPort:    serverPort,
				AdminUsername: adminUsername,
				AdminPassword: adminPassword,
				Error:         fmt.Sprintf("数据库连接失败: %v", err),
			}
			
			// 直接解析和执行 config.html 模板
			t, err := template.ParseFiles("templates/config.html")
			if err != nil {
				log.Printf("Error parsing template config.html: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			err = t.Execute(w, data)
			if err != nil {
				log.Printf("Error executing template config.html: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		}

		// 记录数据库创建状态
		if created {
			log.Printf("数据库 %s 不存在，已自动创建", mysqlDatabase)
		}

		// 保存配置
		err = config.SaveConfig()
		if err != nil {
			// 合并配置数据
			data := struct {
				MySQLHost     string
				MySQLPort     string
				MySQLUser     string
				MySQLPassword string
				MySQLDatabase string
				ServerPort    string
				AdminUsername string
				AdminPassword string
				Error         string
				Success       string
			}{
				MySQLHost:     mysqlHost,
				MySQLPort:     mysqlPort,
				MySQLUser:     mysqlUser,
				MySQLPassword: mysqlPassword,
				MySQLDatabase: mysqlDatabase,
				ServerPort:    serverPort,
				AdminUsername: adminUsername,
				AdminPassword: adminPassword,
				Error:         fmt.Sprintf("保存配置失败: %v", err),
			}
			
			// 直接解析和执行 config.html 模板
			t, err := template.ParseFiles("templates/config.html")
			if err != nil {
				log.Printf("Error parsing template config.html: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			err = t.Execute(w, data)
			if err != nil {
				log.Printf("Error executing template config.html: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		}

		// 标记为已初始化
		config.SetInitialized()
		err = config.SaveConfig()
		if err != nil {
			log.Printf("保存初始化状态失败: %v", err)
		}

		// 初始化数据库
		err = db.InitDB()
		if err != nil {
			log.Printf("数据库初始化失败: %v", err)
			// 合并配置数据
			data := struct {
				MySQLHost     string
				MySQLPort     string
				MySQLUser     string
				MySQLPassword string
				MySQLDatabase string
				ServerPort    string
				AdminUsername string
				AdminPassword string
				Error         string
				Success       string
			}{
				MySQLHost:     mysqlHost,
				MySQLPort:     mysqlPort,
				MySQLUser:     mysqlUser,
				MySQLPassword: mysqlPassword,
				MySQLDatabase: mysqlDatabase,
				ServerPort:    serverPort,
				AdminUsername: adminUsername,
				AdminPassword: adminPassword,
				Error:         fmt.Sprintf("数据库初始化失败: %v", err),
			}
			
			// 直接解析和执行 config.html 模板
			t, err := template.ParseFiles("templates/config.html")
			if err != nil {
				log.Printf("Error parsing template config.html: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			err = t.Execute(w, data)
			if err != nil {
				log.Printf("Error executing template config.html: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		}

		// 保存配置完成，准备重启服务器
		// 显示重启提示页面
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		
		// 重启提示页面
		restartHTML := `
		<!DOCTYPE html>
		<html lang="zh-CN">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>服务器重启中</title>
			<script src="https://cdn.tailwindcss.com"></script>
			<link href="https://cdn.jsdelivr.net/npm/font-awesome@4.7.0/css/font-awesome.min.css" rel="stylesheet">
			<style>
				@keyframes spin {
					from { transform: rotate(0deg); }
					to { transform: rotate(360deg); }
				}
				.spinner {
					animation: spin 1s linear infinite;
				}
			</style>
		</head>
		<body class="bg-gray-50 min-h-screen flex items-center justify-center">
			<div class="bg-white rounded-lg shadow-lg p-8 max-w-md w-full text-center">
				<div class="flex justify-center mb-6">
					<div class="spinner text-blue-600">
						<i class="fa fa-circle-o-notch fa-4x"></i>
					</div>
				</div>
				<h1 class="text-2xl font-bold text-gray-800 mb-4">服务器重启中</h1>
				<p class="text-gray-600 mb-6">配置已保存，服务器正在重启以应用新配置...</p>
				<div class="text-sm text-gray-500">
					<p>请稍候，页面将自动跳转到登录页面</p>
					<p class="mt-2">如果没有自动跳转，请<a href="/login" class="text-blue-600 hover:underline">点击这里</a></p>
				</div>
			</div>
		</body>
		<script>
			// 3秒后跳转到登录页面
			setTimeout(function() {
				window.location.href = '/login';
			}, 3000);
		</script>
		</html>
		`
		
		w.Write([]byte(restartHTML))
		
		// 在后台重启服务器
		go func() {
			// 等待一点时间让响应完成
			time.Sleep(1 * time.Second)
			
			// 重启当前进程
			log.Println("正在重启服务器以应用新配置...")
			
			// 获取当前可执行文件路径
			currentExec, err := os.Executable()
			if err != nil {
				log.Printf("获取可执行文件路径失败: %v", err)
				return
			}
			
			// 启动新进程
			cmd := exec.Command(currentExec)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			// 启动新进程
			err = cmd.Start()
			if err != nil {
				log.Printf("启动新进程失败: %v", err)
				return
			}
			
			// 退出当前进程
			log.Println("新进程已启动，正在退出当前进程...")
			os.Exit(0)
		}()
		
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// testDatabaseConnection 处理数据库连接测试请求
func testDatabaseConnection(w http.ResponseWriter, r *http.Request) {
	// 读取表单数据
	mysqlHost := r.FormValue("mysql_host")
	mysqlPort := r.FormValue("mysql_port")
	mysqlUser := r.FormValue("mysql_user")
	mysqlPassword := r.FormValue("mysql_password")
	mysqlDatabase := r.FormValue("mysql_database")

	// 测试数据库连接
	success, created, err := testDatabaseConnectionInternal(mysqlHost, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase)

	// 返回 JSON 响应
	response := map[string]interface{}{
		"success": success,
		"created": created,
	}

	if err != nil {
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// testDatabaseConnectionInternal 内部测试数据库连接函数
func testDatabaseConnectionInternal(mysqlHost, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase string) (bool, bool, error) {
	log.Printf("开始测试数据库连接: Host=%s, Port=%s, User=%s, Database=%s", mysqlHost, mysqlPort, mysqlUser, mysqlDatabase)

	// 首先连接到 MySQL 服务器
	serverDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlUser, mysqlPassword, mysqlHost, mysqlPort)

	log.Printf("服务器连接字符串: %s", serverDSN)

	serverDB, err := sql.Open("mysql", serverDSN)
	if err != nil {
		log.Printf("打开服务器连接失败: %v", err)
		return false, false, fmt.Errorf("打开服务器连接失败: %w", err)
	}
	defer serverDB.Close()

	// 设置连接超时
	serverDB.SetConnMaxLifetime(time.Second * 5)
	serverDB.SetMaxOpenConns(1)
	serverDB.SetMaxIdleConns(0)

	// 测试服务器连接
	log.Println("正在测试服务器连接...")
	err = serverDB.Ping()
	if err != nil {
		log.Printf("服务器连接测试失败: %v", err)
		return false, false, fmt.Errorf("服务器连接测试失败: %w", err)
	}
	log.Println("服务器连接测试成功")

	// 检查数据库是否存在
	var dbExists bool
	log.Printf("正在检查数据库 %s 是否存在...", mysqlDatabase)
	err = serverDB.QueryRow("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?", mysqlDatabase).Scan(&dbExists)
	if err != nil {
		log.Printf("检查数据库存在性失败: %v", err)
		return false, false, fmt.Errorf("检查数据库存在性失败: %w", err)
	}

	// 创建数据库（如果不存在）
	created := false
	if !dbExists {
		created = true
		log.Printf("数据库 %s 不存在，正在创建...", mysqlDatabase)
		_, err = serverDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", mysqlDatabase))
		if err != nil {
			log.Printf("创建数据库失败: %v", err)
			// 检查是否是权限错误
			if strings.Contains(err.Error(), "Access denied") {
				return false, false, fmt.Errorf("创建数据库失败: %w\n请确保 MySQL 用户 '%s' 有创建数据库的权限，或者手动创建 '%s' 数据库后再测试", err, mysqlUser, mysqlDatabase)
			}
			return false, false, fmt.Errorf("创建数据库失败: %w", err)
		}
		log.Printf("数据库 %s 创建成功", mysqlDatabase)
	} else {
		log.Printf("数据库 %s 已存在", mysqlDatabase)
	}

	// 连接到数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDatabase)

	log.Printf("数据库连接字符串: %s", dsn)

	testDB, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("打开数据库连接失败: %v", err)
		return false, created, fmt.Errorf("打开数据库连接失败: %w", err)
	}
	defer testDB.Close()

	// 设置连接超时
	testDB.SetConnMaxLifetime(time.Second * 5)
	testDB.SetMaxOpenConns(1)
	testDB.SetMaxIdleConns(0)

	// 测试数据库连接
	log.Printf("正在测试数据库 %s 连接...", mysqlDatabase)
	err = testDB.Ping()
	if err != nil {
		log.Printf("数据库连接测试失败: %v", err)
		return false, created, fmt.Errorf("数据库连接测试失败: %w", err)
	}
	log.Printf("数据库 %s 连接测试成功", mysqlDatabase)

	// 检查数据库中是否存在表
	var tableCount int
	err = testDB.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", mysqlDatabase).Scan(&tableCount)
	if err != nil {
		log.Printf("检查表存在性失败: %v", err)
		return false, created, fmt.Errorf("检查表存在性失败: %w", err)
	}

	log.Printf("数据库 %s 中存在 %d 个表", mysqlDatabase, tableCount)

	// 如果表数量为 0，则创建表
	if tableCount == 0 {
		log.Println("数据库中没有表，正在创建表...")
		
		// 创建用户表
		_, err = testDB.Exec(`CREATE TABLE IF NOT EXISTS users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`)
		if err != nil {
			log.Printf("创建用户表失败: %v", err)
			return false, created, fmt.Errorf("创建用户表失败: %w", err)
		}

		// 创建分类表
		_, err = testDB.Exec(`CREATE TABLE IF NOT EXISTS categories (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			icon VARCHAR(50),
			color VARCHAR(50),
			is_default INT DEFAULT 0,
			is_custom INT DEFAULT 0,
			sort_order INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`)
		if err != nil {
			log.Printf("创建分类表失败: %v", err)
			return false, created, fmt.Errorf("创建分类表失败: %w", err)
		}

		// 创建其他表
		tables := []string{
			`CREATE TABLE IF NOT EXISTS transactions (
				id INT PRIMARY KEY AUTO_INCREMENT,
				user_id INT NOT NULL,
				type VARCHAR(50),
				category_id INT,
				category VARCHAR(255),
				amount DECIMAL(10,2),
				date DATETIME,
				note TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(category_id) REFERENCES categories(id),
				FOREIGN KEY(user_id) REFERENCES users(id)
			);`,
			`CREATE TABLE IF NOT EXISTS finance_goals (
				id INT PRIMARY KEY AUTO_INCREMENT,
				user_id INT NOT NULL,
				type VARCHAR(50),
				target_amount DECIMAL(10,2),
				start_date DATETIME,
				end_date DATETIME,
				FOREIGN KEY(user_id) REFERENCES users(id)
			);`,
			`CREATE TABLE IF NOT EXISTS habits (
				id INT PRIMARY KEY AUTO_INCREMENT,
				user_id INT NOT NULL,
				name VARCHAR(255),
				description TEXT,
				frequency VARCHAR(50),
				streak INT DEFAULT 0,
				total_days INT DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(user_id) REFERENCES users(id)
			);`,
			`CREATE TABLE IF NOT EXISTS habit_logs (
				id INT PRIMARY KEY AUTO_INCREMENT,
				habit_id INT,
				date DATETIME,
				FOREIGN KEY(habit_id) REFERENCES habits(id)
			);`,
			`CREATE TABLE IF NOT EXISTS todos (
				id INT PRIMARY KEY AUTO_INCREMENT,
				user_id INT NOT NULL,
				content TEXT,
				status VARCHAR(50) DEFAULT 'pending',
				due_date DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(user_id) REFERENCES users(id)
			);`,
			`CREATE TABLE IF NOT EXISTS todo_checkins (
				id INT PRIMARY KEY AUTO_INCREMENT,
				todo_id INT,
				checkin_date DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(todo_id) REFERENCES todos(id)
			);`,
			`CREATE TABLE IF NOT EXISTS badges (
				id INT PRIMARY KEY AUTO_INCREMENT,
				user_id INT NOT NULL,
				name VARCHAR(255),
				description TEXT,
				icon VARCHAR(50),
				unlocked INT DEFAULT 0,
				condition_days INT,
				FOREIGN KEY(user_id) REFERENCES users(id)
			);`,
			`CREATE TABLE IF NOT EXISTS diaries (
				id INT PRIMARY KEY AUTO_INCREMENT,
				user_id INT NOT NULL,
				title VARCHAR(255),
				content TEXT,
				weather VARCHAR(50),
				mood VARCHAR(50),
				date DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				FOREIGN KEY(user_id) REFERENCES users(id)
			);`,
		}

		for _, query := range tables {
			_, err = testDB.Exec(query)
			if err != nil {
				log.Printf("创建表失败: %s, %v", query, err)
				return false, created, fmt.Errorf("创建表失败: %w", err)
			}
		}

		// 启用外键约束
		_, err = testDB.Exec("SET FOREIGN_KEY_CHECKS = 1")
		if err != nil {
			log.Printf("启用外键约束失败: %v", err)
			return false, created, fmt.Errorf("启用外键约束失败: %w", err)
		}

		// 插入默认分类数据
		log.Println("正在插入默认分类数据...")
		
		// 默认收入分类
		incomeCategories := []struct {
			Name      string
			Icon      string
			Color     string
			SortOrder int
		}{
			{"工资收入", "💰", "#10B981", 1},
			{"奖金福利", "🎁", "#10B981", 2},
			{"投资理财", "📈", "#10B981", 3},
			{"副业兼职", "💼", "#10B981", 4},
			{"经营收入", "🏪", "#10B981", 5},
			{"其他收入", "💵", "#10B981", 6},
			{"自定义输入", "✏️", "#6B7280", 999},
		}

		// 默认支出分类
		expenseCategories := []struct {
			Name      string
			Icon      string
			Color     string
			SortOrder int
		}{
			{"餐饮美食", "🍽️", "#EF4444", 1},
			{"超市购物", "🛒", "#EF4444", 2},
			{"交通出行", "🚗", "#EF4444", 3},
			{"休闲娱乐", "🎮", "#EF4444", 4},
			{"房租房贷", "🏠", "#EF4444", 5},
			{"水电物业", "💡", "#EF4444", 6},
			{"医疗保健", "🏥", "#EF4444", 7},
			{"教育学习", "📚", "#EF4444", 8},
			{"人情往来", "🎁", "#EF4444", 9},
			{"运动健身", "🏃", "#EF4444", 10},
			{"美容护肤", "💄", "#EF4444", 11},
			{"服饰鞋包", "👔", "#EF4444", 12},
			{"通讯费用", "📱", "#EF4444", 13},
			{"其他支出", "📝", "#EF4444", 14},
			{"自定义输入", "✏️", "#6B7280", 999},
		}

		// 插入收入分类
		for _, cat := range incomeCategories {
			_, err := testDB.Exec(
				"INSERT INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES (?, ?, ?, ?, 1, 0, ?)",
				cat.Name, "income", cat.Icon, cat.Color, cat.SortOrder,
			)
			if err != nil {
				log.Printf("插入收入分类失败: %v", err)
			}
		}

		// 插入支出分类
		for _, cat := range expenseCategories {
			_, err := testDB.Exec(
				"INSERT INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES (?, ?, ?, ?, 1, 0, ?)",
				cat.Name, "expense", cat.Icon, cat.Color, cat.SortOrder,
			)
			if err != nil {
				log.Printf("插入支出分类失败: %v", err)
			}
		}

		log.Println("表创建和数据插入成功！")
	} else {
		log.Println("数据库中已存在表，继续使用现有表...")
	}

	return true, created, nil
}
