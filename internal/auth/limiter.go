package auth

import (
	"sync"
	"time"
)

// LoginLimiter limits failed logins per key (IP + email).
type LoginLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	m      map[string]*attempts
}

type attempts struct {
	count int
	first time.Time
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{max: max, window: window, m: make(map[string]*attempts)}
}

// Allowed reports whether attempts are still allowed for key.
func (l *LoginLimiter) Allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.m[key]
	if a == nil {
		return true
	}
	if time.Since(a.first) > l.window {
		delete(l.m, key)
		return true
	}
	return a.count < l.max
}

func (l *LoginLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.m) > 10_000 { // limit memory: clean up expired entries
		for k, a := range l.m {
			if time.Since(a.first) > l.window {
				delete(l.m, k)
			}
		}
	}
	a := l.m[key]
	if a == nil || time.Since(a.first) > l.window {
		l.m[key] = &attempts{count: 1, first: time.Now()}
		return
	}
	a.count++
}

func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	delete(l.m, key)
	l.mu.Unlock()
}
