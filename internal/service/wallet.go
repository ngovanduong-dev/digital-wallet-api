package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/db"
)

var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type WalletService struct {
	queries *db.Queries
}

func NewWalletService(queries *db.Queries) *WalletService {
	return &WalletService{queries: queries}
}

func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

type WalletResponse struct {
	ID        pgtype.UUID `json:"id"`
	Balance   float64     `json:"balance"`
	Currency  string      `json:"currency"`
	CreatedAt time.Time   `json:"created_at"`
}

type TransactionResponse struct {
	ID           pgtype.UUID `json:"id"`
	Type         string      `json:"type"`
	Amount       float64     `json:"amount"`
	BalanceAfter float64     `json:"balance_after"`
	Note         *string     `json:"note"`
	CreatedAt    time.Time   `json:"created_at"`
}

func toTransactionResponse(tx db.Transaction) TransactionResponse {
	var note *string
	if tx.Note.Valid {
		note = &tx.Note.String
	}
	return TransactionResponse{
		ID:           tx.ID,
		Type:         tx.Type,
		Amount:       float64(tx.Amount) / 100,
		BalanceAfter: float64(tx.BalanceAfter) / 100,
		Note:         note,
		CreatedAt:    tx.CreatedAt.Time,
	}
}

func (s *WalletService) GetWallet(ctx context.Context, userID uuid.UUID) (*WalletResponse, error) {
	wallet, err := s.queries.GetWalletByUserID(ctx, db.GetWalletByUserIDParams{
		UserID:   toPgtypeUUID(userID),
		Currency: "USD",
	})
	if err != nil {
		return nil, ErrWalletNotFound
	}

	return &WalletResponse{
		ID:        wallet.ID,
		Balance:   float64(wallet.Balance) / 100,
		Currency:  wallet.Currency,
		CreatedAt: wallet.CreatedAt.Time,
	}, nil
}

type DepositRequest struct {
	Amount float64 `json:"amount" validate:"required,gt=0"`
	Note   string  `json:"note"`
}

func (s *WalletService) Deposit(ctx context.Context, userID uuid.UUID, req DepositRequest) (*TransactionResponse, error) {
	wallet, err := s.queries.GetWalletByUserID(ctx, db.GetWalletByUserIDParams{
		UserID:   toPgtypeUUID(userID),
		Currency: "USD",
	})
	if err != nil {
		return nil, ErrWalletNotFound
	}

	amountCents := int64(math.Round(req.Amount * 100))
	newBalance := wallet.Balance + amountCents

	updatedWallet, err := s.queries.UpdateWalletBalance(ctx, db.UpdateWalletBalanceParams{
		Balance: newBalance,
		ID:      wallet.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	noteText := pgtype.Text{}
	if req.Note != "" {
		noteText = pgtype.Text{String: req.Note, Valid: true}
	}

	tx, err := s.queries.CreateTransaction(ctx, db.CreateTransactionParams{
		WalletID:     wallet.ID,
		Type:         "deposit",
		Amount:       amountCents,
		BalanceAfter: updatedWallet.Balance,
		ReferenceID:  pgtype.UUID{Valid: false},
		Note:         noteText,
	})
	if err != nil {
		return nil, fmt.Errorf("create transaction record: %w", err)
	}

	res := toTransactionResponse(tx)
	return &res, nil
}

type WithdrawRequest struct {
	Amount float64 `json:"amount" validate:"required,gt=0"`
	Note   string  `json:"note"`
}

func (s *WalletService) Withdraw(ctx context.Context, userID uuid.UUID, req WithdrawRequest) (*TransactionResponse, error) {
	wallet, err := s.queries.GetWalletByUserID(ctx, db.GetWalletByUserIDParams{
		UserID:   toPgtypeUUID(userID),
		Currency: "USD",
	})
	if err != nil {
		return nil, ErrWalletNotFound
	}

	amountCents := int64(math.Round(req.Amount * 100))

	if wallet.Balance < amountCents {
		return nil, ErrInsufficientFunds
	}

	newBalance := wallet.Balance - amountCents

	updatedWallet, err := s.queries.UpdateWalletBalance(ctx, db.UpdateWalletBalanceParams{
		Balance: newBalance,
		ID:      wallet.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	noteText := pgtype.Text{}
	if req.Note != "" {
		noteText = pgtype.Text{String: req.Note, Valid: true}
	}

	tx, err := s.queries.CreateTransaction(ctx, db.CreateTransactionParams{
		WalletID:     wallet.ID,
		Type:         "withdrawal",
		Amount:       amountCents,
		BalanceAfter: updatedWallet.Balance,
		ReferenceID:  pgtype.UUID{Valid: false},
		Note:         noteText,
	})
	if err != nil {
		return nil, fmt.Errorf("create transaction record: %w", err)
	}

	res := toTransactionResponse(tx)
	return &res, nil
}

type ListTransactionsResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
}

func (s *WalletService) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int32) (*ListTransactionsResponse, error) {
	wallet, err := s.queries.GetWalletByUserID(ctx, db.GetWalletByUserIDParams{
		UserID:   toPgtypeUUID(userID),
		Currency: "USD",
	})
	if err != nil {
		return nil, ErrWalletNotFound
	}

	txs, err := s.queries.ListTransactionsByWalletID(ctx, db.ListTransactionsByWalletIDParams{
		WalletID: wallet.ID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}

	result := make([]TransactionResponse, len(txs))
	for i, tx := range txs {
		result[i] = toTransactionResponse(tx)
	}

	return &ListTransactionsResponse{Transactions: result}, nil
}
