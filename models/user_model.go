package models

import (
	"app/db"
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID        int `bun:",pk,autoincrement"`
	Email     string
	Password  string
	CreatedAt time.Time `bun:",nullzero,default:current_timestamp"`
}

type PasswordReset struct {
	bun.BaseModel `bun:"table:password_resets"`

	ID        int `bun:",pk,autoincrement"`
	UserID    int
	Token     string
	ExpiresAt time.Time
}

func CreateUser(ctx context.Context, email string, password string) (*User, error) {
	user := &User{
		Email:    email,
		Password: password,
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

func CreatePasswordReset(ctx context.Context, userID int, token string, expiresAt time.Time) (*PasswordReset, error) {
	reset := &PasswordReset{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	_, err := db.Bun.
		NewInsert().
		Model(reset).
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return reset, nil
}

func FindPasswordResetByToken(ctx context.Context, token string) (*PasswordReset, error) {
	reset := new(PasswordReset)

	err := db.Bun.
		NewSelect().
		Model(reset).
		Where("token = ?", token).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	if reset.ExpiresAt.Before(time.Now()) {
		err := DeletePasswordReset(ctx, reset.ID)
		if err != nil {
			return nil, err
		}

		return nil, errors.New("password reset expired")
	}

	return reset, err
}

func FindPasswordResetByUserID(ctx context.Context, userID int) (*PasswordReset, error) {
	reset := new(PasswordReset)

	err := db.Bun.
		NewSelect().
		Model(reset).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	if reset.ExpiresAt.Before(time.Now()) {
		err := DeletePasswordReset(ctx, reset.ID)
		if err != nil {
			return nil, err
		}

		return nil, errors.New("password reset expired")
	}

	return reset, err
}

func DeletePasswordReset(ctx context.Context, id int) error {
	_, err := db.Bun.
		NewDelete().
		Model(new(PasswordReset)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

func DeletePasswordResetByUserID(ctx context.Context, userID int) error {
	_, err := db.Bun.
		NewDelete().
		Model(new(PasswordReset)).
		Where("user_id = ?", userID).
		Exec(ctx)

	return err
}
