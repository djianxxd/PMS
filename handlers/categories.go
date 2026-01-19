package handlers

import (
	"database/sql"
	"encoding/json"
	"goblog/db"
	"goblog/models"
	"net/http"
	"strconv"
	"strings"
)

// GetCategoriesHandler handles getting all categories
func GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	categoryType := r.URL.Query().Get("type")

	var rows *sql.Rows
	var err error

	if categoryType == "income" || categoryType == "expense" {
		rows, err = db.DB.Query("SELECT id, name, type, icon, color, is_default, is_custom, sort_order FROM categories WHERE type = ? ORDER BY sort_order ASC, id ASC", categoryType)
	} else {
		rows, err = db.DB.Query("SELECT id, name, type, icon, color, is_default, is_custom, sort_order FROM categories ORDER BY type, sort_order ASC, id ASC")
	}

	if err != nil {
		http.Error(w, "Error fetching categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		var isDefault, isCustom int
		err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Icon, &cat.Color, &isDefault, &isCustom, &cat.SortOrder, &cat.CreatedAt)
		if err != nil {
			continue
		}
		cat.IsDefault = isDefault == 1
		cat.IsCustom = isCustom == 1
		categories = append(categories, cat)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// AddCategoryHandler handles adding a new custom category
func AddCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cat models.Category
	if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if strings.TrimSpace(cat.Name) == "" {
		http.Error(w, "Category name cannot be empty", http.StatusBadRequest)
		return
	}

	if cat.Type != "income" && cat.Type != "expense" {
		http.Error(w, "Category type must be 'income' or 'expense'", http.StatusBadRequest)
		return
	}

	// Set default values if not provided
	if cat.Icon == "" {
		if cat.Type == "income" {
			cat.Icon = "💰"
		} else {
			cat.Icon = "💳"
		}
	}

	if cat.Color == "" {
		if cat.Type == "income" {
			cat.Color = "#10B981"
		} else {
			cat.Color = "#EF4444"
		}
	}

	// Get max sort order for this type
	var maxSortOrder int
	db.DB.QueryRow("SELECT COALESCE(MAX(sort_order), 0) FROM categories WHERE type = ?", cat.Type).Scan(&maxSortOrder)
	cat.SortOrder = maxSortOrder + 1

	// Insert into database
	result, err := db.DB.Exec(
		"INSERT INTO categories (name, type, icon, color, is_default, is_custom, sort_order) VALUES (?, ?, ?, ?, 0, 1, ?)",
		cat.Name, cat.Type, cat.Icon, cat.Color, cat.SortOrder,
	)
	if err != nil {
		http.Error(w, "Error creating category", http.StatusInternalServerError)
		return
	}

	// Get the inserted ID
	id, _ := result.LastInsertId()
	cat.ID = int(id)
	cat.IsDefault = false
	cat.IsCustom = true

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cat)
}

// UpdateCategoryHandler handles updating an existing category
func UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var cat models.Category
	if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if category exists and is custom
	var isCustom int
	err = db.DB.QueryRow("SELECT is_custom FROM categories WHERE id = ?", id).Scan(&isCustom)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	if isCustom != 1 {
		http.Error(w, "Cannot edit default categories", http.StatusForbidden)
		return
	}

	// Update category
	_, err = db.DB.Exec(
		"UPDATE categories SET name = ?, icon = ?, color = ? WHERE id = ?",
		cat.Name, cat.Icon, cat.Color, id,
	)
	if err != nil {
		http.Error(w, "Error updating category", http.StatusInternalServerError)
		return
	}

	// Return updated category
	cat.ID = id
	cat.IsDefault = false
	cat.IsCustom = true

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cat)
}

// DeleteCategoryHandler handles deleting a category
func DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Check if category exists and is custom
	var isCustom int
	err = db.DB.QueryRow("SELECT is_custom FROM categories WHERE id = ?", id).Scan(&isCustom)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	if isCustom != 1 {
		http.Error(w, "Cannot delete default categories", http.StatusForbidden)
		return
	}

	// Check if category is being used by transactions
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE category_id = ?", id).Scan(&count)
	if count > 0 {
		http.Error(w, "Cannot delete category that is being used by transactions", http.StatusForbidden)
		return
	}

	// Delete category
	_, err = db.DB.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Error deleting category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Category deleted successfully"})
}

// ReorderCategoriesHandler handles reordering categories
func ReorderCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		CategoryIDs []int `json:"category_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update sort order for each category
	for i, id := range request.CategoryIDs {
		_, err := db.DB.Exec("UPDATE categories SET sort_order = ? WHERE id = ?", i+1, id)
		if err != nil {
			http.Error(w, "Error updating category order", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Categories reordered successfully"})
}
