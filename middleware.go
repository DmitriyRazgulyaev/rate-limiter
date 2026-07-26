package rate_limiter

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
)

type RateLimiter interface {
	Check(ctx context.Context, ip string) (bool, error)
}

func Middleware(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotAcceptable)
			}
			userIP := net.ParseIP(ip)
			if userIP == nil {
				//TODO: проверить правильность кода ошибки
				http.Error(w, fmt.Sprintf("userip: %q is not IP:port", r.RemoteAddr), http.StatusBadRequest)
			}

			isAllow, err := limiter.Check(context.Background(), userIP.String())
			if err != nil {
				log.Println(fmt.Sprintf("internal error: %w", err))
				//TODO: проверить правильность ошибки
				http.Error(w, fmt.Sprintf("internal error: %w", err), http.StatusInternalServerError)
				return
			}
			if !isAllow {
				http.Error(w, "too many request", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
