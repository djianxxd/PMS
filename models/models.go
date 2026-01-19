package models

import (
	"time"
)

// Transaction represents an income or expense
type Transaction struct {
	ID         int       `json:"id"`
	Type       string    `json:"type"` // "income" or "expense"
	CategoryID int       `json:"category_id"`
	Category   string    `json:"category"`
	Amount     float64   `json:"amount"`
	Date       time.Time `json:"date"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

// Category represents a transaction category
type Category struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`  // "income" or "expense"
	Icon      string    `json:"icon"`  // emoji or icon class
	Color     string    `json:"color"` // hex color code
	IsDefault bool      `json:"is_default"`
	IsCustom  bool      `json:"is_custom"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

// FinanceGoal represents a budget goal
type FinanceGoal struct {
	ID            int       `json:"id"`
	Type          string    `json:"type"` // "weekly", "monthly", "yearly"
	TargetAmount  float64   `json:"target_amount"`
	CurrentAmount float64   `json:"current_amount"` // Calculated dynamically usually, but struct useful for passing to view
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
}

// Habit represents a habit to track
type Habit struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Frequency   string     `json:"frequency"` // "daily", "weekly"
	Streak      int        `json:"streak"`
	TotalDays   int        `json:"total_days"`
	CreatedAt   time.Time  `json:"created_at"`
	Logs        []HabitLog `json:"logs"` // For easy access in template
}

// HabitLog represents a completion of a habit
type HabitLog struct {
	ID      int       `json:"id"`
	HabitID int       `json:"habit_id"`
	Date    time.Time `json:"date"`
}

// Todo represents a task
type Todo struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"` // "pending", "completed"
	DueDate   time.Time `json:"due_date"`
	CreatedAt time.Time `json:"created_at"`
}

// Badge represents an achievement
type Badge struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"` // FontAwesome class or emoji
	Unlocked    bool   `json:"unlocked"`
}
