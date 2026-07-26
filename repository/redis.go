package repository

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"rate-limiter/dto"
	"strconv"
	"time"
)

const (
	IPKey              = "IP"
	tokensRemainingKey = "tokensRemaining"
	lastRefillKey      = "lastRefill"
)

type RedisStorage struct {
	Client *redis.Client
}

func NewRedisStorage(client *redis.Client) *RedisStorage {
	return &RedisStorage{
		Client: client,
	}
}

func (rs *RedisStorage) GetUser(ctx context.Context, ip string, TTL time.Duration, maxTokens int) (dto.User, error) {
	values, err := rs.Client.HGetAll(ctx, ip).Result()
	if err != nil {
		return dto.User{}, fmt.Errorf("GetUser.Client.HGetAll: %w", err)
	}
	if len(values) == 0 {
		user, err := rs.SetUser(ctx, ip, maxTokens, time.Now(), TTL)
		if err != nil {
			return dto.User{}, err
		}
		return user, nil
	}

	user, err := mapUserFields(values)
	if err != nil {
		return dto.User{}, fmt.Errorf("GetUser.mapUserFlields: %w", err)
	}

	return user, nil
}

func (rs *RedisStorage) SetUser(ctx context.Context, ip string, tokensRemaining int, lastRefill time.Time, TTL time.Duration) (dto.User, error) {
	user := dto.User{
		IP:              ip,
		TokensRemaining: tokensRemaining,
		LastRefill:      lastRefill,
	}
	err := rs.Client.HSet(
		ctx,
		user.IP,
		IPKey, user.IP,
		tokensRemainingKey, user.TokensRemaining,
		lastRefillKey, user.LastRefill,
	).Err()
	if err != nil {
		return user, fmt.Errorf("SetUser.Client.HSet: %w", err)
	}
	//TODO: сделать lua-скрипт для транзакции создания данных в redis и установки времени жизни
	if err := rs.Client.Expire(ctx, ip, TTL).Err(); err != nil {
		return dto.User{}, fmt.Errorf("SetUser.Client.Expire: %w", err)
	}
	return user, nil
}

func mapUserFields(values map[string]string) (dto.User, error) {
	var user dto.User
	user.IP = values[IPKey]
	tr, err := strconv.Atoi(values[tokensRemainingKey])
	if err != nil {
		return dto.User{}, err
	}
	user.TokensRemaining = tr
	lr, err := time.Parse(time.RFC3339Nano, values[lastRefillKey])
	if err != nil {
		return dto.User{}, err
	}
	user.LastRefill = lr

	return user, nil
}
