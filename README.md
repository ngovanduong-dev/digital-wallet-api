# Digital Wallet API

A REST API for user authentication, wallet balances, deposits, withdrawals,
transaction history, and wallet-to-wallet transfers.

V1 focuses on the core API flow. Balance updates and transfers are intentionally
non-atomic so that transaction handling and concurrency control can be addressed
as explicit improvements in Phase 2.

## Tech Stack

- Go 1.26
- Chi router
- PostgreSQL
- pgx
- sqlc
- JWT authentication

## Local Setup

### 1. Configure the environment

Copy `.env.example` to `.env` and provide real values:

```env
DATABASE_URL=postgresql://user:password@host:5432/database
JWT_SECRET=replace-with-a-secret-at-least-32-characters-long
PORT=8080
```

### 2. Create the database schema

Run [migrations/001_init.sql](migrations/001_init.sql) against the PostgreSQL
database.

### 3. Start the API

```bash
go run ./cmd/api
```

The API listens on `http://localhost:8080` by default.

## API Endpoints

### Health

- `GET /health`

### Authentication

- `POST /auth/register`

```json
{
  "email": "user@example.com",
  "password": "password123",
  "full_name": "Example User"
}
```

- `POST /auth/login`

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

### Wallet

The following endpoints require an `Authorization: Bearer <token>` header:

- `GET /wallets/me`
- `POST /wallets/deposit`
- `POST /wallets/withdraw`
- `GET /transactions?limit=10&offset=0`

Deposit and withdrawal request:

```json
{
  "amount": 25.5,
  "note": "Example transaction"
}
```

### Transfer

- `POST /transfers`

```json
{
  "recipient_email": "recipient@example.com",
  "amount": 10,
  "note": "Example transfer"
}
```

## Verification

```bash
go test ./...
go build ./cmd/api
```

## Docker

Build the image:

```bash
docker build -t digital-wallet-api .
```

Run it with environment variables:

```bash
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="postgresql://user:password@host:5432/database" \
  -e JWT_SECRET="replace-with-a-secret-at-least-32-characters-long" \
  digital-wallet-api
```

## Railway Deployment

Railway uses the included `Dockerfile` and `railway.toml`.

Configure these variables in the Railway service:

- `DATABASE_URL`
- `JWT_SECRET`
- `PORT` is provided by Railway

Apply `migrations/001_init.sql` to the production database before starting the
service. Railway checks `GET /health` during deployment.

## V1 Limitations

- Deposits and withdrawals update balances and create transaction records in
  separate database operations.
- Transfers perform debit, credit, and transaction-record creation as separate
  operations.
- Concurrent requests can cause race conditions or partial updates.

These limitations are intentional Phase 1 learning targets and should be fixed
with database transactions and row-level locking in Phase 2.
