// 鉴权工具
package auth

import (
	"time"
)

// 定义 token 数据类型 Claims 结构体
type Claims struct {
	Subject   string    // 用户名
	IssuedAt  time.Time // 签发时间
	ExpiresAt time.Time // 过期时间
}

// 定义 token 验证器 Authenticator 结构体
type Authenticator struct {
	secret string        // 密钥
	ttl    time.Duration // 过期时间
}

// 定义账号凭证 User 结构体
type User struct {
	userName     string // 预置账号名
	passwordHash string // 预置密码
}

func NewAuthenticator(secret string, ttl time.Duration) *Authenticator {
	return &Authenticator{
		secret: secret,
		ttl:    ttl,
	}
}

func NewUser(userName, passwordHash string) *User {
	return &User{
		userName:     userName,
		passwordHash: passwordHash,
	}
}

// 验身份凭证
func (user *User) VerifyPassword(name, password string) bool {
	// 跟预置账号名/哈希码比较...

	return true
}

// 签发 token
func (a *Authenticator) GenerateToken(name string) (string, error) {
	// 生成并返回 token...

	return "token", nil
}

// 校验 token
func (a *Authenticator) ParseToken(token string) (Claims, error) {
	// 验签...

	return Claims{}, nil
}
