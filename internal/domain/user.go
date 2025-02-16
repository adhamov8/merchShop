package domain

import "time"

type User struct {
	ID           int
	Username     string
	PasswordHash string
	Coins        int
}

type UserInventory struct {
	ID        int
	UserID    int
	ItemName  string
	Quantity  int
	CreatedAt time.Time
}
