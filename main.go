package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"goblog/config"
	"goblog/db"
	"goblog/handlers"
	"goblog/utils"
)

// 检查并修复所有用户的徽章的函数
func checkBadges() {
	// 检查 db.DB 是否初始化
	if db.DB == nil {
		log.Println("数据库未初始化，跳过徽章检查")
		return
	}

	log.Println("正在检查所有用户的徽章...")

	// 查询所有用户
	rows, err := db.DB.Query("SELECT id, username FROM users")
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var userID int
		var username string
		err := rows.Scan(&userID, &username)
		if err != nil {
			log.Printf("扫描用户失败: %v", err)
			continue
		}

		// 为用户创建徽章
		db.CreateUserBadges(userID)
		log.Printf("已为用户 %s (ID: %d) 创建徽章", username, userID)
		count++
	}

	log.Printf("已为 %d 个用户创建徽章", count)
	log.Println("徽章检查和修复完成！")
}

func main() {
	// 加载配置
	err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 确保端口可用
	port, _ := strconv.Atoi(config.AppConfig.Server.Port)
	err = utils.EnsurePortAvailable(port)
	if err != nil {
		log.Printf("警告: 确保端口可用失败: %v", err)
		// 继续执行，可能端口未被占用
	}

	// 设置路由
	setupRoutes()

	// 自动打开浏览器
	go func() {
		time.Sleep(2 * time.Second) // 等待服务器启动
		// 检查是否已初始化
		if config.IsInitialized() {
			// 已初始化，打开登录页面
			openBrowser(fmt.Sprintf("http://localhost:%s/login", config.AppConfig.Server.Port))
		} else {
			// 未初始化，打开配置页面
			openBrowser(fmt.Sprintf("http://localhost:%s/config", config.AppConfig.Server.Port))
		}
	}()

	// 启动服务器
	addr := fmt.Sprintf(":%s", config.AppConfig.Server.Port)
	log.Printf("服务器已启动，访问地址：http://localhost:%s", config.AppConfig.Server.Port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// setupRoutes 设置路由
func setupRoutes() {
	// 配置路由
	http.HandleFunc("/config", handlers.ConfigHandler)
	// 测试路由
	http.HandleFunc("/config/test", handlers.ConfigHandler)

	// 检查是否已初始化
	if config.IsInitialized() {
		// 清理旧的 badges 表（如果存在但没有 user_id 字段）
		// 这是为了修复数据隔离问题
		log.Println("正在检查数据库结构...")

		// 临时初始化数据库连接来检查结构
		tempErr := db.InitDB()
		if tempErr == nil {
			// 检查 badges 表是否有 user_id 字段
			var hasUserIDColumn bool
			err := db.DB.QueryRow(`
				SELECT COUNT(*)
				FROM information_schema.columns
				WHERE table_schema = DATABASE()
				AND table_name = 'badges'
				AND column_name = 'user_id'
			`).Scan(&hasUserIDColumn)

			if err != nil {
				log.Printf("检查 badges 表结构失败: %v", err)
			} else if !hasUserIDColumn {
				log.Println("发现旧的 badges 表结构，正在清理...")
				// 删除旧的 badges 表
				_, dropErr := db.DB.Exec("DROP TABLE IF EXISTS badges")
				if dropErr != nil {
					log.Printf("删除旧的 badges 表失败: %v", dropErr)
				} else {
					log.Println("旧的 badges 表已删除，将重新创建")
				}
			}
		}

		// 重新初始化数据库连接
		err := db.InitDB()
		if err != nil {
			log.Printf("数据库初始化失败: %v", err)
			// 数据库初始化失败，重定向到配置页面
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/config" && r.URL.Path != "/config/test" {
					http.Redirect(w, r, "/config", http.StatusSeeOther)
				}
			})
			return
		}

		// 检查并修复所有用户的徽章
		go checkBadges()

		// 注册其他路由
		setupNormalRoutes()
	} else {
		// 未初始化，重定向到配置页面
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/config" && r.URL.Path != "/config/test" {
				http.Redirect(w, r, "/config", http.StatusSeeOther)
			}
		})
	}
}

// setupNormalRoutes 设置正常路由
func setupNormalRoutes() {
	// Test route
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
		<h2>数据库测试</h2>
		<a href="/test/insert">插入测试数据</a> | 
		<a href="/">查看仪表板</a>
		`)
	})

	http.HandleFunc("/test/insert", func(w http.ResponseWriter, r *http.Request) {
		// 插入测试数据
		now := time.Now()

		_, err := db.DB.Exec(
			"INSERT INTO transactions (user_id, type, category_id, category, amount, date, note) VALUES (?, ?, ?, ?, ?, ?, ?)",
			1, "income", 1, "工资收入", 8000.00, now, "测试收入",
		)
		if err != nil {
			fmt.Fprintf(w, "插入收入失败: %v", err)
			return
		}

		_, err = db.DB.Exec(
			"INSERT INTO transactions (user_id, type, category_id, category, amount, date, note) VALUES (?, ?, ?, ?, ?, ?, ?)",
			1, "expense", 1, "餐饮美食", 150.50, now, "测试支出",
		)
		if err != nil {
			fmt.Fprintf(w, "插入支出失败: %v", err)
			return
		}

		fmt.Fprintf(w, `
		<h2>✅ 测试数据插入成功！</h2>
		<p>收入: ¥8000.00</p>
		<p>支出: ¥150.50</p>
		<a href="/">查看仪表板</a>
		`)
	})

	// Authentication routes (no middleware required)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Protected routes
	http.HandleFunc("/", handlers.AuthMiddleware(handlers.DashboardHandler))

	http.HandleFunc("/finance", handlers.AuthMiddleware(handlers.FinanceHandler))
	http.HandleFunc("/finance/add", handlers.AuthMiddleware(handlers.AddTransactionHandler))
	http.HandleFunc("/finance/delete", handlers.AuthMiddleware(handlers.DeleteTransactionHandler))

	// Category management
	http.HandleFunc("/api/categories", handlers.AuthMiddleware(handlers.GetCategoriesHandler))

	http.HandleFunc("/habits", handlers.AuthMiddleware(handlers.HabitsHandler))
	http.HandleFunc("/habits/add", handlers.AuthMiddleware(handlers.AddHabitHandler))
	http.HandleFunc("/habits/delete", handlers.AuthMiddleware(handlers.DeleteHabitHandler))
	http.HandleFunc("/habits/checkin", handlers.AuthMiddleware(handlers.CheckinHabitHandler))

	http.HandleFunc("/todos", handlers.AuthMiddleware(handlers.TodosHandler))
	http.HandleFunc("/todos/add", handlers.AuthMiddleware(handlers.AddTodoHandler))
	http.HandleFunc("/todos/toggle", handlers.AuthMiddleware(handlers.ToggleTodoHandler))
	http.HandleFunc("/todos/checkin", handlers.AuthMiddleware(handlers.CheckinTodoHandler))
	http.HandleFunc("/todos/checkins", handlers.AuthMiddleware(handlers.TodoCheckinsHandler))
	http.HandleFunc("/todos/delete", handlers.AuthMiddleware(handlers.DeleteTodoHandler))

	http.HandleFunc("/diary", handlers.AuthMiddleware(handlers.DiaryHandler))
	http.HandleFunc("/diary/add", handlers.AuthMiddleware(handlers.AddDiaryHandler))
	http.HandleFunc("/diary/delete", handlers.AuthMiddleware(handlers.DeleteDiaryHandler))
	http.HandleFunc("/diary/get", handlers.AuthMiddleware(handlers.GetDiaryHandler))
	http.HandleFunc("/diary/update", handlers.AuthMiddleware(handlers.UpdateDiaryHandler))

	http.HandleFunc("/export", handlers.AuthMiddleware(handlers.ExportHandler))
}

// openBrowser 打开浏览器
func openBrowser(url string) {
	var err error
	switch os := os.Getenv("OS"); os {
	case "Windows_NT":
		err = exec.Command("cmd", "/c", "start", url).Start()
	default:
		// 其他操作系统不处理
	}
	if err != nil {
		log.Printf("打开浏览器失败: %v", err)
	}
}
