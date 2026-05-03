package jwe

import (
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

func (s *Service) GenerateAuthRefreshToken(userID string) (string, error) {
	claims := AuthClaims{
		UserID: userID,
		Iat:    time.Now().Unix(),
		Exp:    time.Now().Add(s.authRefreshTokenDuration).Unix(),
	}
	return encryptClaims(s.refreshTokenEncrypter, claims)
}

func (s *Service) ValidateAuthRefreshToken(tokenString string) (*domain.TokenPayload, error) {
	var claims AuthClaims
	if err := decryptClaims(tokenString, s.refreshTokenEncryptionKey, &claims); err != nil {
		return nil, err
	}
	if isExpired(claims.Exp) {
		return nil, domain.ErrExpiredToken
	}

	return &domain.TokenPayload{
		UserID: claims.UserID,
	}, nil
}
