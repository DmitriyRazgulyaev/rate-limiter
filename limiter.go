package rate_limiter

import (
	"context"
	"rate-limiter/dto"
	"time"
)

type Repository interface {
	GetUser(ctx context.Context, ip string, TTL time.Duration, MaxTokens int) (dto.User, error)
	SetUser(ctx context.Context, ip string, tokensRemaining int, lastRefill time.Time, TTL time.Duration) (dto.User, error)
}

type Limiter struct {
	repo           Repository
	TTL            time.Duration
	MaxTokens      int
	RefillInterval int
}

func NewRateLimiter(repository Repository, TTL time.Duration, MaxTokens int, RefillInterval int) *Limiter {
	return &Limiter{
		repo:           repository,
		TTL:            TTL,
		MaxTokens:      MaxTokens,
		RefillInterval: RefillInterval,
	}
}

func (rl *Limiter) Check(ctx context.Context, ip string) (bool, error) {
	user, err := rl.repo.GetUser(ctx, ip, rl.TTL, rl.MaxTokens)
	if err != nil {
		return false, err
	}

	currentTime := time.Now()
	elapsed := int(currentTime.Sub(user.LastRefill).Seconds())
	tokensToAdd := elapsed / rl.RefillInterval
	newTokensRemaining := min(rl.MaxTokens, user.TokensRemaining+tokensToAdd)
	if newTokensRemaining == 0 {
		return false, nil
	}

	if tokensToAdd > 0 {
		user.LastRefill = user.LastRefill.Add(time.Duration(tokensToAdd*rl.RefillInterval) * time.Second)
	}

	user.TokensRemaining = newTokensRemaining - 1
	_, err = rl.repo.SetUser(ctx, user.IP, user.TokensRemaining, user.LastRefill, rl.TTL)
	if err != nil {
		return false, err
	}

	return true, nil
}
