package api

import (
	"sync"
	"time"
)

const (
	loginWindow           = 5 * time.Minute
	loginMaxAttempts      = 10
	loginMaxTrackedEmails = 10000
)

type loginLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	window   time.Duration
	max      int
	attempts map[string][]time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		now:      time.Now,
		window:   loginWindow,
		max:      loginMaxAttempts,
		attempts: make(map[string][]time.Time),
	}
}

func (l *loginLimiter) Allow(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts[email] = pruneAttempts(l.attempts[email], l.now().Add(-l.window))
	if len(l.attempts[email]) == 0 {
		delete(l.attempts, email)
	}
	return len(l.attempts[email]) < l.max
}

func (l *loginLimiter) Fail(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.attempts) >= loginMaxTrackedEmails {
		l.sweepLocked()
	}
	if len(l.attempts) >= loginMaxTrackedEmails {
		return
	}
	l.attempts[email] = append(l.attempts[email], l.now())
}

func (l *loginLimiter) sweepLocked() {
	cutoff := l.now().Add(-l.window)
	for email, attempts := range l.attempts {
		pruned := pruneAttempts(attempts, cutoff)
		if len(pruned) == 0 {
			delete(l.attempts, email)
			continue
		}
		l.attempts[email] = pruned
	}
}

func (l *loginLimiter) Reset(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, email)
}

func pruneAttempts(attempts []time.Time, cutoff time.Time) []time.Time {
	kept := attempts[:0]
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}
	return kept
}
