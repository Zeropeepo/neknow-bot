package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Zeropeepo/neknow-bot/internal/auth/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/config"
)

type authService struct {
	repo domain.Repository
	cfg  *config.Config
}

func NewAuthService(repo domain.Repository, cfg *config.Config) domain.Service {
	return &authService{repo: repo, cfg: cfg}
}

// Helper functions

func (s *authService) generateToken(user *domain.User, expiry time.Duration, tokenType string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email": user.Email,
		"type": tokenType,
		"exp": time.Now().Add(expiry).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

func (s *authService) parseToken(ctx context.Context, tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	
	return claims, nil
}

func (s *authService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil{
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email: email,
		PasswordHash: string(hash),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}

	if user == nil {
		return "", "", errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash), 
		[]byte(password)); err != nil {
			return "", "", errors.New("invalid email or password")
		}

	accessToken, err := s.generateToken(user, s.cfg.JWT.AccessExpiry, "access")
	if err != nil {
		return "", "", err
	}

	refreshToken, err := s.generateToken(user, s.cfg.JWT.RefreshExpiry, "refresh")
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	claims, err := s.parseToken(ctx, refreshToken)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	if claims["type"] != "refresh" {
		return "", errors.New("invalid token type")
	}

	user := &domain.User{
		ID: claims["user_id"].(string),
		Email: claims["email"].(string),
	}
	
	return s.generateToken(user, s.cfg.JWT.AccessExpiry, "access")
}

func (s *authService) ValidateToken(ctx context.Context, token string) (*domain.TokenClaims, error) {
	claims, err := s.parseToken(ctx, token)
	if err != nil {
		return nil, errors.New("invalid token")
	}
	
	if claims["type"] != "access" {
		return nil, errors.New("invalid token type")
	}

	return &domain.TokenClaims{
		UserID: claims["user_id"].(string),
		Email: claims["email"].(string),
	}, nil
}
