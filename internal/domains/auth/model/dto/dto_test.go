package dto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"oil/infras/jwt"
	"oil/internal/domains/auth/model/dto"
	"oil/shared/constant"
	"oil/shared/timezone"
)

func TestRegisterRequest_ToUserModel(t *testing.T) {
	req := dto.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		FullName: stringPtr("Test User"),
	}

	username := "system"
	hashedPassword := "hashed-password"

	model := req.ToUserModel(username, hashedPassword)

	assert.NotEmpty(t, model.ID, "expected ID to be generated")
	assert.Equal(t, req.Email, model.Email)
	assert.Equal(t, hashedPassword, model.Password)
	assert.Equal(t, constant.RoleUser, model.Level)
	assert.Equal(t, req.FullName, model.FullName)
	assert.False(t, model.IsVerified, "expected IsVerified to be false by default")
	assert.True(t, model.Active, "expected Active to be true by default")
	assert.Equal(t, username, model.CreatedBy)
	assert.Equal(t, username, model.ModifiedBy)
	assert.False(t, model.CreatedAt.IsZero(), "expected CreatedAt to be set")
	assert.False(t, model.ModifiedAt.IsZero(), "expected ModifiedAt to be set")
}

func TestRegisterRequest_ToUserModel_WithoutFullName(t *testing.T) {
	req := dto.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		// FullName not provided
	}

	username := "system"
	hashedPassword := "hashed-password"

	model := req.ToUserModel(username, hashedPassword)

	assert.NotEmpty(t, model.ID)
	assert.Equal(t, req.Email, model.Email)
	assert.Equal(t, hashedPassword, model.Password)
	assert.Nil(t, model.FullName, "expected FullName to be nil when not provided")
}

func TestLoginResponse_FromTokenPair(t *testing.T) {
	tokenPair := &jwt.TokenPair{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	}

	var response dto.LoginResponse
	response.FromTokenPair(tokenPair)

	assert.Equal(t, tokenPair.AccessToken, response.AccessToken)
	assert.Equal(t, tokenPair.RefreshToken, response.RefreshToken)
}

func TestRefreshTokenResponse_FromTokenPair(t *testing.T) {
	tokenPair := &jwt.TokenPair{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}

	var response dto.RefreshTokenResponse
	response.FromTokenPair(tokenPair)

	assert.Equal(t, tokenPair.AccessToken, response.AccessToken)
	assert.Equal(t, tokenPair.RefreshToken, response.RefreshToken)
}

func TestUpdateLastLoginRequest(t *testing.T) {
	now := timezone.Now()

	req := dto.UpdateLastLoginRequest{
		LastLogin: now,
	}

	assert.Equal(t, now, req.LastLogin)
}

func TestUpdatePasswordRequest(t *testing.T) {
	hashedPassword := "hashed-new-password"

	req := dto.UpdatePasswordRequest{
		Password: hashedPassword,
	}

	assert.Equal(t, hashedPassword, req.Password)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
