package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"oil/config"
	"oil/infras/jwt"
	jwtMocks "oil/infras/jwt/mocks"
	"oil/infras/otel/mocks"
	"oil/internal/domains/auth/model/dto"
	"oil/internal/domains/auth/service"
	userMocks "oil/internal/domains/user/mocks"
	userModel "oil/internal/domains/user/model"
	"oil/shared/constant"
	gModel "oil/shared/model"
	"oil/shared/timezone"
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := userMocks.NewMockUser(ctrl)
	mockJWT := jwtMocks.NewMockJWT(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}

	svc := service.New(mockUserRepo, cfg, mockOtel, mockJWT)

	tests := []struct {
		name      string
		req       dto.RegisterRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful registration",
			req: dto.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				FullName: stringPtr("Test User"),
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, nil)

				mockUserRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "email already exists",
			req: dto.RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(true, nil)
			},
			wantErr: true,
		},
		{
			name: "user exist check error",
			req: dto.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "user insert error",
			req: dto.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Exist(gomock.Any(), gomock.Any()).
					Return(false, nil)

				mockUserRepo.EXPECT().
					Insert(gomock.Any(), gomock.Any()).
					Return(errors.New("insert error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			err := svc.Register(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := userMocks.NewMockUser(ctrl)
	mockJWT := jwtMocks.NewMockJWT(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}

	svc := service.New(mockUserRepo, cfg, mockOtel, mockJWT)

	// Valid user for successful login
	validUser := userModel.User{
		ID:         "user-id-123",
		Email:      "test@example.com",
		Password:   "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // "password" hashed
		Level:      constant.RoleUser,
		FullName:   stringPtr("Test User"),
		IsVerified: true,
		Active:     true,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  "system",
			ModifiedBy: "system",
		},
	}

	tests := []struct {
		name      string
		req       dto.LoginRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful login",
			req: dto.LoginRequest{
				Email:    "test@example.com",
				Password: "password",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(validUser, nil)

				mockJWT.EXPECT().
					GenerateTokenPair(gomock.Any(), validUser.ID, validUser.Email, validUser.Level).
					Return(&jwt.TokenPair{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil)

				mockUserRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: dto.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(userModel.User{}, errors.New("user not found"))
			},
			wantErr: true,
		},
		{
			name: "wrong password",
			req: dto.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(validUser, nil)
			},
			wantErr: true,
		},
		{
			name: "inactive user",
			req: dto.LoginRequest{
				Email:    "test@example.com",
				Password: "password",
			},
			setupMock: func() {
				inactiveUser := validUser
				inactiveUser.Active = false

				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(inactiveUser, nil)
			},
			wantErr: true,
		},
		{
			name: "token generation error",
			req: dto.LoginRequest{
				Email:    "test@example.com",
				Password: "password",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(validUser, nil)

				mockJWT.EXPECT().
					GenerateTokenPair(gomock.Any(), validUser.ID, validUser.Email, validUser.Level).
					Return(nil, errors.New("token generation failed"))
			},
			wantErr: true,
		},
		{
			name: "update last login error",
			req: dto.LoginRequest{
				Email:    "test@example.com",
				Password: "password",
			},
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(validUser, nil)

				mockJWT.EXPECT().
					GenerateTokenPair(gomock.Any(), validUser.ID, validUser.Email, validUser.Level).
					Return(&jwt.TokenPair{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil)

				mockUserRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("update error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.Login(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
			}
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := userMocks.NewMockUser(ctrl)
	mockJWT := jwtMocks.NewMockJWT(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}

	svc := service.New(mockUserRepo, cfg, mockOtel, mockJWT)

	tests := []struct {
		name      string
		req       dto.RefreshTokenRequest
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful token refresh",
			req: dto.RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			setupMock: func() {
				mockJWT.EXPECT().
					RefreshTokens(gomock.Any(), "valid-refresh-token").
					Return(&jwt.TokenPair{
						AccessToken:  "new-access-token",
						RefreshToken: "new-refresh-token",
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "invalid refresh token",
			req: dto.RefreshTokenRequest{
				RefreshToken: "invalid-refresh-token",
			},
			setupMock: func() {
				mockJWT.EXPECT().
					RefreshTokens(gomock.Any(), "invalid-refresh-token").
					Return(nil, errors.New("invalid token"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.Background()
			result, err := svc.RefreshToken(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
			}
		})
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := userMocks.NewMockUser(ctrl)
	mockJWT := jwtMocks.NewMockJWT(ctrl)
	mockOtel := mocks.NewOtel()

	cfg := &config.Config{}

	svc := service.New(mockUserRepo, cfg, mockOtel, mockJWT)

	// Valid user for password change
	validUser := userModel.User{
		ID:         "user-id-123",
		Email:      "test@example.com",
		Password:   "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // "password" hashed
		Level:      constant.RoleUser,
		FullName:   stringPtr("Test User"),
		IsVerified: true,
		Active:     true,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  "system",
			ModifiedBy: "system",
		},
	}

	tests := []struct {
		name      string
		req       dto.ChangePasswordRequest
		userID    string
		setupMock func()
		wantErr   bool
	}{
		{
			name: "successful password change",
			req: dto.ChangePasswordRequest{
				CurrentPassword: "password",
				NewPassword:     "newpassword123",
			},
			userID: "user-id-123",
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(validUser, nil)

				mockUserRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: dto.ChangePasswordRequest{
				CurrentPassword: "password",
				NewPassword:     "newpassword123",
			},
			userID: "nonexistent-id",
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(userModel.User{}, errors.New("user not found"))
			},
			wantErr: true,
		},
		{
			name: "user exists but empty ID (not found case)",
			req: dto.ChangePasswordRequest{
				CurrentPassword: "password",
				NewPassword:     "newpassword123",
			},
			userID: "user-id-123",
			setupMock: func() {
				emptyUser := userModel.User{} // Empty ID indicates not found

				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(emptyUser, nil)
			},
			wantErr: true,
		},
		{
			name: "wrong current password",
			req: dto.ChangePasswordRequest{
				CurrentPassword: "wrongpassword",
				NewPassword:     "newpassword123",
			},
			userID: "user-id-123",
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(validUser, nil)
			},
			wantErr: true,
		},
		{
			name: "update password error",
			req: dto.ChangePasswordRequest{
				CurrentPassword: "password",
				NewPassword:     "newpassword123",
			},
			userID: "user-id-123",
			setupMock: func() {
				mockUserRepo.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(validUser, nil)

				mockUserRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("update error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			ctx := context.WithValue(context.Background(), constant.ContextGuest, "test-user")
			err := svc.ChangePassword(ctx, tt.req, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
