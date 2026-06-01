package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/db"
)

var (
	ErrSelfTransfer      = errors.New("cannot transfer to yourself")
	ErrRecipientNotFound = errors.New("recipient not found")
)

type TransferService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewTransferService(queries *db.Queries, pool *pgxpool.Pool) *TransferService {
	return &TransferService{queries: queries, pool: pool}
}

type TransferRequest struct {
	RecipientEmail string  `json:"recipient_email" validate:"required,email"`
	Amount         float64 `json:"amount" validate:"required,gt=0"`
	Note           string  `json:"note"`
}

type TransferResponse struct {
	SenderTransaction    TransactionResponse `json:"sender_transaction"`
	RecipientTransaction TransactionResponse `json:"recipient_transaction"`
}

func (s *TransferService) Transfer(ctx context.Context, senderUserID uuid.UUID, req TransferRequest) (*TransferResponse, error) {
	senderWallet, err := s.queries.GetWalletByUserID(ctx, db.GetWalletByUserIDParams{
		UserID:   toPgtypeUUID(senderUserID),
		Currency: "USD",
	})
	if err != nil {
		return nil, ErrWalletNotFound
	}

	recipientUser, err := s.queries.GetUserByEmail(ctx, req.RecipientEmail)
	if err != nil {
		return nil, ErrRecipientNotFound
	}

	recipientUserID := uuid.UUID(recipientUser.ID.Bytes)
	if recipientUserID == senderUserID {
		return nil, ErrSelfTransfer
	}

	recipientWallet, err := s.queries.GetWalletByUserID(ctx, db.GetWalletByUserIDParams{
		UserID:   recipientUser.ID,
		Currency: "USD",
	})
	if err != nil {
		return nil, ErrRecipientNotFound
	}

	amountCents := int64(math.Round(req.Amount * 100))
	if senderWallet.Balance < amountCents {
		return nil, ErrInsufficientFunds
	}

	noteText := pgtype.Text{}
	if req.Note != "" {
		noteText = pgtype.Text{String: req.Note, Valid: true}
	}

	// V1 is intentionally non-atomic. Phase 2 will wrap this flow in a DB
	// transaction with row-level locking to avoid partial updates and races.
	newSenderBalance := senderWallet.Balance - amountCents
	updatedSenderWallet, err := s.queries.UpdateWalletBalance(ctx, db.UpdateWalletBalanceParams{
		Balance: newSenderBalance,
		ID:      senderWallet.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("deduct sender balance: %w", err)
	}

	newRecipientBalance := recipientWallet.Balance + amountCents
	updatedRecipientWallet, err := s.queries.UpdateWalletBalance(ctx, db.UpdateWalletBalanceParams{
		Balance: newRecipientBalance,
		ID:      recipientWallet.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("add recipient balance: %w", err)
	}

	refID := uuid.New()
	refPgtype := pgtype.UUID{Bytes: [16]byte(refID), Valid: true}

	senderTx, err := s.queries.CreateTransaction(ctx, db.CreateTransactionParams{
		WalletID:     senderWallet.ID,
		Type:         "transfer_out",
		Amount:       amountCents,
		BalanceAfter: updatedSenderWallet.Balance,
		ReferenceID:  refPgtype,
		Note:         noteText,
	})
	if err != nil {
		return nil, fmt.Errorf("create sender transaction record: %w", err)
	}

	recipientTx, err := s.queries.CreateTransaction(ctx, db.CreateTransactionParams{
		WalletID:     recipientWallet.ID,
		Type:         "transfer_in",
		Amount:       amountCents,
		BalanceAfter: updatedRecipientWallet.Balance,
		ReferenceID:  refPgtype,
		Note:         noteText,
	})
	if err != nil {
		return nil, fmt.Errorf("create recipient transaction record: %w", err)
	}

	return &TransferResponse{
		SenderTransaction:    toTransactionResponse(senderTx),
		RecipientTransaction: toTransactionResponse(recipientTx),
	}, nil
}
