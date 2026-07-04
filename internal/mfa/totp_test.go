package mfa_test

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/naruto361/command-line-login-system/internal/mfa"
)

func TestGenerateAndValidateTOTP(t *testing.T) {
	secret, url, err := mfa.GenerateSecret("user@example.com", "OSTO")
	if err != nil {
		t.Fatalf("generate secret: %v", err)
	}
	if secret == "" || url == "" {
		t.Fatal("expected secret and url")
	}

	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}

	if !mfa.ValidateCode(secret, code) {
		t.Fatal("expected valid TOTP code")
	}
	if mfa.ValidateCode(secret, "000000") {
		t.Fatal("expected invalid code to fail")
	}
}
