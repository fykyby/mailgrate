package errorsx

import (
	"database/sql"
	"errors"
	"strings"
)

func IsUniqueConstraintError(err error) bool {
	if err != nil && (strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "UNIQUE constraint")) {
		return true
	}

	return false
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found")
}
