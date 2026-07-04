// Password validation and bcrypt hashing helpers.
// bcrypt is one-way: hashes cannot be reversed to recover the original password.
package auth

import (
	"errors"
	"fmt"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var (
	ErrInvalidUsername   = errors.New("username must be at least 6 characters")
	ErrInvalidPassword   = errors.New("password must be at least 10 characters and include uppercase, lowercase, digit, and special character")
	ErrPasswordMismatch  = errors.New("passwords do not match")
	ErrInvalidEmail       = errors.New("invalid email address")
	ErrAccountNotFound    = errors.New("username or email does not exist")
	ErrWrongPassword      = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid username/email or password")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func ValidateUsername(username string) error {
	if len(username) < 6 {
		return ErrInvalidUsername
	}
	return nil
}

func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 10 {
		return ErrInvalidPassword
	}

	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasLower || !hasUpper || !hasDigit || !hasSpecial {
		return ErrInvalidPassword
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
