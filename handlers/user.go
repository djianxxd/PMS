package handlers

import (
	"goblog/db"
	"goblog/models"
	"log"
)

// InitTestDB initializes the database for testing
func InitTestDB() {
	db.InitDB()
	log.Println("测试数据库初始化完成")
}

// GetUserByUsername retrieves a user by username
func GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := db.DB.QueryRow(
		"SELECT id, username, email, password, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := db.DB.QueryRow(
		"SELECT id, username, email, password, created_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser creates a new user
func CreateUser(username, email, hashedPassword string) (*models.User, error) {
	result, err := db.DB.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, hashedPassword,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:       int(id),
		Username: username,
		Email:    email,
		Password: hashedPassword,
	}

	// Create badges for new user
	db.CreateUserBadges(user.ID)

	return user, nil
}
