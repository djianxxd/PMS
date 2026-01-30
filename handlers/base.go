package handlers

import (
	"goblog/auth"
	"net/http"
)

// BaseData contains common data for all templates
type BaseData struct {
	ActivePage string
	User       *auth.Session
	IsLoggedIn bool
}

// GetBaseData returns common template data including user session
func GetBaseData(r *http.Request, activePage string) BaseData {
	session, _ := auth.ValidateSession(r)
	return BaseData{
		ActivePage: activePage,
		User:       session,
		IsLoggedIn: session != nil,
	}
}
