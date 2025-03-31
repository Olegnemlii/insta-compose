package repository

import (
	"auth/config"
	"auth/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type UserRepository struct {
	db  *sql.DB
	cfg config.VerificationConfig
}

type TokenRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB, cfg config.VerificationConfig) *UserRepository {
	return &UserRepository{db: db, cfg: cfg}
}

func (u *UserRepository) CreateUser(ctx context.Context, email, hashedPassword string) (uuid.UUID, error) {
	_, err := u.GetUserByEmail(ctx, email)
	if err == nil {
		return uuid.Nil, fmt.Errorf("user with email %s already exists", email)
	}

	id := uuid.New()
	currentTime := time.Now()

	insertUserSQL := `
        INSERT INTO users (id, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `

	var idStr string

	if err := u.db.QueryRowContext(ctx, insertUserSQL, id, email, hashedPassword, currentTime, currentTime).Scan(&idStr); err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return uuid.MustParse(idStr), nil
}

func (u *UserRepository) CreateVerificationCodeData(ctx context.Context, code string, userID uuid.UUID, signature string) error {
	createSignatureSQL := `
        INSERT INTO codes_signatures (code, signature, user_id, expires_at)
        VALUES ($1, $2, $3, $4)
    `

	durationStr := u.cfg.CodeExpireTime
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("incorrect time format for CodeExpireTime: %w", err)
	}

	codeExpirationTime := time.Now().Add(duration)

	if _, err := u.db.ExecContext(ctx, createSignatureSQL, code, signature, userID, codeExpirationTime); err != nil {
		return fmt.Errorf("failed to create verification code %w", err)
	}

	return nil
}

func (u *UserRepository) GetEmailBySignature(ctx context.Context, signature string) (string, error) {
	query := `
		SELECT u.email
		FROM users u
		JOIN codes_signatures cs ON u.id = cs.user_id
		WHERE cs.signature = $1
	`

	row := u.db.QueryRowContext(ctx, query, signature)

	var email string
	err := row.Scan(&email)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user not found with signature: %s", signature)
		}
		return "", fmt.Errorf("failed to scan email: %w", err)
	}

	return email, nil
}

func (u *UserRepository) CheckVerificationCodeByUserId(ctx context.Context, code string, signature string, userId string) error {
	query := `
SELECT u.id 
FROM users u 
JOIN codes_signatures cs ON u.id = cs.user_id 
WHERE  cs.code = $1 AND cs.signature = $2
	`

	row := u.db.QueryRowContext(ctx, query, userId, code, signature)

	var email string
	err := row.Scan(&email)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("verification code not found or expired for email: %s", email)
		}
		return fmt.Errorf("failed to get verification code: %w", err)
	}

	return nil
}

func (u *UserRepository) ConfirmUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET is_confirmed = TRUE
		WHERE id = $1
	`
	_, err := u.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to confirm user: %w", err)
	}
	return nil
}

func (u *UserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password
		FROM users
		WHERE email = $1
	`
	row := u.db.QueryRowContext(ctx, query, email)

	var user domain.User
	var idStr string
	err := row.Scan(&idStr, &user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found with email: %s", email)
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	user.ID = uuid.MustParse(idStr)
	return &user, nil
}

func (u *UserRepository) NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

func (t *TokenRepository) StoreRefreshToken(ctx context.Context, email string, refreshToken string) error {
	log.Printf("save refresh token in database from email: %s", email)

	var userID uuid.UUID
	query := `SELECT id FROM users WHERE email = $1`
	err := t.db.QueryRowContext(ctx, query, email).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}

	query = `
        INSERT INTO tokens (access_token, refresh_token, user_id)
        VALUES ($1, $2, $3)
    `
	_, err = t.db.ExecContext(ctx, query, "", refreshToken, userID.String())
	if err != nil {
		log.Printf("error from saving refresh token in database: %v", err)
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	log.Println("refresh token successfully saved to the database!")
	return nil
}

func (t *TokenRepository) GetRefreshToken(ctx context.Context, email string) (string, error) {
	log.Printf("Getting refresh token in database from email: %s", email)

	var userID uuid.UUID
	query := `SELECT id FROM users WHERE email = $1`
	err := t.db.QueryRowContext(ctx, query, email).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user ID: %w", err)
	}

	query = `
    SELECT refresh_token
    FROM tokens
    WHERE user_id = $1
`

	row := t.db.QueryRowContext(ctx, query, userID)

	var refreshToken string
	err = row.Scan(&refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("refresh token not found for user with id: %s", userID)
		}
		return "", fmt.Errorf("failed to get refresh token: %w", err)
	}

	return refreshToken, nil
}

func (t *TokenRepository) DeleteToken(ctx context.Context, email string) error {
	log.Printf("Deleting refresh token in database from email: %s", email)

	var userID uuid.UUID
	query := `SELECT id FROM users WHERE email = $1`
	err := t.db.QueryRowContext(ctx, query, email).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to get user ID: %w", err)
	}

	query = `
        DELETE FROM tokens
        WHERE user_id = $1
    `

	_, err = t.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}
