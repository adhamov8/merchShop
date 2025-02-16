package domain

import "time"

type CoinTransaction struct {
	ID         int
	FromUserID int
	ToUserID   int
	Amount     int
	CreatedAt  time.Time
}
