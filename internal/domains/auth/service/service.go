package service

import (
	"context"
	"fmt"
	"oil/config"
	"oil/infras/jwt"
	"oil/infras/otel"
	"oil/internal/domains/auth/model/dto"
	userModel "oil/internal/domains/user/model"
	userDto "oil/internal/domains/user/model/dto"
	userRepo "oil/internal/domains/user/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"
	"oil/shared/password"
	"oil/shared/timezone"

	"github.com/rs/zerolog/log"
)

type Auth interface {
	Register(ctx context.Context, req dto.RegisterRequest) error
	Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.RefreshTokenResponse, error)
	ChangePassword(ctx context.Context, req dto.ChangePasswordRequest, userID string) error
}

type serviceImpl struct {
	userRepo   userRepo.User
	cfg        *config.Config
	otel       otel.Otel
	jwtService jwt.JWT
}

func New(userRepo userRepo.User, cfg *config.Config, otel otel.Otel, jwt jwt.JWT) Auth {
	return &serviceImpl{
		userRepo:   userRepo,
		cfg:        cfg,
		otel:       otel,
		jwtService: jwt,
	}
}

func (s *serviceImpl) Register(ctx context.Context, req dto.RegisterRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Register")
	defer scope.End()
	defer scope.TraceIfError(err)

	emailFilter := gDto.FilterGroup{
		Filters: []any{
			gDto.Filter{
				Field:    userModel.FieldEmail,
				Operator: gDto.FilterOperatorEq,
				Value:    req.Email,
				Table:    userModel.TableName,
			},
		},
	}

	exists, err := s.userRepo.Exist(ctx, emailFilter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if user exists")

		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		return failure.BadRequestFromString("email already registered")
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash password")

		return fmt.Errorf("failed to hash password: %w", err)
	}

	username := constant.ContextGuest

	if err = s.userRepo.Insert(ctx, req.ToUserModel(username, hashedPassword)); err != nil {
		log.Error().Err(err).Msg("failed to create user")

		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (s *serviceImpl) Login(ctx context.Context, req dto.LoginRequest) (res dto.LoginResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Login")
	defer scope.End()
	defer scope.TraceIfError(err)

	emailFilter := gDto.FilterGroup{
		Filters: []any{
			gDto.Filter{
				Field:    userModel.FieldEmail,
				Operator: gDto.FilterOperatorEq,
				Value:    req.Email,
				Table:    userModel.TableName,
			},
		},
	}

	user, err := s.userRepo.Get(ctx, emailFilter)
	if err != nil {
		log.Warn().Str("email", req.Email).Msg("login attempt with non-existent email")

		return res, failure.BadRequestFromString("invalid email or password")
	}

	if err := password.Verify(req.Password, user.Password); err != nil {
		log.Warn().Str("email", req.Email).Msg("login attempt with wrong password")

		return res, failure.BadRequestFromString("invalid email or password")
	}

	if !user.Active {
		return res, failure.BadRequestFromString("user account is deactivated")
	}

	tokenPair, err := s.jwtService.GenerateTokenPair(ctx, user.ID, user.Email, user.Level)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate tokens")

		return res, fmt.Errorf("failed to generate tokens: %w", err)
	}

	lastLogin := dto.UpdateLastLoginRequest{LastLogin: timezone.Now()}
	updatedFields := shared.TransformFields(lastLogin, user.ID)

	if err := s.userRepo.Update(ctx, updatedFields, emailFilter); err != nil {
		log.Warn().Err(err).Str("user_id", user.ID).Msg("failed to update last login")

		return res, fmt.Errorf("failed to update last login: %w", err)
	}

	var userResponse userDto.UserResponse

	userResponse.FromModel(user)

	res.FromTokenPair(tokenPair)

	return res, nil
}

func (s *serviceImpl) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (res dto.RefreshTokenResponse, err error) {
	_, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".RefreshToken")
	defer scope.End()
	defer scope.TraceIfError(err)

	tokenPair, err := s.jwtService.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		log.Warn().Err(err).Msg("failed to refresh tokens")

		return res, failure.Unauthorized("invalid refresh token")
	}

	res.FromTokenPair(tokenPair)

	return res, nil
}

func (s *serviceImpl) ChangePassword(ctx context.Context, req dto.ChangePasswordRequest, userID string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".ChangePassword")
	defer scope.End()
	defer scope.TraceIfError(err)

	user, _ := ctx.Value(constant.ContextGuest).(string)
	filter := gDto.FilterGroup{
		Filters: []any{
			gDto.Filter{
				Field:    userModel.FieldID,
				Operator: gDto.FilterOperatorEq,
				Value:    userID,
				Table:    userModel.TableName,
			},
		},
	}

	model, err := s.userRepo.Get(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")

		return fmt.Errorf("failed to get user: %w", err)
	}

	if model.ID == "" {
		return failure.NotFound("user not found")
	}

	if err := password.Verify(req.CurrentPassword, model.Password); err != nil {
		return failure.BadRequestFromString("current password is incorrect")
	}

	hashedPassword, err := password.Hash(req.NewPassword)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash new password")

		return fmt.Errorf("failed to hash new password: %w", err)
	}

	updatePassword := dto.UpdatePasswordRequest{Password: hashedPassword}
	updatedFields := shared.TransformFields(updatePassword, user)

	if err = s.userRepo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update password")

		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
