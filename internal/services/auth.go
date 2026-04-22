package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	jwtauth "github.com/allposty/allposty-backend/internal/auth"
	"github.com/allposty/allposty-backend/internal/config"
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailTaken      = errors.New("email already registered")
	ErrInvalidPassword = errors.New("invalid email or password")
	ErrTokenExpired    = errors.New("refresh token expired or revoked")
)

type AuthService struct {
	users *repository.UserRepository
	cfg   *config.Config
}

func NewAuthService(users *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{users: users, cfg: cfg}
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

func (s *AuthService) Register(input RegisterInput) (*models.User, *TokenPair, error) {
	// Check duplicate email
	existing, err := s.users.FindByEmail(input.Email)
	if err == nil && existing != nil {
		return nil, nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	user := &models.User{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: string(hash),
		PlanTier:     "free",
	}
	if err := s.users.Create(user); err != nil {
		return nil, nil, err
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

type LoginInput struct {
	Email    string
	Password string
}

func (s *AuthService) Login(input LoginInput) (*models.User, *TokenPair, error) {
	user, err := s.users.FindByEmail(input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidPassword
		}
		return nil, nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, ErrInvalidPassword
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

func (s *AuthService) Refresh(refreshToken string) (*TokenPair, error) {
	stored, err := s.users.FindRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrTokenExpired
	}

	if time.Now().Unix() > stored.ExpiresAt {
		_ = s.users.RevokeRefreshToken(refreshToken)
		return nil, ErrTokenExpired
	}

	user, err := s.users.FindByID(stored.UserID)
	if err != nil {
		return nil, err
	}

	// Rotate: revoke old, issue new
	_ = s.users.RevokeRefreshToken(refreshToken)
	return s.issueTokens(user)
}

func (s *AuthService) Logout(refreshToken string) error {
	return s.users.RevokeRefreshToken(refreshToken)
}

func (s *AuthService) issueTokens(user *models.User) (*TokenPair, error) {
	accessToken, err := jwtauth.NewAccessToken(user.ID, user.Email, s.cfg.JWT.Secret, s.cfg.JWT.AccessTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	rt := &models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(s.cfg.JWT.RefreshTTL).Unix(),
	}
	if err := s.users.CreateRefreshToken(rt); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.cfg.JWT.AccessTTL.Seconds()),
	}, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
