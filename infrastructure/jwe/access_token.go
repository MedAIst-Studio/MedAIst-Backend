package jwe

import (
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

func (s *Service) GenerateAuthAccessToken(userID string) (string, error) {
	claims := AuthClaims{
		UserID: userID,
		Iat:    time.Now().Unix(),
		Exp:    time.Now().Add(s.authAccessTokenDuration).Unix(),
	}
	return encryptClaims(s.accessTokenEncrypter, claims)
}

func (s *Service) ValidateAuthAccessToken(tokenString string) (*domain.TokenPayload, error) {
	var claims AuthClaims
	if err := decryptClaims(tokenString, s.accessTokenEncryptionKey, &claims); err != nil {
		return nil, err
	}
	if isExpired(claims.Exp) {
		return nil, domain.ErrExpiredToken
	}
	return &domain.TokenPayload{
		UserID: claims.UserID,
	}, nil
}
