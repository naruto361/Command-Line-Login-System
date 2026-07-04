package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := s.db.Exec(string(schema)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func scanUser(row scanner) (*User, error) {
	var u User
	var mfaEnabled, accountLocked int
	var totpSecret, resetToken sql.NullString
	var resetExpiry, lastLogin sql.NullTime

	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&mfaEnabled, &totpSecret, &u.FailedLoginAttempts, &accountLocked,
		&resetToken, &resetExpiry, &u.RegisteredAt, &lastLogin,
	)
	if err != nil {
		return nil, err
	}

	u.MFAEnabled = mfaEnabled == 1
	u.AccountLocked = accountLocked == 1
	if totpSecret.Valid {
		u.TOTPSecret = &totpSecret.String
	}
	if resetToken.Valid {
		u.PasswordResetToken = &resetToken.String
	}
	if resetExpiry.Valid {
		t := resetExpiry.Time
		u.PasswordResetExpiry = &t
	}
	if lastLogin.Valid {
		t := lastLogin.Time
		u.LastLogin = &t
	}

	return &u, nil
}

type scanner interface {
	Scan(dest ...any) error
}

const userColumns = `id, username, email, password_hash, mfa_enabled, totp_secret,
	failed_login_attempts, account_locked, password_reset_token, password_reset_expiry,
	registered_at, last_login`

func (s *Store) CreateUser(u *User) error {
	_, err := s.db.Exec(`
		INSERT INTO users (username, email, password_hash, mfa_enabled, totp_secret,
			failed_login_attempts, account_locked, registered_at)
		VALUES (?, ?, ?, 0, NULL, 0, 0, ?)`,
		u.Username, u.Email, u.PasswordHash, u.RegisteredAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	row := s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE username = ? COLLATE NOCASE`, username)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *Store) GetUserByEmail(email string) (*User, error) {
	row := s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE email = ? COLLATE NOCASE`, email)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *Store) GetUserByUsernameOrEmail(identifier string) (*User, error) {
	row := s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE username = ? COLLATE NOCASE OR email = ? COLLATE NOCASE`,
		identifier, identifier)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	row := s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *Store) GetUserByResetToken(token string) (*User, error) {
	row := s.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE password_reset_token = ?`, token)
	u, err := scanUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (s *Store) UsernameExists(username string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ? COLLATE NOCASE`, username).Scan(&count)
	return count > 0, err
}

func (s *Store) EmailExists(email string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = ? COLLATE NOCASE`, email).Scan(&count)
	return count > 0, err
}

func (s *Store) UpdatePasswordHash(userID int64, hash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, userID)
	return err
}

func (s *Store) UpdateFailedLoginAttempts(userID int64, attempts int, locked bool) error {
	lockedInt := 0
	if locked {
		lockedInt = 1
	}
	_, err := s.db.Exec(`UPDATE users SET failed_login_attempts = ?, account_locked = ? WHERE id = ?`,
		attempts, lockedInt, userID)
	return err
}

func (s *Store) ResetFailedLoginAttempts(userID int64) error {
	_, err := s.db.Exec(`UPDATE users SET failed_login_attempts = 0, account_locked = 0 WHERE id = ?`, userID)
	return err
}

func (s *Store) UpdateLastLogin(userID int64, t time.Time) error {
	_, err := s.db.Exec(`UPDATE users SET last_login = ? WHERE id = ?`, t.UTC().Format(time.RFC3339), userID)
	return err
}

func (s *Store) SetMFA(userID int64, enabled bool, secret *string) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := s.db.Exec(`UPDATE users SET mfa_enabled = ?, totp_secret = ? WHERE id = ?`,
		enabledInt, secret, userID)
	return err
}

func (s *Store) SetPasswordResetToken(userID int64, token string, expiry time.Time) error {
	_, err := s.db.Exec(`UPDATE users SET password_reset_token = ?, password_reset_expiry = ? WHERE id = ?`,
		token, expiry.UTC().Format(time.RFC3339), userID)
	return err
}

func (s *Store) ClearPasswordResetToken(userID int64) error {
	_, err := s.db.Exec(`UPDATE users SET password_reset_token = NULL, password_reset_expiry = NULL WHERE id = ?`, userID)
	return err
}

func (s *Store) UnlockAndResetAttempts(userID int64, hash string) error {
	_, err := s.db.Exec(`
		UPDATE users SET password_hash = ?, failed_login_attempts = 0, account_locked = 0,
			password_reset_token = NULL, password_reset_expiry = NULL
		WHERE id = ?`, hash, userID)
	return err
}
