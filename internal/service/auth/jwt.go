package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func NewJWT(secret, issuer string, ttl time.Duration) *JWTService {
	return &JWTService{secret: []byte(secret), issuer: issuer, ttl: ttl}
}

type Claims struct {
	TenantID int64  `json:"tenant_id"`
	UserID   int64  `json:"user_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (s *JWTService) Sign(tenantID, userID int64, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) Verify(tokenStr string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) { return s.secret, nil })
	if err != nil { return nil, err }
	if c, ok := tok.Claims.(*Claims); ok && tok.Valid { return c, nil }
	return nil, jwt.ErrTokenInvalidClaims
}
