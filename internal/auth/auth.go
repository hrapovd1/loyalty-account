package auth

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
)

const (
	hashSalt       = "sd!oJFDw4-3409sdf."
	expireDuration = 30 * time.Minute
	signingKey     = "r'tyJFSdf384SLD.jsdf"
)

func CreateUser(ctx context.Context, storage *dbstorage.DBStorage, user models.User) error {
	pwdHash := sha1.New()
	pwdHash.Write([]byte(user.Password))
	pwdHash.Write([]byte(hashSalt))
	user.Password = fmt.Sprintf("%x", pwdHash.Sum(nil))

	return storage.CreateUser(ctx, user)
}

func GetToken(ctx context.Context, storage *dbstorage.DBStorage, user models.User) (string, error) {
	pwdHash := sha1.New()
	pwdHash.Write([]byte(user.Password))
	pwdHash.Write([]byte(hashSalt))
	user.Password = fmt.Sprintf("%x", pwdHash.Sum(nil))
	dbUser, err := storage.GetUser(ctx, user.Login)
	if err != nil {
		return "", err
	}
	if dbUser.Password != user.Password {
		return "", dbstorage.ErrInvalidLoginPassword
	}
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		&types.Claims{
			Login: user.Login,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		})

	return token.SignedString([]byte(signingKey))
}

func CheckToken(accessToken string) (string, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&types.Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(signingKey), nil
		},
	)
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*types.Claims); ok && token.Valid {
		return claims.Login, nil
	}

	return "", ErrTokenWrong
}
