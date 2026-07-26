package dto

import "time"

type User struct {
	IP              string
	TokensRemaining int
	LastRefill      time.Time
}
