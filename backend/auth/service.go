package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type contextKey string

const (
	userContextKey contextKey = "auth.user"
	defaultDBPath             = "data/app.db"
	defaultSessionTTL         = 30 * 24 * time.Hour
	sqliteTimeLayout          = "2006-01-02 15:04:05"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrSetupRequired      = errors.New("initial setup is required")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrUserExists         = errors.New("initial user already exists")
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuthResult struct {
	Token     string `json:"token"`
	User      User   `json:"user"`
	ExpiresAt string `json:"expiresAt"`
}

type Service struct {
	db         *sql.DB
	sessionTTL time.Duration
}

func Init(dbPath string) (*Service, error) {
	resolvedPath := strings.TrimSpace(dbPath)
	if resolvedPath == "" {
		resolvedPath = strings.TrimSpace(os.Getenv("DOCKER_UI_DB_PATH"))
	}
	if resolvedPath == "" {
		resolvedPath = defaultDBPath
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return nil, fmt.Errorf("create auth data dir: %w", err)
	}

	db, err := sql.Open("sqlite", resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	svc := &Service{
		db:         db,
		sessionTTL: defaultSessionTTL,
	}

	if err := svc.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	if err := svc.cleanupExpiredSessions(); err != nil {
		db.Close()
		return nil, err
	}

	return svc, nil
}

func (s *Service) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Service) initSchema() error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA busy_timeout = 5000;`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);`,
	}

	for _, stmt := range statements {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("init auth schema: %w", err)
		}
	}

	return nil
}

func (s *Service) HasUsers() (bool, error) {
	row := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users LIMIT 1)`)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Service) SetupRequired() (bool, error) {
	hasUsers, err := s.HasUsers()
	if err != nil {
		return false, err
	}
	return !hasUsers, nil
}

func (s *Service) CreateInitialUser(username, password string) (AuthResult, error) {
	cleanUsername, cleanPassword, err := validateCredentials(username, password)
	if err != nil {
		return AuthResult{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return AuthResult{}, err
	}
	defer tx.Rollback()

	var userExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM users LIMIT 1)`).Scan(&userExists); err != nil {
		return AuthResult{}, err
	}
	if userExists {
		return AuthResult{}, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cleanPassword), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, err
	}

	result, err := tx.Exec(`INSERT INTO users (username, password_hash) VALUES (?, ?)`, cleanUsername, string(hash))
	if err != nil {
		return AuthResult{}, err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.loadUserTx(tx, userID)
	if err != nil {
		return AuthResult{}, err
	}

	authResult, err := s.createSessionTx(tx, user)
	if err != nil {
		return AuthResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return AuthResult{}, err
	}

	return authResult, nil
}

func (s *Service) Login(username, password string) (AuthResult, error) {
	cleanUsername := strings.TrimSpace(username)
	if cleanUsername == "" || strings.TrimSpace(password) == "" {
		return AuthResult{}, ErrInvalidCredentials
	}

	if err := s.cleanupExpiredSessions(); err != nil {
		return AuthResult{}, err
	}

	var user User
	var passwordHash string
	var createdAtRaw string
	row := s.db.QueryRow(`
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE username = ?
	`, cleanUsername)
	if err := row.Scan(&user.ID, &user.Username, &passwordHash, &createdAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}
	user.CreatedAt = parseSQLiteTime(createdAtRaw)

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	tx, err := s.db.Begin()
	if err != nil {
		return AuthResult{}, err
	}
	defer tx.Rollback()

	authResult, err := s.createSessionTx(tx, user)
	if err != nil {
		return AuthResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return AuthResult{}, err
	}

	return authResult, nil
}

func (s *Service) Logout(token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}

	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

func (s *Service) ValidateToken(token string) (*User, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrUnauthorized
	}

	row := s.db.QueryRow(`
		SELECT users.id, users.username, users.created_at
		FROM sessions
		INNER JOIN users ON users.id = sessions.user_id
		WHERE sessions.token_hash = ? AND sessions.expires_at > CURRENT_TIMESTAMP
	`, hashToken(token))

	var user User
	var createdAtRaw string
	if err := row.Scan(&user.ID, &user.Username, &createdAtRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
	user.CreatedAt = parseSQLiteTime(createdAtRaw)

	return &user, nil
}

func (s *Service) AuthorizeRequest(r *http.Request) error {
	setupRequired, err := s.SetupRequired()
	if err != nil {
		return err
	}
	if setupRequired {
		return ErrSetupRequired
	}

	user, err := s.ValidateToken(TokenFromRequest(r))
	if err != nil {
		return err
	}

	ctx := context.WithValue(r.Context(), userContextKey, user)
	*r = *r.WithContext(ctx)
	return nil
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setupRequired, err := s.SetupRequired()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if setupRequired {
			http.Error(w, ErrSetupRequired.Error(), http.StatusServiceUnavailable)
			return
		}

		token := TokenFromRequest(r)
		user, err := s.ValidateToken(token)
		if err != nil {
			log.Printf("auth middleware denied path=%s method=%s has_token=%t token_prefix=%q err=%v", r.URL.Path, r.Method, strings.TrimSpace(token) != "", tokenPrefix(token), err)
			if errors.Is(err, ErrUnauthorized) {
				http.Error(w, ErrUnauthorized.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("auth middleware allowed path=%s method=%s user=%s", r.URL.Path, r.Method, user.Username)
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) Status(token string) (bool, *User, error) {
	setupRequired, err := s.SetupRequired()
	if err != nil {
		return false, nil, err
	}
	if setupRequired {
		return true, nil, nil
	}
	if strings.TrimSpace(token) == "" {
		return false, nil, nil
	}

	user, err := s.ValidateToken(token)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			return false, nil, nil
		}
		return false, nil, err
	}

	return false, user, nil
}

func CurrentUser(r *http.Request) *User {
	if r == nil {
		return nil
	}
	value := r.Context().Value(userContextKey)
	user, _ := value.(*User)
	return user
}

func TokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}

	return strings.TrimSpace(r.URL.Query().Get("token"))
}

func validateCredentials(username, password string) (string, string, error) {
	cleanUsername := strings.TrimSpace(username)
	cleanPassword := strings.TrimSpace(password)

	switch {
	case len(cleanUsername) < 3:
		return "", "", errors.New("username must be at least 3 characters")
	case len(cleanUsername) > 64:
		return "", "", errors.New("username must be at most 64 characters")
	case len(cleanPassword) < 8:
		return "", "", errors.New("password must be at least 8 characters")
	case len(cleanPassword) > 128:
		return "", "", errors.New("password must be at most 128 characters")
	}

	return cleanUsername, cleanPassword, nil
}

func (s *Service) loadUserTx(tx *sql.Tx, id int64) (User, error) {
	row := tx.QueryRow(`
		SELECT id, username, created_at
		FROM users
		WHERE id = ?
	`, id)

	var user User
	var createdAtRaw string
	if err := row.Scan(&user.ID, &user.Username, &createdAtRaw); err != nil {
		return User{}, err
	}
	user.CreatedAt = parseSQLiteTime(createdAtRaw)

	return user, nil
}

func (s *Service) createSessionTx(tx *sql.Tx, user User) (AuthResult, error) {
	token, err := generateToken()
	if err != nil {
		return AuthResult{}, err
	}

	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	_, err = tx.Exec(`
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES (?, ?, ?)
	`, user.ID, hashToken(token), expiresAt.Format(sqliteTimeLayout))
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) cleanupExpiredSessions() error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	return err
}

func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokenPrefix(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 8 {
		return token
	}
	return token[:8]
}

func parseSQLiteTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, sqliteTimeLayout, "2006-01-02 15:04:05-07:00"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
