package model

import "oil/shared/model"

const (
	TableName  = "users"
	EntityName = "user"

	FieldID           = "id"
	FieldEmail        = "email"
	FieldPassword     = "password"
	FieldLevel        = "level"
	FieldGoogleID     = "google_id"
	FieldFullName     = "full_name"
	FieldProfileImage = "profile_image"
	FieldIsVerified   = "is_verified"
	FieldLastLogin    = "last_login"
	FieldActive       = "active"
)

type User struct {
	ID           string  `db:"id"`
	Email        string  `db:"email"`
	Password     string  `db:"password"`
	Level        string  `db:"level"`
	GoogleID     *string `db:"google_id"`
	FullName     *string `db:"full_name"`
	ProfileImage *string `db:"profile_image"`
	IsVerified   bool    `db:"is_verified"`
	LastLogin    *string `db:"last_login"`
	Active       bool    `db:"active"`
	model.Metadata
}
