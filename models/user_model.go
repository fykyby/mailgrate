package models

import (
	"app/db"
	"context"
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	Id                     int `bun:",pk,autoincrement"`
	Email                  string
	PasswordHash           string
	Confirmed              bool
	ConfirmationTokenHash  string    `bun:",nullzero,default:null"`
	ConfirmationExpiresAt  time.Time `bun:",nullzero,default:null"`
	PasswordResetTokenHash string    `bun:",nullzero,default:null"`
	PasswordResetExpiresAt time.Time `bun:",nullzero,default:null"`
	CreatedAt              time.Time `bun:",nullzero,default:current_timestamp"`
}

func CreateUser(ctx context.Context, email string, passwordHash string, confirmationTokenHash string, confirmationExpiresAt time.Time) (*User, error) {
	user := &User{
		Email:                  email,
		PasswordHash:           passwordHash,
		Confirmed:              false,
		ConfirmationTokenHash:  confirmationTokenHash,
		ConfirmationExpiresAt:  confirmationExpiresAt,
		PasswordResetTokenHash: "",
		PasswordResetExpiresAt: time.Time{},
	}

	_, err := db.Bun.
		NewInsert().
		Model(user).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func FindUserByID(ctx context.Context, id int) (*User, error) {
	user := new(User)

	err := db.Bun.
		NewSelect().
		Model(user).
		Where("id = ?", id).
		Scan(ctx)

	return user, err
}

func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	user := new(User)

	err := db.Bun.
		NewSelect().
		Model(user).
		Where("email = ?", email).
		Scan(ctx)

	return user, err
}

func FindUserByPasswordResetTokenhash(ctx context.Context, tokenHash string) (*User, error) {
	user := new(User)

	err := db.Bun.
		NewSelect().
		Model(user).
		Where("password_reset_token_hash = ?", tokenHash).
		Scan(ctx)

	return user, err
}

func FindUserByConfirmationTokenHash(ctx context.Context, tokenHash string) (*User, error) {
	user := new(User)

	err := db.Bun.
		NewSelect().
		Model(user).
		Where("confirmation_token_hash = ?", tokenHash).
		Scan(ctx)

	return user, err
}

func UpdateUser(ctx context.Context, user *User) (*User, error) {
	_, err := db.Bun.
		NewUpdate().
		Model(user).
		WherePK().
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func DeleteExpiredUsers(ctx context.Context) error {
	_, err := db.Bun.
		NewDelete().
		Model((*User)(nil)).
		Where("confirmed = ?", false).
		Where("confirmation_expires_at < ?", time.Now()).
		Exec(ctx)

	return err
}
