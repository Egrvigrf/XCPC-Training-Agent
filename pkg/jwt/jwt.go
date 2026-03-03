package jwt

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	secret []byte
	expire time.Duration
}

func NewJWT(secret, expire string) *JWT {
	d, _ := time.ParseDuration(expire)
	return &JWT{
		secret: []byte(secret),
		expire: d,
	}
}

// GenerateToken 将用户信息打包成 Token
func (j *JWT) GenerateToken(claims map[string]interface{}) (string, error) { // 生成 Token
	mapClaims := jwt.MapClaims{
		"exp": time.Now().Add(j.expire).Unix(), // 设置过期时间
		"iat": time.Now().Unix(),               // 记录签发时间
	}

	// 将额外信息填充入 JWT 的信息 map 中
	for k, v := range claims {
		mapClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mapClaims) // 采用 HS256 加密之前生成的 map，也就是储存信息了
	return token.SignedString(j.secret)
}

// ParseToken 解析并且验证 Token，返回解析出来的数据
func (j *JWT) ParseToken(tokenStr string) (jwt.MapClaims, error) { // 解析 Token
	// 传入 Token 和匿名函数，匿名函数要求返回空接口以兼容所有类型的秘钥。
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})

	// 最后解析成功，需要用函数判断这个 Token 是否合法
	if err != nil || !token.Valid {
		return nil, err
	}

	// token.Claims 在库底层是一个通用接口类型，通过括号可以强行绑定这个是一个 Map，其实 jwt.MapClaims 是 map[string]interface{} 的别名
	return token.Claims.(jwt.MapClaims), nil
}

// GetUID 从 Gin 的 Context 中安全地取出存储的 UID
func (j *JWT) GetUID(ctx *gin.Context) (int64, error) {
	uidVal, ok := ctx.Get("uid")
	if !ok {
		return 0, errors.New("未登录")
	}

	uid, ok := uidVal.(int64)
	if !ok {
		return 0, errors.New("用户信息异常")
	}

	return uid, nil
}

// GetIsAdmin 从 Gin 的 Context 中提取用户身份
func (j *JWT) GetIsAdmin(ctx *gin.Context) bool {
	res, ok := ctx.Get("isAdmin")
	if !ok {
		return false
	}
	return res.(bool)
}
