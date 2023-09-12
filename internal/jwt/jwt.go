package jwt

import (
	"github.com/CvitoyBamp/gopher/internal/config"
	"github.com/go-chi/jwtauth/v5"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

var TokenAuth *jwtauth.JWTAuth

type Claims struct {
	jwt.RegisteredClaims
	Username string
	Password string
}

func init() {
	TokenAuth = jwtauth.New("HS256", []byte(config.Config.SecretToken), nil)
}

func CreateJWTToken(username, password string) (string, error) {

	claims := jwt.MapClaims{
		"username": username,
		"password": password,
	}

	jwtauth.SetExpiry(claims, time.Now().Add(time.Hour))

	_, tokenString, err := TokenAuth.Encode(claims)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}
