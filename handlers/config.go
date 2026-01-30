package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"goblog/config"
	"goblog/db"
)

// ConfigHandler 处理配置页面
func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := map[string]interface{}{
			"MySQLHost":     config.AppConfig.MySQL.Host,
			"MySQLPort":     config.AppConfig.MySQL.Port,
			"MySQLUser":     config.AppConfig.MySQL.User,
			"MySQLPassword": config.AppConfig.MySQL.Password,
			"MySQLDatabase": config.AppConfig.MySQL.Database,
			"ServerPort":    config.AppConfig.Server.Port,
		}
		renderTemplate(w, "config.html", data)
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

		// 更新配置
		config.AppConfig.MySQL.Host = mysqlHost
		config.AppConfig.MySQL.Port = mysqlPort
		config.AppConfig.MySQL.User = mysqlUser
		config.AppConfig.MySQL.Password = mysqlPassword
		config.AppConfig.MySQL.Database = mysqlDatabase
		config.AppConfig.Server.Port = serverPort

		// 测试数据库连接
		success, created, err := testDatabaseConnectionInternal(mysqlHost, mysqlPort, mysqlUser, mysqlPassword, mysqlDatabase)
		if !success {
			data := map[string]interface{}{
				"MySQLHost":     mysqlHost,
				"MySQLPort":     mysqlPort,
				"MySQLUser":     mysqlUser,
				"MySQLPassword": mysqlPassword,
				"MySQLDatabase": mysqlDatabase,
				"ServerPort":    serverPort,
				"Error":         fmt.Sprintf("数据库连接失败: %v", err),
			}
			renderTemplate(w, "config.html", data)
			return
		}

		// 记录数据库创建状态
		if created {
			log.Printf("数据库 %s 不存在，已自动创建", mysqlDatabase)
		}

		// 保存配置
		err = config.SaveConfig()
		if err != nil {
			data := map[string]interface{}{
				"MySQLHost":     mysqlHost,
				"MySQLPort":     mysqlPort,
				"MySQLUser":     mysqlUser,
				"MySQLPassword": mysqlPassword,
				"MySQLDatabase": mysqlDatabase,
				"ServerPort":    serverPort,
				"Error":         fmt.Sprintf("保存配置失败: %v", err),
			}
			renderTemplate(w, "config.html", data)
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
			data := map[string]interface{}{
				"MySQLHost":     mysqlHost,
				"MySQLPort":     mysqlPort,
				"MySQLUser":     mysqlUser,
				"MySQLPassword": mysqlPassword,
				"MySQLDatabase": mysqlDatabase,
				"ServerPort":    serverPort,
				"Error":         fmt.Sprintf("数据库初始化失败: %v", err),
			}
			renderTemplate(w, "config.html", data)
			return
		}

		// 重定向到登录页面
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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

	return true, created, nil
}
