package auth

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/hrapovd1/loyalty-account/internal/dbstorage"
	"github.com/hrapovd1/loyalty-account/internal/models"
	"github.com/hrapovd1/loyalty-account/internal/types"
)

const (
	hashSalt       = "sd!oJFDw4-3409sdf."
	expireDuration = 30 // minutes
	signingKey     = "r'tyJFSdf384SLD.jsdf"
)

func CreateUser(ctx context.Context, db *sql.DB, user models.User) error {
	pwdHash := sha1.New()
	pwdHash.Write([]byte(user.Password))
	pwdHash.Write([]byte(hashSalt))
	user.Password = fmt.Sprintf("%x", pwdHash.Sum(nil))

	return dbstorage.Save(ctx, db, user)
}

func GetToken(ctx context.Context, db *sql.DB, user models.User) (string, error) {
	pwdHash := sha1.New()
	pwdHash.Write([]byte(user.Password))
	pwdHash.Write([]byte(hashSalt))
	user.Password = fmt.Sprintf("%x", pwdHash.Sum(nil))
	dbUser, err := dbstorage.GetUser(ctx, db, user.Login)
	if err != nil {
		return "", err
	}
	if dbUser.Password != user.Password {
		return "", fmt.Errorf("Wrong password")
	}
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		&types.Claims{
			Login: user.Login,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireDuration * time.Minute)),
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
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
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

	return "", fmt.Errorf("token wrong")
}
