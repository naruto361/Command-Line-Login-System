package session_test

import (
	"testing"
	"time"

	"github.com/naruto361/command-line-login-system/internal/session"
	"github.com/naruto361/command-line-login-system/internal/store"
)

func testUser() *store.User {
	return &store.User{
		ID:       1,
		Username: "alice",
		Email:    "alice@example.com",
	}
}

func TestLoginLogout(t *testing.T) {
	m := session.NewManager(time.Minute, 5*time.Minute)

	if m.IsLoggedIn() {
		t.Fatal("expected no session before login")
	}

	m.Login(testUser(), nil)
	if !m.IsLoggedIn() {
		t.Fatal("expected logged in after login")
	}

	cur := m.Current()
	if cur == nil || cur.User.Username != "alice" {
		t.Fatalf("unexpected session: %+v", cur)
	}
	if cur.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatal("expected future expiry")
	}

	m.Logout()
	if m.IsLoggedIn() {
		t.Fatal("expected logged out")
	}
	if m.Current() != nil {
		t.Fatal("expected nil session after logout")
	}
}

func TestPreviousLastLogin(t *testing.T) {
	m := session.NewManager(time.Minute, 5*time.Minute)

	m.Login(testUser(), nil)
	if got := m.Current().PreviousLastLogin; got != nil {
		t.Fatalf("expected nil previous login, got %v", got)
	}
	m.Logout()

	prev := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	m.Login(testUser(), &prev)

	got := m.Current().PreviousLastLogin
	if got == nil {
		t.Fatal("expected previous login time")
	}
	if !got.Equal(prev) {
		t.Fatalf("previous login = %v, want %v", got, prev)
	}

	prev = prev.Add(time.Hour)
	if m.Current().PreviousLastLogin.Equal(prev) {
		t.Fatal("session should store a copy, not share the caller's pointer")
	}
}

func TestRefreshUser(t *testing.T) {
	m := session.NewManager(time.Minute, 5*time.Minute)
	m.Login(testUser(), nil)

	updated := *testUser()
	updated.Username = "bob"
	m.RefreshUser(&updated)

	if m.Current().User.Username != "bob" {
		t.Fatalf("expected refreshed username, got %q", m.Current().User.Username)
	}
}

func TestTouchExtendsSession(t *testing.T) {
	m := session.NewManager(80*time.Millisecond, time.Minute)
	m.Login(testUser(), nil)

	time.Sleep(50 * time.Millisecond)
	m.Touch()
	time.Sleep(50 * time.Millisecond)

	if !m.IsLoggedIn() {
		t.Fatal("touch should extend session past original expiry")
	}
}

func TestCheckExpiryNoSession(t *testing.T) {
	m := session.NewManager(time.Minute, time.Minute)
	if m.CheckExpiry() {
		t.Fatal("CheckExpiry should return false when no session exists")
	}
}

func TestCheckExpiryNotYetExpired(t *testing.T) {
	m := session.NewManager(time.Minute, time.Minute)
	m.Login(testUser(), nil)
	defer m.Logout()

	if m.CheckExpiry() {
		t.Fatal("CheckExpiry should return false before session timeout")
	}
	if !m.IsLoggedIn() {
		t.Fatal("session should remain active")
	}
}

func TestCurrentClearsExpiredSession(t *testing.T) {
	m := session.NewManager(30*time.Millisecond, time.Minute)
	m.Login(testUser(), nil)

	time.Sleep(60 * time.Millisecond)

	if m.Current() != nil {
		t.Fatal("Current should return nil for expired session")
	}
	if m.IsLoggedIn() {
		t.Fatal("IsLoggedIn should be false after expiry")
	}
}

func TestWatcherWarnBeforeExpiry(t *testing.T) {
	// warnBefore > timeout so the warning fires on the first watch tick.
	m := session.NewManager(100*time.Millisecond, 200*time.Millisecond)

	warned := make(chan time.Time, 1)
	m.SetCallbacks(func(expiresAt time.Time) {
		select {
		case warned <- expiresAt:
		default:
		}
	}, nil)

	m.Login(testUser(), nil)
	defer m.Logout()

	select {
	case at := <-warned:
		if at.Before(time.Now().UTC()) {
			t.Fatal("warn callback should receive future expiry time")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected expiry warning before session ended")
	}
}

func TestWatcherAutoLogoutOnExpiry(t *testing.T) {
	m := session.NewManager(50*time.Millisecond, time.Hour)

	expired := make(chan struct{}, 1)
	m.SetCallbacks(nil, func() {
		select {
		case expired <- struct{}{}:
		default:
		}
	})

	m.Login(testUser(), nil)

	select {
	case <-expired:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected auto-logout when session expires")
	}

	if m.IsLoggedIn() {
		t.Fatal("session should be cleared after watcher expiry")
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
