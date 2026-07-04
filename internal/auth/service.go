package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/naruto361/command-line-login-system/internal/config"
	"github.com/naruto361/command-line-login-system/internal/email"
	"github.com/naruto361/command-line-login-system/internal/mfa"
	"github.com/naruto361/command-line-login-system/internal/store"
)

var (
	ErrAccountLocked        = errors.New("account is locked due to too many failed login attempts; use reset-password to unlock")
	ErrUsernameTaken        = errors.New("username already taken")
	ErrEmailTaken           = errors.New("email already taken")
	ErrUserNotFound         = errors.New("user not found")
	ErrMFAAlreadyEnabled    = errors.New("MFA is already enabled")
	ErrMFANotEnabled        = errors.New("MFA is not enabled")
	ErrInvalidTOTP          = errors.New("invalid TOTP code")
	ErrTOTPAttemptsExceeded = errors.New("too many invalid TOTP attempts; login aborted")
	ErrInvalidResetToken    = errors.New("invalid or expired reset token")
	ErrAccountNotLocked     = errors.New("account is not locked; password reset is only required when the account is locked")
)

type Service struct {
	store  *store.Store
	cfg    *config.Config
	mailer *email.Sender
}

func NewService(s *store.Store, cfg *config.Config, mailer *email.Sender) *Service {
	return &Service{store: s, cfg: cfg, mailer: mailer}
}

type RegisterInput struct {
	Username        string
	Email           string
	Password        string
	ConfirmPassword string
}

func (svc *Service) Register(in RegisterInput) (*store.User, error) {
	if err := ValidateUsername(in.Username); err != nil {
		return nil, err
	}
	if err := ValidateEmail(in.Email); err != nil {
		return nil, err
	}
	if err := ValidatePassword(in.Password); err != nil {
		return nil, err
	}
	if in.Password != in.ConfirmPassword {
		return nil, ErrPasswordMismatch
	}

	exists, err := svc.store.UsernameExists(in.Username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	exists, err = svc.store.EmailExists(in.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, ErrEmailTaken
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	user := &store.User{
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: hash,
		RegisteredAt: time.Now().UTC(),
	}

	if err := svc.store.CreateUser(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return svc.store.GetUserByUsername(in.Username)
}

type LoginResult struct {
	User *store.User
}

func (svc *Service) LookupUser(identifier string) (*store.User, error) {
	return svc.store.GetUserByUsernameOrEmail(identifier)
}

func (svc *Service) CheckUsernameAvailable(username string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	exists, err := svc.store.UsernameExists(username)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if exists {
		return ErrUsernameTaken
	}
	return nil
}

func (svc *Service) CheckEmailAvailable(email string) error {
	if err := ValidateEmail(email); err != nil {
		return err
	}
	exists, err := svc.store.EmailExists(email)
	if err != nil {
		return fmt.Errorf("check email: %w", err)
	}
	if exists {
		return ErrEmailTaken
	}
	return nil
}

func (svc *Service) VerifyCredentials(identifier, password string) (*store.User, error) {
	user, err := svc.LookupUser(identifier)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	return svc.verifyPassword(user, password)
}

func (svc *Service) VerifyPassword(user *store.User, password string) (*store.User, error) {
	fresh, err := svc.store.GetUserByID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if fresh == nil {
		return nil, ErrInvalidCredentials
	}
	return svc.verifyPassword(fresh, password)
}

func (svc *Service) verifyPassword(user *store.User, password string) (*store.User, error) {
	if user.AccountLocked {
		return nil, ErrAccountLocked
	}

	if !CheckPassword(user.PasswordHash, password) {
		attempts := user.FailedLoginAttempts + 1
		locked := attempts >= svc.cfg.MaxFailedLoginAttempts
		if err := svc.store.UpdateFailedLoginAttempts(user.ID, attempts, locked); err != nil {
			return nil, fmt.Errorf("update failed attempts: %w", err)
		}
		if locked {
			return nil, ErrAccountLocked
		}
		remaining := svc.cfg.MaxFailedLoginAttempts - attempts
		return nil, fmt.Errorf("%w (%d attempt(s) remaining before lockout)", ErrInvalidCredentials, remaining)
	}

	return user, nil
}

func (svc *Service) CompleteLogin(user *store.User) (*store.User, error) {
	now := time.Now().UTC()
	if err := svc.store.ResetFailedLoginAttempts(user.ID); err != nil {
		return nil, fmt.Errorf("reset failed attempts: %w", err)
	}
	if err := svc.store.UpdateLastLogin(user.ID, now); err != nil {
		return nil, fmt.Errorf("update last login: %w", err)
	}
	return svc.store.GetUserByID(user.ID)
}

func (svc *Service) VerifyTOTP(user *store.User, code string) error {
	if user.TOTPSecret == nil || *user.TOTPSecret == "" {
		return ErrMFANotEnabled
	}
	if !mfa.ValidateCode(*user.TOTPSecret, code) {
		return ErrInvalidTOTP
	}
	return nil
}

type EnableMFAResult struct {
	Secret string
	URL    string
}

func (svc *Service) PrepareMFA(userID int64) (*EnableMFAResult, error) {
	user, err := svc.store.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if user.MFAEnabled {
		return nil, ErrMFAAlreadyEnabled
	}

	secret, url, err := mfa.GenerateSecret(user.Email, svc.cfg.AppName)
	if err != nil {
		return nil, err
	}

	return &EnableMFAResult{Secret: secret, URL: url}, nil
}

func (svc *Service) ConfirmEnableMFA(userID int64, secret, code string) error {
	if !mfa.ValidateCode(secret, code) {
		return ErrInvalidTOTP
	}
	return svc.store.SetMFA(userID, true, &secret)
}

func (svc *Service) EnableMFA(userID int64) (*EnableMFAResult, error) {
	return svc.PrepareMFA(userID)
}

func (svc *Service) DisableMFA(userID int64, password, totpCode string) error {
	user, err := svc.store.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	if !user.MFAEnabled {
		return ErrMFANotEnabled
	}
	if !CheckPassword(user.PasswordHash, password) {
		return ErrInvalidCredentials
	}
	if err := svc.VerifyTOTP(user, totpCode); err != nil {
		return err
	}

	return svc.store.SetMFA(userID, false, nil)
}

func (svc *Service) RequestPasswordReset(emailAddr string) (string, error) {
	user, err := svc.store.GetUserByEmail(emailAddr)
	if err != nil {
		return "", fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		// Do not reveal whether email exists
		return "", nil
	}
	if !user.AccountLocked {
		return "", ErrAccountNotLocked
	}

	token, err := generateToken(32)
	if err != nil {
		return "", err
	}

	expiry := time.Now().UTC().Add(svc.cfg.PasswordResetTokenExpiry)
	if err := svc.store.SetPasswordResetToken(user.ID, token, expiry); err != nil {
		return "", fmt.Errorf("set reset token: %w", err)
	}

	subject := fmt.Sprintf("%s - Password Reset", svc.cfg.AppName)
	body := fmt.Sprintf("Your password reset token is: %s\n\nThis token expires at %s UTC.",
		token, expiry.Format(time.RFC3339))

	if err := svc.mailer.Send(user.Email, subject, body); err != nil {
		return "", fmt.Errorf("send reset email: %w", err)
	}

	return token, nil
}

type ResetPasswordInput struct {
	Email           string
	Token           string
	NewPassword     string
	ConfirmPassword string
}

func (svc *Service) ResetPassword(in ResetPasswordInput) error {
	if err := ValidateEmail(in.Email); err != nil {
		return err
	}
	if err := ValidatePassword(in.NewPassword); err != nil {
		return err
	}
	if in.NewPassword != in.ConfirmPassword {
		return ErrPasswordMismatch
	}

	user, err := svc.store.GetUserByEmail(in.Email)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return ErrInvalidResetToken
	}
	if !user.AccountLocked {
		return ErrAccountNotLocked
	}
	if user.PasswordResetToken == nil || *user.PasswordResetToken != in.Token {
		return ErrInvalidResetToken
	}
	if user.PasswordResetExpiry == nil || time.Now().UTC().After(*user.PasswordResetExpiry) {
		return ErrInvalidResetToken
	}

	hash, err := HashPassword(in.NewPassword)
	if err != nil {
		return err
	}

	return svc.store.UnlockAndResetAttempts(user.ID, hash)
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func (svc *Service) GetUser(userID int64) (*store.User, error) {
	return svc.store.GetUserByID(userID)
}

func (svc *Service) MaxTOTPAttempts() int {
	return svc.cfg.MaxTOTPAttempts
}
