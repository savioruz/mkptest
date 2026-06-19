package dto

import (
	"oil/infras/jwt"
	userModel "oil/internal/domains/user/model"
	"oil/shared/constant"
	gModel "oil/shared/model"
	"oil/shared/timezone"
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email    string  `json:"email"               validate:"required,email"`
	Password string  `json:"password"            validate:"required,min=8"`
	FullName *string `json:"full_name,omitempty"`
}

func (r *RegisterRequest) ToUserModel(username string, hashedPassword string) userModel.User {
	return userModel.User{
		ID:         uuid.NewString(),
		Email:      r.Email,
		Password:   hashedPassword,
		Level:      constant.RoleUser,
		FullName:   r.FullName,
		IsVerified: false,
		Active:     true,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  username,
			ModifiedBy: username,
		},
	}
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateLastLoginRequest struct {
	LastLogin time.Time `db:"last_login" json:"last_login" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (l *LoginResponse) FromTokenPair(tokenPair *jwt.TokenPair) {
	l.AccessToken = tokenPair.AccessToken
	l.RefreshToken = tokenPair.RefreshToken
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r *RefreshTokenResponse) FromTokenPair(tokenPair *jwt.TokenPair) {
	r.AccessToken = tokenPair.AccessToken
	r.RefreshToken = tokenPair.RefreshToken
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=8"`
}

type UpdatePasswordRequest struct {
	Password string `db:"password" json:"password" validate:"required,min=8"`
}
