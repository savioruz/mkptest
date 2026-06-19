package jwt

//go:generate go run go.uber.org/mock/mockgen -source=./jwt.go -destination=./mocks/jwt_mock.go -package=mocks

import (
	"context"
	"errors"
	"fmt"
	"oil/config"
	"oil/shared"
	"oil/shared/cache"
	"oil/shared/constant"
	"oil/shared/timezone"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrExpiredToken            = errors.New("token has expired")
	ErrInvalidClaim            = errors.New("invalid token claim")
	ErrUnknownTokenType        = errors.New("unknown token type")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrAuthHeaderMissing       = errors.New("authorization header is missing")
	ErrInvalidAuthHeaderFormat = errors.New("invalid authorization header format")
	ErrTokenParsingFailed      = errors.New("failed to parse token")
	ErrTokenSigningFailed      = errors.New("failed to sign token")
	ErrTokenGenerationFailed   = errors.New("failed to generate token")
	ErrCacheOperationFailed    = errors.New("cache operation failed")
)

type TokenType string

const (
	AccessToken             TokenType = "access"
	RefreshToken            TokenType = "refresh"
	cacheJwtUserPrefix      string    = "jwt:user"
	cacheJwtBlacklistPrefix string    = "jwt:blacklist"
	cacheJwtRevokedValue    string    = "revoked"
)

type Claims struct {
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role,omitempty"`
	TokenID  string    `json:"token_id"`
	Type     TokenType `json:"type"`
	IssuedAt time.Time `json:"iat"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JWT interface {
	GenerateTokenPair(ctx context.Context, userID, email, role string) (*TokenPair, error)
	ValidateToken(ctx context.Context, tokenString string, tokenType TokenType) (*Claims, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error)
	RevokeToken(ctx context.Context, tokenString string, tokenType TokenType) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
}

type Service struct {
	config *config.Config
	cache  cache.RedisCache
}

// New creates a new JWT service with Redis integration
func New(cfg *config.Config, redisCache cache.RedisCache) JWT {
	return &Service{
		config: cfg,
		cache:  redisCache,
	}
}

// GenerateTokenPair generates both access and refresh tokens with Redis tracking
func (s *Service) GenerateTokenPair(ctx context.Context, userID, email, role string) (*TokenPair, error) {
	now := timezone.Now()

	// Generate access token
	accessToken, accessTokenID, err := s.generateToken(userID, email, role, AccessToken, now, s.config.JWT.AccessExpireMin)
	if err != nil {
		return nil, ErrTokenGenerationFailed
	}

	// Generate refresh token
	refreshToken, refreshTokenID, err := s.generateToken(userID, email, role, RefreshToken, now, s.config.JWT.RefreshExpireMin)
	if err != nil {
		return nil, ErrTokenGenerationFailed
	}

	// Store token metadata in Redis
	if err := s.storeTokenMetadata(ctx, userID, accessTokenID, AccessToken, s.config.JWT.AccessExpireMin*constant.MinutesToSeconds); err != nil {
		return nil, ErrCacheOperationFailed
	}

	if err := s.storeTokenMetadata(ctx, userID, refreshTokenID, RefreshToken, s.config.JWT.RefreshExpireMin*constant.MinutesToSeconds); err != nil {
		return nil, ErrCacheOperationFailed
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// generateToken creates a JWT token with the specified parameters
func (s *Service) generateToken(userID, email, role string, tokenType TokenType, issuedAt time.Time, expireMin int) (string, string, error) {
	expiresAt := issuedAt.Add(time.Duration(expireMin) * time.Minute)
	tokenID := uuid.New().String()

	claims := Claims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		TokenID:  tokenID,
		Type:     tokenType,
		IssuedAt: issuedAt,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
			Issuer:    s.config.App.Name,
			Subject:   userID,
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	var secret string

	switch tokenType {
	case AccessToken:
		secret = s.config.JWT.AccessSecret
	case RefreshToken:
		secret = s.config.JWT.RefreshSecret
	default:
		return "", "", ErrUnknownTokenType
	}

	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", ErrTokenSigningFailed
	}

	return signedToken, tokenID, nil
}

// ValidateToken validates and parses a JWT token with Redis blacklist check
func (s *Service) ValidateToken(ctx context.Context, tokenString string, tokenType TokenType) (*Claims, error) {
	var secret string

	switch tokenType {
	case AccessToken:
		secret = s.config.JWT.AccessSecret
	case RefreshToken:
		secret = s.config.JWT.RefreshSecret
	default:
		return nil, ErrUnknownTokenType
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnexpectedSigningMethod
		}

		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}

		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Verify token type
	if claims.Type != tokenType {
		return nil, ErrInvalidClaim
	}

	// Check if token is revoked in Redis
	revoked, err := s.IsTokenRevoked(ctx, claims.TokenID)
	if err != nil {
		return nil, ErrCacheOperationFailed
	}

	if revoked {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshTokens generates new token pair using refresh token
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateToken(ctx, refreshToken, RefreshToken)
	if err != nil {
		return nil, err
	}

	// Revoke the old refresh token
	if err := s.RevokeToken(ctx, refreshToken, RefreshToken); err != nil {
		return nil, ErrCacheOperationFailed
	}

	// Generate new token pair
	return s.GenerateTokenPair(ctx, claims.UserID, claims.Email, claims.Role)
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrAuthHeaderMissing
	}

	const prefix = "Bearer "
	if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", ErrInvalidAuthHeaderFormat
	}

	return authHeader[len(prefix):], nil
}

// storeTokenMetadata stores token metadata in Redis
func (s *Service) storeTokenMetadata(ctx context.Context, userID, tokenID string, tokenType TokenType, expireSeconds int) error {
	// Store in user's token list using BuildCacheKey
	cacheKey := shared.BuildCacheKey(cacheJwtUserPrefix, userID, tokenID)
	tokenData := fmt.Sprintf("%s:%s", tokenID, tokenType)

	if err := s.cache.Save(ctx, cacheKey, tokenData, expireSeconds); err != nil {
		return ErrCacheOperationFailed
	}

	return nil
}

// RevokeToken revokes a specific token by adding it to blacklist
func (s *Service) RevokeToken(ctx context.Context, tokenString string, _ TokenType) error {
	// Parse token to get claims without validation
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return ErrTokenParsingFailed
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return ErrInvalidToken
	}

	// Add to blacklist with TTL equal to remaining token lifetime using BuildCacheKey
	blacklistKey := shared.BuildCacheKey(cacheJwtBlacklistPrefix, claims.TokenID)
	remaining := time.Until(claims.ExpiresAt.Time)

	if remaining > 0 {
		if err := s.cache.Save(ctx, blacklistKey, cacheJwtRevokedValue, int(remaining.Seconds())); err != nil {
			return ErrCacheOperationFailed
		}
	}

	// Remove from user's active tokens using BuildCacheKey
	userTokenKey := shared.BuildCacheKey(cacheJwtUserPrefix, claims.UserID, claims.TokenID)
	if err := s.cache.Delete(ctx, userTokenKey); err != nil {
		return ErrCacheOperationFailed
	}

	return nil
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID string) error {
	// Clear all user tokens using BuildCacheKey pattern
	userTokensPattern := shared.BuildCacheKey(cacheJwtUserPrefix, userID, "*")
	if err := s.cache.Clear(ctx, userTokensPattern); err != nil {
		return ErrCacheOperationFailed
	}

	return nil
}

// IsTokenRevoked checks if a token is in the blacklist
func (s *Service) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	blacklistKey := shared.BuildCacheKey(cacheJwtBlacklistPrefix, tokenID)

	var result string

	err := s.cache.Get(ctx, blacklistKey, &result)
	if err != nil {
		if errors.Is(err, cache.Nil) {
			return false, nil
		}

		return false, ErrCacheOperationFailed
	}

	return true, nil
}
