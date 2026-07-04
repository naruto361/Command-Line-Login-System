package auth_test

import (
	"testing"

	"github.com/naruto361/command-line-login-system/internal/auth"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid", "alice1", false},
		{"too short", "abc", true},
		{"exactly 6", "abcdef", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid", "SecurePass1!", false},
		{"too short", "Short1!", true},
		{"no upper", "securepass1!", true},
		{"no lower", "SECUREPASS1!", true},
		{"no digit", "SecurePass!!", true},
		{"no special", "SecurePass12", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"invalid", true},
		{"user@domain", true},
	}

	for _, tt := range tests {
		err := auth.ValidateEmail(tt.email)
		if (err != nil) != tt.wantErr {
			t.Fatalf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
		}
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "MySecurePass1!"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !auth.CheckPassword(hash, password) {
		t.Fatal("expected password to match hash")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatal("expected wrong password to fail")
	}
}
