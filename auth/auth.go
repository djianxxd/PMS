package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/argon2"
	"net/http"
	"strings"
	"time"
)

// Password hashing parameters
type params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

var p *params = &params{
	memory:      64 * 1024,
	iterations:  3,
	parallelism: 2,
	saltLength:  16,
	keyLength:   32,
}

// HashPassword hashes the password using Argon2
func HashPassword(password string) (string, error) {
	salt, err := generateRandomBytes(p.saltLength)
	if err != nil {
		return "", fmt.Errorf("unable to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	format := "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	fullHash := fmt.Sprintf(format, argon2.Version, p.memory, p.iterations, p.parallelism, b64Salt, b64Hash)

	return fullHash, nil
}

// ComparePassword compares a plaintext password with a hashed password
func ComparePassword(password, hash string) (bool, error) {
	salt, hashBytes, err := decodeHash(hash)
	if err != nil {
		return false, fmt.Errorf("unable to decode hash: %w", err)
	}

	comparisonHash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	return subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1, nil
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func decodeHash(encodedHash string) (salt, hash []byte, err error) {
	var version int
	var memory, iterations uint32
	var parallelism uint8
	_, err = fmt.Sscanf(encodedHash, "$argon2id$v=%d$m=%d,t=%d,p=%d", &version, &memory, &iterations, &parallelism)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid hash format: %w", err)
	}

	salt, hash, err = extractSaltAndHash(encodedHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract salt and hash: %w", err)
	}

	return salt, hash, nil
}

func extractSaltAndHash(encodedHash string) (salt, hash []byte, err error) {
	// Split the hash string
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, fmt.Errorf("invalid hash format: expected 6 parts, got %d", len(parts))
	}

	// parts[0] is empty string
	// parts[1] = "argon2id"
	// parts[2] = "v=19"
	// parts[3] = "m=65536,t=3,p=2"
	// parts[4] = salt (base64)
	// parts[5] = hash (base64)

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}

	return salt, hash, nil
}

// Session management
type Session struct {
	UserID   int
	Username string
	Expiry   time.Time
}

// Sessions store
var sessions = map[string]Session{}

// Session cookie name
const SessionCookieName = "goblog_session"

// CreateSession creates a new session for the user
func CreateSession(w http.ResponseWriter, userID int, username string) {
	sessionID, _ := generateRandomBytes(32)
	sessionIDStr := base64.RawURLEncoding.EncodeToString(sessionID)

	expiry := time.Now().Add(24 * time.Hour)
	sessions[sessionIDStr] = Session{
		UserID:   userID,
		Username: username,
		Expiry:   expiry,
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionIDStr,
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})
}

// ValidateSession checks if a session is valid
func ValidateSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, err
	}

	session, exists := sessions[cookie.Value]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.Expiry) {
		delete(sessions, cookie.Value)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// ClearSession removes the user's session
func ClearSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		delete(sessions, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// CleanupExpiredSessions removes expired sessions
func CleanupExpiredSessions() {
	for sessionID, session := range sessions {
		if time.Now().After(session.Expiry) {
			delete(sessions, sessionID)
		}
	}
}
