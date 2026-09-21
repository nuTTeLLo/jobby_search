package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// PurposeAttachmentView marks the short-lived tokens minted for viewing one
// attachment in a browser tab. Browsers can't set an Authorization header on a
// plain navigation, so the token travels in the URL — hence the narrow scope
// (one attachment, minutes of life) rather than reusing the session token.
const PurposeAttachmentView = "attachment_view"

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	// Purpose and AttachmentID are set only on view tokens; a session token
	// leaves both empty.
	Purpose      string `json:"purpose,omitempty"`
	AttachmentID string `json:"attachment_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateViewToken mints a token good for exactly one attachment.
func GenerateViewToken(userID, attachmentID, secret string, expiration time.Duration) (string, error) {
	claims := Claims{
		UserID:       userID,
		Purpose:      PurposeAttachmentView,
		AttachmentID: attachmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateToken(userID, email, secret string, expiration time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
