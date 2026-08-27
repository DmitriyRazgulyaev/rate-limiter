package rate_limiter

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

type RateLimiter interface {
	Check(ctx context.Context, ip string) (bool, error)
}

func getIPFromRequest(r *http.Request, proxiesAmount int) (string, error) {
	if proxiesAmount > 0 {
		ips := r.Header.Get("X-Forwarded-For")

		if ips != "" {
			addresses := strings.Split(ips, ",")

			if len(addresses) < proxiesAmount {
				return "", fmt.Errorf(
					"invalid X-Forwarded-For: got %d addresses, expected at least %d",
					len(addresses),
					proxiesAmount,
				)
			}

			ip := strings.TrimSpace(addresses[len(addresses)-proxiesAmount])
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				return "", fmt.Errorf("invalid IP in X-Forwarded-For: %q", ip)
			}

			return parsedIP.String(), nil
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("invalid RemoteAddr: %w", err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return "", fmt.Errorf("invalid IP in RemoteAddr: %q", host)
	}

	return ip.String(), nil
}

func Middleware(limiter RateLimiter, proxiesAmount int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userIP, err := getIPFromRequest(r, proxiesAmount)

			allowed, err := limiter.Check(r.Context(), userIP)
			if err != nil {
				log.Printf("internal error: %w", err)
				http.Error(w, "internal error: %w", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "too many request", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
