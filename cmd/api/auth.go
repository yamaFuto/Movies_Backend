package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Auth struct {
	//tokenの発行者
	Issuer        string
	//tokenの対象者
	Audience      string
	Secret        string
	TokenExpiry   time.Duration
	RefreshExpiry time.Duration
	CookieDomain  string
	CookiePath    string
	CookieName    string
}

//tokenを作るための必要最低限の情報を集積
type jwtUser struct {
	ID        int    `json:"time"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type TokenPairs struct {
	Token        string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

//claims
type Claims struct {
	jwt.RegisteredClaims
}

func (j *Auth) GenerateTokenPair(user *jwtUser) (TokenPairs, error) {
	//Create a token
	token := jwt.New(jwt.SigningMethodHS256)

	//Set the claims(tokenに保存される情報)
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = fmt.Sprintf("%s %s", user.FirstName, user.LastName)
	//databaseからuserを取り出す手段となる
	claims["sub"] =fmt.Sprint(user.ID)
	claims["aud"] = j.Audience
	claims["iss"] = j.Issuer
	claims["iat"] = time.Now().UTC().Unix()
	claims["typ"] = "JWT"

	//Set the expiry for JWT
	claims["exp"] = time.Now().UTC().Add(j.TokenExpiry).Unix()

	//Create a signed token
	//sliceByteが第一引数でなくてはいけない
	signedAccessToken, err := token.SignedString([]byte(j.Secret))
	if err != nil {
		return TokenPairs{}, err
	}

	//Create a refresh token and set claims
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshTokenClaims := refreshToken.Claims.(jwt.MapClaims)
	refreshTokenClaims["sub"] = fmt.Sprint(user.ID)
	refreshTokenClaims["iat"] = time.Now().UTC().Unix()

	//Set the expiry for the refresh token
	refreshTokenClaims["exp"] = time.Now().UTC().Add(j.RefreshExpiry).Unix()

	//Create signed refresh token
	signedRefreshToken, err := refreshToken.SignedString([]byte(j.Secret))
	if err != nil {
		return TokenPairs{}, err
	}

	//Create TokenPairs and populate with signed tokens
	var tokenPairs = TokenPairs {
		Token: signedAccessToken,
		RefreshToken: signedRefreshToken,
	}

	//Return tokenPairs
	return tokenPairs, nil
}

func (j *Auth) GetRefreshCookie(refreshToken string) *http.Cookie {
	return &http.Cookie{
		Name: j.CookieName,
		Path: j.CookiePath,
		Value: refreshToken,
		Expires: time.Now().Add(j.RefreshExpiry),
		MaxAge: int(j.RefreshExpiry.Seconds()),
		SameSite: http.SameSiteStrictMode,
		Domain: j.CookieDomain,
		HttpOnly: true,
		Secure: true,
	}
}

func (j *Auth) GetExpiredRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name: j.CookieName,
		Path: j.CookiePath,
		Value: "",
		Expires: time.Unix(0,0),
		MaxAge: -1,
		SameSite: http.SameSiteStrictMode,
		Domain: j.CookieDomain,
		HttpOnly: true,
		Secure: true,
	}
}

func (j *Auth) GetTokenFromHeaderAndVerify(w http.ResponseWriter, r *http.Request) (string, *Claims, error) {
	w.Header().Add("Vary", "Authorization")

	//get auth header
	authHeader := r.Header.Get("Authorization")

	//sanity check
	if authHeader == "" {
		return "", nil, errors.New("no auth header")
	}

	//split the header on space
	headerParts := strings.Split(authHeader, " ")
	if len(headerParts) != 2 {
		return "", nil, errors.New("invalid auth header")
	}

	//check to see if we have the word Bearer
	if headerParts[0] != "Bearer" {
		return "", nil, errors.New("invalid auth header")
	}


	token := headerParts[1]

	//declare an empty claims
	claims := &Claims{}

	//parse the token
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.Secret), nil
	})

	if err != nil {
		if strings.HasPrefix(err.Error(), "token is expired by") {
			return "", nil, errors.New("expired token")
		}
		return "", nil, err
	}

	if claims.Issuer != j.Issuer {
		return "", nil, errors.New("invalid issuer")
	}

	return token, claims, nil
}