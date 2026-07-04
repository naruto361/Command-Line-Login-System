package store

import "time"

type User struct {
	ID                   int64
	Username             string
	Email                string
	PasswordHash         string
	MFAEnabled           bool
	TOTPSecret           *string
	FailedLoginAttempts  int
	AccountLocked        bool
	PasswordResetToken   *string
	PasswordResetExpiry  *time.Time
	RegisteredAt         time.Time
	LastLogin            *time.Time
}
