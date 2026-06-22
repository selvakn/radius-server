package web

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const sessionTTL = 8 * time.Hour

type Session struct {
	AdminUsername string
	ExpiresAt     time.Time
	CSRFToken     string
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewSessionStore() *SessionStore {
	s := &SessionStore{sessions: make(map[string]Session)}
	go s.cleanup()
	return s
}

func (s *SessionStore) Create(username string) string {
	token := randomHex(32)
	csrf := randomHex(16)
	s.mu.Lock()
	s.sessions[token] = Session{
		AdminUsername: username,
		ExpiresAt:     time.Now().Add(sessionTTL),
		CSRFToken:     csrf,
	}
	s.mu.Unlock()
	return token
}

func (s *SessionStore) Get(token string) (*Session, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return &sess, true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for token, sess := range s.sessions {
			if now.After(sess.ExpiresAt) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
