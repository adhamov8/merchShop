package usecase

import (
	"context"
	"errors"
	"testing"

	"merchShop/internal/domain"

	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	users        map[int]*domain.User
	usersByName  map[string]*domain.User
	inventory    []domain.UserInventory
	transactions []domain.CoinTransaction
	lastUserID   int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:        make(map[int]*domain.User),
		usersByName:  make(map[string]*domain.User),
		inventory:    []domain.UserInventory{},
		transactions: []domain.CoinTransaction{},
	}
}

func (m *mockRepo) CreateUser(ctx context.Context, username, passwordHash string) (int, error) {
	m.lastUserID++
	newUser := &domain.User{
		ID:           m.lastUserID,
		Username:     username,
		PasswordHash: passwordHash,
		Coins:        1000,
	}
	m.users[newUser.ID] = newUser
	m.usersByName[newUser.Username] = newUser
	return newUser.ID, nil
}

func (m *mockRepo) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	if user, ok := m.usersByName[username]; ok {
		return user, nil
	}
	return nil, nil
}

func (m *mockRepo) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, nil
}

func (m *mockRepo) UpdateUserCoins(ctx context.Context, userID int, newCoins int) error {
	user, ok := m.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	user.Coins = newCoins
	return nil
}

func (m *mockRepo) CreateTransaction(ctx context.Context, fromID, toID, amount int) error {
	m.transactions = append(m.transactions, domain.CoinTransaction{
		ID:         len(m.transactions) + 1,
		FromUserID: fromID,
		ToUserID:   toID,
		Amount:     amount,
	})
	return nil
}

func (m *mockRepo) ListSentTransactions(ctx context.Context, userID int) ([]domain.CoinTransaction, error) {
	var result []domain.CoinTransaction
	for _, t := range m.transactions {
		if t.FromUserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockRepo) ListReceivedTransactions(ctx context.Context, userID int) ([]domain.CoinTransaction, error) {
	var result []domain.CoinTransaction
	for _, t := range m.transactions {
		if t.ToUserID == userID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockRepo) AddItemToUser(ctx context.Context, userID int, itemName string, qty int) error {
	found := false
	for i, inv := range m.inventory {
		if inv.UserID == userID && inv.ItemName == itemName {
			m.inventory[i].Quantity += qty
			found = true
			break
		}
	}
	if !found {
		m.inventory = append(m.inventory, domain.UserInventory{
			ID:       len(m.inventory) + 1,
			UserID:   userID,
			ItemName: itemName,
			Quantity: qty,
		})
	}
	return nil
}

func (m *mockRepo) ListUserInventory(ctx context.Context, userID int) ([]domain.UserInventory, error) {
	var result []domain.UserInventory
	for _, inv := range m.inventory {
		if inv.UserID == userID {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (m *mockRepo) TransferCoins(ctx context.Context, fromID, toID, amount int) error {
	fromUser, ok := m.users[fromID]
	if !ok {
		return errors.New("sender not found")
	}
	toUser, ok2 := m.users[toID]
	if !ok2 {
		return errors.New("recipient not found")
	}
	if fromUser.Coins < amount {
		return errors.New("insufficient funds")
	}
	if fromID == toID {
		return errors.New("you can't send to yourself")
	}
	fromUser.Coins -= amount
	toUser.Coins += amount
	m.transactions = append(m.transactions, domain.CoinTransaction{
		ID:         len(m.transactions) + 1,
		FromUserID: fromID,
		ToUserID:   toID,
		Amount:     amount,
	})
	return nil
}

func (m *mockRepo) BuyMerchTx(ctx context.Context, userID int, itemName string, price int) error {
	user, ok := m.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	if user.Coins < price {
		return errors.New("insufficient funds")
	}
	user.Coins -= price
	found := false
	for i, inv := range m.inventory {
		if inv.UserID == userID && inv.ItemName == itemName {
			m.inventory[i].Quantity++
			found = true
			break
		}
	}
	if !found {
		m.inventory = append(m.inventory, domain.UserInventory{
			ID:       len(m.inventory) + 1,
			UserID:   userID,
			ItemName: itemName,
			Quantity: 1,
		})
	}
	return nil
}

func TestService_RegisterOrLogin(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	svc := NewService(mock)

	u, err := svc.RegisterOrLogin(ctx, "Ziyo", "Strong@Pass123")
	assert.NoError(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, "Ziyo", u.Username)
	assert.Equal(t, 1000, u.Coins)

	assert.NotEqual(t, "Strong@Pass123", u.PasswordHash)

	_, err = svc.RegisterOrLogin(ctx, "Ali", "password") // слишком слабый
	assert.Error(t, err)
	assert.Equal(t, ErrWeakPassword, err)

	u2, err := svc.RegisterOrLogin(ctx, "Ali", "Strong@Pass123")
	assert.NoError(t, err)
	assert.Equal(t, "Ali", u2.Username)
	assert.Equal(t, 1000, u2.Coins)

	u2Again, err := svc.RegisterOrLogin(ctx, "Ali", "Strong@Pass123")
	assert.NoError(t, err)
	assert.Equal(t, u2.ID, u2Again.ID, "must match ID")
}

func TestService_SendCoin(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	svc := NewService(mock)

	ziyo, _ := svc.RegisterOrLogin(ctx, "Ziyo", "Strong@Pass123")
	ali, _ := svc.RegisterOrLogin(ctx, "Ali", "Strong@Pass123")

	err := svc.SendCoin(ctx, ziyo.ID, "Ali", 200)
	assert.NoError(t, err)

	ziyoUpdated, _ := mock.GetUserByID(ctx, ziyo.ID)
	aliUpdated, _ := mock.GetUserByID(ctx, ali.ID)

	assert.Equal(t, 800, ziyoUpdated.Coins, "Ziyo = 1000 - 200")
	assert.Equal(t, 1200, aliUpdated.Coins, "Ali = 1000 + 200")

	err = svc.SendCoin(ctx, ziyo.ID, "Ali", 900)
	assert.Error(t, err, "insufficient funds expected")

	err = svc.SendCoin(ctx, ziyo.ID, "Ziyo", 100)
	assert.Error(t, err, "you can't send to yourself")
}

func TestService_BuyMerch(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	svc := NewService(mock)

	user, _ := svc.RegisterOrLogin(ctx, "TestUser", "Valid@Pass123")
	err := svc.BuyMerch(ctx, user.ID, "book")
	assert.NoError(t, err)

	userAfter, _ := mock.GetUserByID(ctx, user.ID)
	assert.Equal(t, 950, userAfter.Coins)

	err = svc.BuyMerch(ctx, user.ID, "someUnknownItem")
	assert.Error(t, err)

	err = svc.BuyMerch(ctx, user.ID, "pink-hoody")
	assert.NoError(t, err)
	userAfter2, _ := mock.GetUserByID(ctx, user.ID)
	assert.Equal(t, 450, userAfter2.Coins, "950 - 500")

	err = svc.BuyMerch(ctx, user.ID, "pink-hoody")
	assert.Error(t, err, "not enough coins")
}

func TestService_GetInfo(t *testing.T) {
	ctx := context.Background()
	mock := newMockRepo()
	svc := NewService(mock)

	ziyo, _ := svc.RegisterOrLogin(ctx, "Ziyo", "Valid@Pass123")
	ali, _ := svc.RegisterOrLogin(ctx, "Ali", "Valid@Pass123")

	_ = svc.BuyMerch(ctx, ziyo.ID, "book")
	_ = svc.BuyMerch(ctx, ziyo.ID, "socks")

	_ = svc.SendCoin(ctx, ziyo.ID, "Ali", 100)

	respZiyo, err := svc.GetInfo(ctx, ziyo.ID)
	assert.NoError(t, err)
	assert.Equal(t, 840, respZiyo.Coins)
	assert.Len(t, respZiyo.Inventory, 2) // book, socks

	respAli, err := svc.GetInfo(ctx, ali.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1100, respAli.Coins)
	assert.Len(t, respAli.Inventory, 0)
	assert.Len(t, respAli.CoinHistory.Received, 1, "one incoming tx from Ziyo")
	assert.Len(t, respAli.CoinHistory.Sent, 0)
}
