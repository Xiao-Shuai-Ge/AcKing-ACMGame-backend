package jwtUtils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"tgwp/global"
	"time"
)

type TokenData struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
}

func GenToken(userID int64, username string, role int, exp time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(exp).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(global.Config.JWT.Secret))
	return tokenString, err
}

func IdentifyToken(tokenString string) (TokenData, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名方法: %v", token.Header["alg"])
		}
		return []byte(global.Config.JWT.Secret), nil
	})
	if err != nil {
		return TokenData{}, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		if time.Now().Unix() > int64(claims["exp"].(float64)) {
			return TokenData{}, fmt.Errorf("token已过期")
		}
	} else {
		return TokenData{}, fmt.Errorf("无效的token")
	}
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return TokenData{}, fmt.Errorf("token数据错误")
	}
	username, _ := claims["username"].(string)
	roleFloat, _ := claims["role"].(float64)
	return TokenData{
		UserID:   int64(userIDFloat),
		Username: username,
		Role:     int(roleFloat),
	}, nil
}

func GetUserId(c *gin.Context) int64 {
	if data, exists := c.Get(global.TOKEN_USER_ID); exists {
		userId, ok := data.(int64)
		if ok {
			return userId
		}
	}
	return 0
}

func GetRole(c *gin.Context) int {
	if data, exists := c.Get(global.TOKEN_ROLE); exists {
		role, ok := data.(int)
		if ok {
			return role
		}
	}
	return 0
}
