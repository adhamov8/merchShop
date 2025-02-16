package repository

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"

	"merchShop/internal/domain"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(dsn string) (*PostgresRepo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot ping db: %w", err)
	}
	return &PostgresRepo{db: db}, nil
}

func (r *PostgresRepo) CreateUser(ctx context.Context, username, passwordHash string) (int, error) {
	query := `INSERT INTO users (username, password_hash, coins) VALUES ($1, $2, 1000) RETURNING id;`
	var newID int
	if err := r.db.QueryRowContext(ctx, query, username, passwordHash).Scan(&newID); err != nil {
		return 0, errors.Wrap(err, "repo: CreateUser")
	}
	return newID, nil
}

func (r *PostgresRepo) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT id, username, password_hash, coins FROM users WHERE username = $1;`
	row := r.db.QueryRowContext(ctx, query, username)
	u := &domain.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Coins); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "repo: GetUserByUsername")
	}
	return u, nil
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT id, username, password_hash, coins FROM users WHERE id = $1;`
	row := r.db.QueryRowContext(ctx, query, id)
	u := &domain.User{}
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Coins); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "repo: GetUserByID")
	}
	return u, nil
}

func (r *PostgresRepo) UpdateUserCoins(ctx context.Context, userID int, newCoins int) error {
	query := `UPDATE users SET coins = $1 WHERE id = $2;`
	res, err := r.db.ExecContext(ctx, query, newCoins, userID)
	if err != nil {
		return errors.Wrap(err, "repo: UpdateUserCoins")
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("no user updated, user_id=%d not found", userID)
	}
	return nil
}

func (r *PostgresRepo) CreateTransaction(ctx context.Context, fromID, toID, amount int) error {
	query := `INSERT INTO coin_transactions (from_user_id, to_user_id, amount) VALUES ($1, $2, $3);`
	_, err := r.db.ExecContext(ctx, query, fromID, toID, amount)
	if err != nil {
		return errors.Wrap(err, "repo: CreateTransaction")
	}
	return nil
}

func (r *PostgresRepo) ListSentTransactions(ctx context.Context, userID int) ([]domain.CoinTransaction, error) {
	query := `SELECT id, from_user_id, to_user_id, amount, created_at
	          FROM coin_transactions 
			  WHERE from_user_id = $1
	          ORDER BY created_at DESC LIMIT 100;`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "repo: ListSentTransactions")
	}
	defer rows.Close()

	var res []domain.CoinTransaction
	for rows.Next() {
		var t domain.CoinTransaction
		if err := rows.Scan(&t.ID, &t.FromUserID, &t.ToUserID, &t.Amount, &t.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil
}

func (r *PostgresRepo) ListReceivedTransactions(ctx context.Context, userID int) ([]domain.CoinTransaction, error) {
	query := `SELECT id, from_user_id, to_user_id, amount, created_at
	          FROM coin_transactions 
			  WHERE to_user_id = $1
	          ORDER BY created_at DESC LIMIT 100;`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "repo: ListReceivedTransactions")
	}
	defer rows.Close()

	var res []domain.CoinTransaction
	for rows.Next() {
		var t domain.CoinTransaction
		if err := rows.Scan(&t.ID, &t.FromUserID, &t.ToUserID, &t.Amount, &t.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil
}

func (r *PostgresRepo) AddItemToUser(ctx context.Context, userID int, itemName string, qty int) error {
	query := `
        INSERT INTO user_inventory (user_id, item_name, quantity)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, item_name) DO UPDATE
        SET quantity = user_inventory.quantity + EXCLUDED.quantity;
    `
	_, err := r.db.ExecContext(ctx, query, userID, itemName, qty)
	if err != nil {
		return errors.Wrap(err, "repo: AddItemToUser")
	}
	return nil
}

func (r *PostgresRepo) ListUserInventory(ctx context.Context, userID int) ([]domain.UserInventory, error) {
	query := `SELECT id, user_id, item_name, quantity, created_at
	          FROM user_inventory
	          WHERE user_id = $1
	          ORDER BY item_name;`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, errors.Wrap(err, "repo: ListUserInventory")
	}
	defer rows.Close()

	var res []domain.UserInventory
	for rows.Next() {
		var ui domain.UserInventory
		if err := rows.Scan(&ui.ID, &ui.UserID, &ui.ItemName, &ui.Quantity, &ui.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, ui)
	}
	return res, nil
}

func (r *PostgresRepo) TransferCoins(ctx context.Context, fromID, toID, amount int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, "UPDATE users SET coins = coins - $1 WHERE id = $2 AND coins >= $1", amount, fromID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		_ = tx.Rollback()
		return errors.New("insufficient funds or sender not found")
	}
	_, err = tx.ExecContext(ctx, "UPDATE users SET coins = coins + $1 WHERE id = $2", amount, toID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO coin_transactions (from_user_id, to_user_id, amount) VALUES ($1, $2, $3)", fromID, toID, amount)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *PostgresRepo) BuyMerchTx(ctx context.Context, userID int, itemName string, price int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, "UPDATE users SET coins = coins - $1 WHERE id = $2 AND coins >= $1", price, userID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		_ = tx.Rollback()
		return errors.New("insufficient funds or user not found")
	}

	query := `
        INSERT INTO user_inventory (user_id, item_name, quantity)
        VALUES ($1, $2, 1)
        ON CONFLICT (user_id, item_name) DO UPDATE
        SET quantity = user_inventory.quantity + 1;
    `
	_, err = tx.ExecContext(ctx, query, userID, itemName)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
