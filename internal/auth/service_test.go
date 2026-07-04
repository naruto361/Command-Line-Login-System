package auth_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/naruto361/command-line-login-system/internal/auth"
	"github.com/naruto361/command-line-login-system/internal/config"
	"github.com/naruto361/command-line-login-system/internal/email"
	"github.com/naruto361/command-line-login-system/internal/store"
)

func testService(t *testing.T, maxAttempts int) (*auth.Service, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	cfg := &config.Config{
		MaxFailedLoginAttempts:   maxAttempts,
		MaxTOTPAttempts:          3,
		PasswordResetTokenExpiry: time.Hour,
		DevMode:                  true,
		AppName:                  "OSTO-TEST",
	}

	svc := auth.NewService(s, cfg, email.NewSender(cfg))
	return svc, s
}

func registerUser(t *testing.T, svc *auth.Service) *store.User {
	t.Helper()
	user, err := svc.Register(auth.RegisterInput{
		Username:        "testuser",
		Email:           "test@example.com",
		Password:        "SecurePass1!",
		ConfirmPassword: "SecurePass1!",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return user
}

func TestRegisterAndLogin(t *testing.T) {
	svc, s := testService(t, 5)
	defer s.Close()

	registerUser(t, svc)

	user, err := svc.VerifyCredentials("testuser", "SecurePass1!")
	if err != nil {
		t.Fatalf("verify credentials: %v", err)
	}
	if user == nil {
		t.Fatal("expected user")
	}

	user, err = svc.CompleteLogin(user)
	if err != nil {
		t.Fatalf("complete login: %v", err)
	}
	if user.LastLogin == nil {
		t.Fatal("expected last login to be set")
	}
}

func TestLoginWithEmail(t *testing.T) {
	svc, s := testService(t, 5)
	defer s.Close()
	registerUser(t, svc)

	_, err := svc.VerifyCredentials("test@example.com", "SecurePass1!")
	if err != nil {
		t.Fatalf("login with email: %v", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc, s := testService(t, 5)
	defer s.Close()

	_, err := svc.VerifyCredentials("nobody", "anypassword")
	if !errors.Is(err, auth.ErrAccountNotFound) {
		t.Fatalf("expected account not found, got %v", err)
	}
}

func TestWrongPasswordMessage(t *testing.T) {
	svc, s := testService(t, 5)
	defer s.Close()
	registerUser(t, svc)

	_, err := svc.VerifyCredentials("testuser", "short")
	if !errors.Is(err, auth.ErrWrongPassword) {
		t.Fatalf("expected wrong password error, got %v", err)
	}
}

func TestAccountLockout(t *testing.T) {
	svc, s := testService(t, 3)
	defer s.Close()
	registerUser(t, svc)

	for i := 0; i < 2; i++ {
		_, err := svc.VerifyCredentials("testuser", "wrong")
		if err == nil {
			t.Fatal("expected error for wrong password")
		}
	}

	_, err := svc.VerifyCredentials("testuser", "wrong")
	if err != auth.ErrAccountLocked {
		t.Fatalf("expected account locked, got %v", err)
	}

	// Locked account rejects even correct password
	_, err = svc.VerifyCredentials("testuser", "SecurePass1!")
	if err != auth.ErrAccountLocked {
		t.Fatalf("expected locked error, got %v", err)
	}
}

func TestFailedAttemptsResetOnLogin(t *testing.T) {
	svc, s := testService(t, 5)
	defer s.Close()
	registerUser(t, svc)

	_, _ = svc.VerifyCredentials("testuser", "wrong")

	user, err := svc.VerifyCredentials("testuser", "SecurePass1!")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	user, err = svc.CompleteLogin(user)
	if err != nil {
		t.Fatalf("complete login: %v", err)
	}
	if user.FailedLoginAttempts != 0 || user.AccountLocked {
		t.Fatalf("expected attempts reset, got attempts=%d locked=%v", user.FailedLoginAttempts, user.AccountLocked)
	}
}

func TestPasswordResetUnlocksAccount(t *testing.T) {
	svc, s := testService(t, 2)
	defer s.Close()
	registerUser(t, svc)

	for i := 0; i < 2; i++ {
		_, _ = svc.VerifyCredentials("testuser", "wrong")
	}

	token, err := svc.RequestPasswordReset("test@example.com")
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if token == "" {
		t.Fatal("expected token in dev mode")
	}

	err = svc.ResetPassword(auth.ResetPasswordInput{
		Email:           "test@example.com",
		Token:           token,
		NewPassword:     "NewSecurePass2@",
		ConfirmPassword: "NewSecurePass2@",
	})
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}

	user, err := svc.VerifyCredentials("testuser", "NewSecurePass2@")
	if err != nil {
		t.Fatalf("login after reset: %v", err)
	}
	if user.AccountLocked {
		t.Fatal("account should be unlocked")
	}
}

func TestMFAEnableDisable(t *testing.T) {
	svc, s := testService(t, 5)
	defer s.Close()
	user := registerUser(t, svc)

	result, err := svc.EnableMFA(user.ID)
	if err != nil {
		t.Fatalf("prepare MFA: %v", err)
	}

	code, err := totp.GenerateCode(result.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}

	if err := svc.ConfirmEnableMFA(user.ID, result.Secret, code); err != nil {
		t.Fatalf("confirm MFA: %v", err)
	}

	updated, _ := svc.GetUser(user.ID)
	if err := svc.VerifyTOTP(updated, code); err != nil {
		t.Fatalf("verify TOTP: %v", err)
	}

	if err := svc.DisableMFA(user.ID, "SecurePass1!", code); err != nil {
		t.Fatalf("disable MFA: %v", err)
	}

	final, _ := svc.GetUser(user.ID)
	if final.MFAEnabled {
		t.Fatal("MFA should be disabled")
	}
}

func TestResetPasswordRequiresLockedAccount(t *testing.T) {
	svc, s := testService(t, 5)
	defer s.Close()
	registerUser(t, svc)

	_, err := svc.RequestPasswordReset("test@example.com")
	if err != auth.ErrAccountNotLocked {
		t.Fatalf("expected not locked error, got %v", err)
	}
}
