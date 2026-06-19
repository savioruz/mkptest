package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default cost for bcrypt hashing
	DefaultCost = bcrypt.DefaultCost
)

var (
	ErrInvalidPassword   = errors.New("invalid password")
	ErrEmptyPassword     = errors.New("password cannot be empty")
	ErrHashingPassword   = errors.New("error hashing password")
	ErrVerifyingPassword = errors.New("error verifying password")
)

// Hash generates a bcrypt hash of the password
func Hash(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrHashingPassword, err)
	}

	return string(bytes), nil
}

// Verify checks if the provided password matches the hash
func Verify(password, hash string) error {
	if password == "" || hash == "" {
		return ErrInvalidPassword
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidPassword
		}

		return fmt.Errorf("%w: %w", ErrVerifyingPassword, err)
	}

	return nil
}
