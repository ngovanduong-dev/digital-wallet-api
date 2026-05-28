package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ngovanduong-dev/digital-wallet-api/internal/db"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type AuthService struct {
	queries   *db.Queries
	jwtSecret string
}

func NewAuthService(queries *db.Queries, jwtSecret string) *AuthService {
	return &AuthService{queries: queries, jwtSecret: jwtSecret}
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required"`
}

type RegisterResponse struct {
	ID       pgtype.UUID `json:"id"`
	Email    string      `json:"email"`
	FullName string      `json:"full_name"`
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	_, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check existing email: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:    req.Email,
		Password: string(hashedPassword),
		FullName: req.FullName,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	_, err = s.queries.CreateWallet(ctx, db.CreateWalletParams{
		UserID:   user.ID,
		Currency: "USD",
	})
	if err != nil {
		return nil, fmt.Errorf("create wallet for user: %w", err)
	}

	return &RegisterResponse{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
	}, nil
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := s.queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	uid := uuid.UUID(user.ID.Bytes)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uid.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("sign jwt token: %w", err)
	}

	return &LoginResponse{Token: tokenString}, nil
}
