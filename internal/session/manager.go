package session

import (
	"sync"
	"time"

	"github.com/naruto361/command-line-login-system/internal/store"
)

type Session struct {
	User      *store.User
	ExpiresAt time.Time
}

type Manager struct {
	mu       sync.RWMutex
	session  *Session
	timeout  time.Duration
	warnBefore time.Duration
	onExpire func()
	onWarn   func(time.Time)
	stopCh   chan struct{}
}

func NewManager(timeout, warnBefore time.Duration) *Manager {
	return &Manager{
		timeout:    timeout,
		warnBefore: warnBefore,
		stopCh:     make(chan struct{}),
	}
}

func (m *Manager) SetCallbacks(onWarn func(time.Time), onExpire func()) {
	m.onWarn = onWarn
	m.onExpire = onExpire
}

func (m *Manager) Login(user *store.User) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.session = &Session{
		User:      user,
		ExpiresAt: time.Now().UTC().Add(m.timeout),
	}
	m.restartWatcher()
}

func (m *Manager) Logout() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.session = nil
	m.stopWatcher()
}

func (m *Manager) IsLoggedIn() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeSessionLocked() != nil
}

func (m *Manager) Current() *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.activeSessionLocked()
	if s == nil {
		return nil
	}
	copy := *s
	return &copy
}

func (m *Manager) RefreshUser(user *store.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		m.session.User = user
	}
}

func (m *Manager) Touch() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		m.session.ExpiresAt = time.Now().UTC().Add(m.timeout)
		m.restartWatcher()
	}
}

func (m *Manager) activeSessionLocked() *Session {
	if m.session == nil {
		return nil
	}
	if time.Now().UTC().After(m.session.ExpiresAt) {
		m.session = nil
		return nil
	}
	return m.session
}

func (m *Manager) CheckExpiry() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		return false
	}
	if time.Now().UTC().After(m.session.ExpiresAt) {
		m.session = nil
		m.stopWatcher()
		if m.onExpire != nil {
			go m.onExpire()
		}
		return true
	}
	return false
}

func (m *Manager) restartWatcher() {
	m.stopWatcher()
	m.stopCh = make(chan struct{})
	go m.watch()
}

func (m *Manager) stopWatcher() {
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
}

func (m *Manager) watch() {
	stop := m.stopCh
	warned := false

	for {
		m.mu.RLock()
		s := m.session
		onWarn := m.onWarn
		onExpire := m.onExpire
		warnBefore := m.warnBefore
		m.mu.RUnlock()

		if s == nil {
			return
		}

		now := time.Now().UTC()
		remaining := s.ExpiresAt.Sub(now)

		if remaining <= 0 {
			m.mu.Lock()
			if m.session != nil && time.Now().UTC().After(m.session.ExpiresAt) {
				m.session = nil
				m.mu.Unlock()
				if onExpire != nil {
					onExpire()
				}
			} else {
				m.mu.Unlock()
			}
			return
		}

		if !warned && remaining <= warnBefore {
			warned = true
			if onWarn != nil {
				onWarn(s.ExpiresAt)
			}
		}

		sleep := time.Minute
		if remaining < sleep {
			sleep = remaining
		}

		select {
		case <-stop:
			return
		case <-time.After(sleep):
		}
	}
}
