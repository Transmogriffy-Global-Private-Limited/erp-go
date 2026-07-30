package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

type Service struct {
	users map[string]User
}

func NewService() *Service {
	return &Service{users: make(map[string]User)}
}

func (s *Service) Signup(_ context.Context, req SignupRequest) (User, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		return User{}, errors.New("email and password are required")
	}

	superadminEmail := strings.TrimSpace(strings.ToLower(os.Getenv("SUPERADMIN_EMAIL")))
	superadminPassword := strings.TrimSpace(os.Getenv("SUPERADMIN_PASSWORD"))
	if superadminEmail != "" && superadminPassword != "" && req.Email == superadminEmail && req.Password == superadminPassword {
		req.Role = "superadmin"
	}
	if req.Role == "" {
		req.Role = "admin"
	}
	if _, exists := s.users[req.Email]; exists {
		return User{}, errors.New("user already exists")
	}

	user := User{
		ID:       fmt.Sprintf("user-%d", len(s.users)+1),
		Email:    req.Email,
		Password: hashPassword(req.Password),
		Role:     req.Role,
	}
	s.users[req.Email] = user
	return user, nil
}

func (s *Service) Login(_ context.Context, req LoginRequest) (string, error) {
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	superadminEmail := strings.TrimSpace(strings.ToLower(os.Getenv("SUPERADMIN_EMAIL")))
	superadminPassword := strings.TrimSpace(os.Getenv("SUPERADMIN_PASSWORD"))

	if req.Email == superadminEmail && req.Password == superadminPassword {
		user := User{
			ID:       "superadmin",
			Email:    req.Email,
			Password: hashPassword(req.Password),
			Role:     "superadmin",
		}
		s.users[req.Email] = user
		return s.issueToken(user)
	}

	user, ok := s.users[req.Email]
	if !ok || !checkPasswordHash(req.Password, user.Password) {
		return "", errors.New("invalid credentials")
	}

	return s.issueToken(user)
}

func (s *Service) issueToken(user User) (string, error) {

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET is not configured")
	}

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return encoded, nil
}

func (s *Service) Validate(tokenString string) (map[string]any, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET is not configured")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func checkPasswordHash(password, hash string) bool {
	return hashPassword(password) == hash
}
