package handlers

import (
	"context"
	"fmt"
	"goblog/auth"
	"goblog/config"
	"log"
	"net/http"
)

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Check if already logged in
		if _, err := auth.ValidateSession(r); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		returnURL := r.URL.Query().Get("return")
		if returnURL == "" {
			returnURL = "/"
		}

		data := map[string]interface{}{
			"ReturnURL": returnURL,
			"Title":     "登录",
		}
		renderTemplate(w, "login.html", data)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		returnURL := r.FormValue("return_url")
		if returnURL == "" {
			returnURL = "/"
		}

		// 检查是否是管理员登录
		if username == config.AppConfig.Admin.Username && password == config.AppConfig.Admin.Password {
			// 管理员登录
			// 创建会话
			auth.CreateSession(w, 0, "admin")
			// 跳转到管理后台
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		// 普通用户登录验证
		user, err := GetUserByUsername(username)
		if err != nil {
			data := map[string]interface{}{
				"Error":     "用户名或密码错误",
				"ReturnURL": returnURL,
				"Title":     "登录",
			}
			renderTemplate(w, "login.html", data)
			return
		}

		valid, err := auth.ComparePassword(password, user.Password)
		if err != nil || !valid {
			data := map[string]interface{}{
				"Error":     "用户名或密码错误",
				"ReturnURL": returnURL,
				"Title":     "登录",
			}
			renderTemplate(w, "login.html", data)
			return
		}

		// Create session
		auth.CreateSession(w, user.ID, user.Username)

		// Redirect to requested page
		http.Redirect(w, r, returnURL, http.StatusSeeOther)
		return
	}
}

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Check if already logged in
		if _, err := auth.ValidateSession(r); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		data := map[string]interface{}{
			"Title": "注册",
		}
		renderTemplate(w, "register.html", data)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		// Basic validation
		if username == "" || email == "" || password == "" {
			data := map[string]interface{}{
				"Error": "所有字段都是必填的",
				"Title": "注册",
				"Values": map[string]string{
					"username": username,
					"email":    email,
				},
			}
			renderTemplate(w, "register.html", data)
			return
		}

		if password != confirmPassword {
			data := map[string]interface{}{
				"Error": "两次输入的密码不一致",
				"Title": "注册",
				"Values": map[string]string{
					"username": username,
					"email":    email,
				},
			}
			renderTemplate(w, "register.html", data)
			return
		}

		if len(password) < 6 {
			data := map[string]interface{}{
				"Error": "密码长度至少为6位",
				"Title": "注册",
				"Values": map[string]string{
					"username": username,
					"email":    email,
				},
			}
			renderTemplate(w, "register.html", data)
			return
		}

		// Check if username already exists
		if _, err := GetUserByUsername(username); err == nil {
			data := map[string]interface{}{
				"Error": "用户名已存在",
				"Title": "注册",
				"Values": map[string]string{
					"username": username,
					"email":    email,
				},
			}
			renderTemplate(w, "register.html", data)
			return
		}

		// Check if email already exists
		if _, err := GetUserByEmail(email); err == nil {
			data := map[string]interface{}{
				"Error": "邮箱已被使用",
				"Title": "注册",
				"Values": map[string]string{
					"username": username,
					"email":    email,
				},
			}
			renderTemplate(w, "register.html", data)
			return
		}

		// Hash password
		hashedPassword, err := auth.HashPassword(password)
		if err != nil {
			log.Printf("Password hashing error: %v", err)
			data := map[string]interface{}{
				"Error": "创建账户失败，请重试",
				"Title": "注册",
				"Values": map[string]string{
					"username": username,
					"email":    email,
				},
			}
			renderTemplate(w, "register.html", data)
			return
		}

		log.Printf("Creating user: %s, %s, hash length: %d", username, email, len(hashedPassword))

		// Create user
		user, err := CreateUser(username, email, hashedPassword)
		if err != nil {
			log.Printf("User creation error: %v", err)
			data := map[string]interface{}{
				"Error": fmt.Sprintf("创建账户失败: %v", err),
				"Title": "注册",
				"Values": map[string]string{
					"username": username,
					"email":    email,
				},
			}
			renderTemplate(w, "register.html", data)
			return
		}

		log.Printf("User created successfully: %+v", user)

		// Create session and redirect
		auth.CreateSession(w, user.ID, user.Username)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}

// LogoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	auth.ClearSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// UserContextKey is key for user context
type UserContextKey string

const UserIDKey UserContextKey = "userID"

// AuthMiddleware protects routes that require authentication
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := auth.ValidateSession(r)
		if err != nil {
			// Redirect to login page with return URL
			returnURL := r.URL.Path
			http.Redirect(w, r, "/login?return="+returnURL, http.StatusSeeOther)
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), UserIDKey, session.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserIDKey).(int)
	return userID, ok
}
