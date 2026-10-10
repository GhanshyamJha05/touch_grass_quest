package app

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// Cleanup routine
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, times := range rl.visitors {
		var valid []time.Time
		for _, t := range times {
			if now.Sub(t) < rl.window {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.visitors, ip)
		} else {
			rl.visitors[ip] = valid
		}
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	times, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = []time.Time{now}
		return true
	}

	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}
	
	if len(valid) >= rl.limit {
		rl.visitors[ip] = valid
		return false
	}
	
	valid = append(valid, now)
	rl.visitors[ip] = valid
	return true
}

func GetIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}
