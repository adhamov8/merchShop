package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"golang.org/x/crypto/bcrypt"
	"merchShop/internal/domain"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrNotEnoughCoins     = errors.New("not enough coins")
	ErrWeakPassword       = errors.New("password does not meet security " +
		"requirements: minimum 8 characters, at least one uppercase letter, one " +
		"lowercase letter, one digit, and one special character")
)

type Repository interface {
	CreateUser(ctx context.Context, username, passwordHash string) (int, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetUserByID(ctx context.Context, id int) (*domain.User, error)
	UpdateUserCoins(ctx context.Context, userID int, newCoins int) error

	CreateTransaction(ctx context.Context, fromID, toID, amount int) error
	ListSentTransactions(ctx context.Context, userID int) ([]domain.CoinTransaction, error)
	ListReceivedTransactions(ctx context.Context, userID int) ([]domain.CoinTransaction, error)

	AddItemToUser(ctx context.Context, userID int, itemName string, qty int) error
	ListUserInventory(ctx context.Context, userID int) ([]domain.UserInventory, error)

	TransferCoins(ctx context.Context, fromID, toID, amount int) error
	BuyMerchTx(ctx context.Context, userID int, itemName string, price int) error
}

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service {
	return &Service{repo: r}
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	if ok, _ := regexp.MatchString("[a-z]", password); !ok {
		return ErrWeakPassword
	}
	if ok, _ := regexp.MatchString("[A-Z]", password); !ok {
		return ErrWeakPassword
	}
	if ok, _ := regexp.MatchString("\\d", password); !ok {
		return ErrWeakPassword
	}
	if ok, _ := regexp.MatchString(`[@$!%*?&]`, password); !ok {
		return ErrWeakPassword
	}

	return nil
}

func (s *Service) RegisterOrLogin(ctx context.Context, username, password string) (*domain.User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		if err := validatePassword(password); err != nil {
			return nil, err
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		newID, err := s.repo.CreateUser(ctx, username, string(hashed))
		if err != nil {
			return nil, err
		}
		return s.repo.GetUserByID(ctx, newID)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Service) SendCoin(ctx context.Context, fromUserID int, toUsername string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	fromUser, err := s.repo.GetUserByID(ctx, fromUserID)
	if err != nil || fromUser == nil {
		return fmt.Errorf("sender user not found")
	}
	toUser, err := s.repo.GetUserByUsername(ctx, toUsername)
	if err != nil || toUser == nil {
		return fmt.Errorf("recipient user not found")
	}
	if fromUser.ID == toUser.ID {
		return fmt.Errorf("cannot send coins to the same user")
	}
	return s.repo.TransferCoins(ctx, fromUser.ID, toUser.ID, amount)
}

func (s *Service) BuyMerch(ctx context.Context, userID int, itemName string) error {
	if !domain.IsValidMerchItem(itemName) {
		return fmt.Errorf("unknown item %s", itemName)
	}
	price := domain.GetItemPrice(itemName)
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}
	if user.Coins < price {
		return ErrNotEnoughCoins
	}

	return s.repo.BuyMerchTx(ctx, user.ID, itemName, price)
}

type InfoResponse struct {
	Coins     int `json:"coins"`
	Inventory []struct {
		Type     string `json:"type"`
		Quantity int    `json:"quantity"`
	} `json:"inventory"`
	CoinHistory struct {
		Received []struct {
			FromUser string `json:"fromUser"`
			Amount   int    `json:"amount"`
		} `json:"received"`
		Sent []struct {
			ToUser string `json:"toUser"`
			Amount int    `json:"amount"`
		} `json:"sent"`
	} `json:"coinHistory"`
}

func (s *Service) GetInfo(ctx context.Context, userID int) (*InfoResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	inv, err := s.repo.ListUserInventory(ctx, userID)
	if err != nil {
		return nil, err
	}
	receivedTx, err := s.repo.ListReceivedTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	sentTx, err := s.repo.ListSentTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &InfoResponse{Coins: user.Coins}

	for _, i := range inv {
		resp.Inventory = append(resp.Inventory, struct {
			Type     string `json:"type"`
			Quantity int    `json:"quantity"`
		}{
			Type:     i.ItemName,
			Quantity: i.Quantity,
		})
	}
	for _, tx := range receivedTx {
		fromUser, err := s.repo.GetUserByID(ctx, tx.FromUserID)
		if err != nil || fromUser == nil {
			continue
		}
		resp.CoinHistory.Received = append(resp.CoinHistory.Received, struct {
			FromUser string `json:"fromUser"`
			Amount   int    `json:"amount"`
		}{
			FromUser: fromUser.Username,
			Amount:   tx.Amount,
		})
	}

	for _, tx := range sentTx {
		toUser, err := s.repo.GetUserByID(ctx, tx.ToUserID)
		if err != nil || toUser == nil {
			continue
		}
		resp.CoinHistory.Sent = append(resp.CoinHistory.Sent, struct {
			ToUser string `json:"toUser"`
			Amount int    `json:"amount"`
		}{
			ToUser: toUser.Username,
			Amount: tx.Amount,
		})
	}
	return resp, nil
}
