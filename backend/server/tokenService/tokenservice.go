package tokenService

// code taken from tutorial here: https://pascalallen.medium.com/jwt-authentication-with-go-242215a9b4f8
import (
	"os"

	"github.com/golang-jwt/jwt"
)

type UserClaims struct {
	UserID   int32  `json:"userid"`
	Username string `json:"username"`
	PartyId  int32  `json:"partyid"`
	jwt.StandardClaims
}

func NewAccessToken(claims UserClaims) (string, error) {
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return accessToken.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
}

func NewRefreshToken(claims jwt.StandardClaims) (string, error) {
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return refreshToken.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
}

func ParseAccessToken(accessToken string) *UserClaims {
	parsedAccessToken, err := jwt.ParseWithClaims(accessToken, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("TOKEN_SECRET")), nil
	})
	if err != nil {
		return nil
	}
	return parsedAccessToken.Claims.(*UserClaims)
}

func ParseRefreshToken(refreshToken string) *jwt.StandardClaims {
	parsedRefreshToken, err := jwt.ParseWithClaims(refreshToken, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("TOKEN_SECRET")), nil
	})
	if err != nil {
		return nil
	}
	return parsedRefreshToken.Claims.(*jwt.StandardClaims)
}
