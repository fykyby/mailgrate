package data

import (
	"app/db"
	"context"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID       int `bun:",pk,autoincrement"`
	Email    string
	Password string
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

func FindUserByEmail(ctx context.Context, email string) (*User, error) {
	user := new(User)

	err := db.Bun.
		NewSelect().
		Model(user).
		Where("email = ?", email).
		Scan(ctx)

	return user, err
}
