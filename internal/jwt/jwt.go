package jwt

import (
	"errors"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type JWT struct {
	secret         []byte
	accessTokenTTL time.Duration
}

func New(secret string) *JWT {
	return &JWT{
		secret:         []byte(secret),
		accessTokenTTL: 15 * time.Minute,
	}
}

func (j *JWT) GenerateAccessToken(userID, email string) (string, error) {
	tok, err := jwt.NewBuilder().
		Subject(userID).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(j.accessTokenTTL)).
		Claim("type", "access").
		Claim("email", email).
		Build()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256(), j.secret))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

func (j *JWT) ValidateAccessToken(tokenStr string) (string, error) {
	tok, err := jwt.Parse(
		[]byte(tokenStr),
		jwt.WithKey(jwa.HS256(), j.secret),
		jwt.WithValidate(true),
	)
	if err != nil {
		return "", err
	}

	subject, ok := tok.Subject()
	if !ok {
		return "", errors.New("token does not contain subject")
	}

	return subject, nil
}
