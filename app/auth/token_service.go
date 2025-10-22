package auth

type TokenPayload struct {
	UserID string `json:"userID"`
}

type TokenService interface {
	GenerateAuthAccessToken(userID string) (string, error)
	ValidateAuthAccessToken(token string) (*TokenPayload, error)
	GenerateAuthRefreshToken(userID string) (string, error)
	ValidateAuthRefreshToken(token string) (*TokenPayload, error)
	GenerateSecureEmailToken(email string) (string, error)
}
