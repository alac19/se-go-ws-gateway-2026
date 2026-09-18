// Package auth 提供网关的鉴权能力: 账号口令校验与 JWT 的签发、校验。
// 本包不依赖 HTTP 框架, 只处理纯粹的鉴权逻辑, 供 handler 与 middleware 调用。
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims 是 token 的负载(声明)类型, 直接复用 jwt 库的标准声明集。
// 这里用 "=" 声明为别名, 因此 auth.Claims 与 jwt.RegisteredClaims 是同一个类型:
// 既满足库要求的 jwt.Claims 接口, 又能让调用方继续使用本包的命名。
// 当前使用的字段: Subject(账号名)、IssuedAt(签发时间)、ExpiresAt(过期时间)。
type Claims = jwt.RegisteredClaims

// ContextKeyClaims 是鉴权中间件写入 gin.Context 时使用的键名。
// 中间件通过 ctx.Set(auth.ContextKeyClaims, claims) 写入当前请求的身份,
// 后续 handler 通过 ctx.Get(auth.ContextKeyClaims) 取出并断言为 auth.Claims。
// 放在本包是为了让中间件与 handler 共用一个键名, 避免两处字符串写得不一致。
const ContextKeyClaims = "auth.claims"

// Authenticator 持有签发与校验 token 所需的配置。
// 由 main 在启动时构造一次并被所有请求共享, 因此运行期只读:
// 各方法只能读取接收者字段, 不得修改, 否则会与其他请求产生数据竞争。
type Authenticator struct {
	secret string        // 签名密钥, 用于 HMAC-SHA256 签名与验签
	ttl    time.Duration // token 有效期, 签发时用于计算 exp
}

// User 表示预置的管理员账号。
// 密码只保存 bcrypt 哈希, 不保存明文; 明文仅在登录请求中出现一次, 用完即弃。
type User struct {
	userName     string // 预置账号名
	passwordHash string // 预置密码的 bcrypt 哈希(60 字符, 以 $2a$/$2b$ 开头)
}

// NewAuthenticator 使用签名密钥与 token 有效期构造 Authenticator。
func NewAuthenticator(secret string, ttl time.Duration) *Authenticator {
	return &Authenticator{
		secret: secret,
		ttl:    ttl,
	}
}

// NewUser 使用账号名与 bcrypt 哈希构造预置账号。
func NewUser(userName, passwordHash string) *User {
	return &User{
		userName:     userName,
		passwordHash: passwordHash,
	}
}

// VerifyPassword 校验登录凭证: 先比对账号名, 再用 bcrypt 比对明文密码与预置哈希。
// 账号名不符或密码错误均返回 false, 不区分二者, 避免暴露账号是否存在。
// 参数 name 为客户端提交的账号名, password 为客户端提交的明文密码。
func (user *User) VerifyPassword(name, password string) bool {
	if name != user.userName {
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.passwordHash), []byte(password)); err != nil {
		return false
	}

	return true
}

// GenerateToken 为指定账号签发 token, 返回可直接交给客户端的字符串。
// 声明中携带 Subject(账号名)、IssuedAt(签发时间)、ExpiresAt(过期时间),
// 使用 HS256 算法与预置密钥签名; 签名失败时返回错误, 调用方应响应 500。
// 参数 name 为已通过校验的账号名。
func (a *Authenticator) GenerateToken(name string) (string, error) {
	// 组装标准声明: 过期时间 = 当前时刻 + 有效期
	claims := jwt.RegisteredClaims{
		Subject:   name,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.ttl)),
	}
	// NewWithClaims 负责生成 header 并关联声明;
	// SignedString 完成序列化、base64url 编码、签名与三段拼接, 返回最终 token。
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(a.secret))

	return signed, err
}

// ParseToken 校验客户端传来的 token, 成功时返回其中的声明。
// 校验由 jwt 库完成: 拆分三段、解码、按 HS256 校验签名、检查 exp 存在且未过期。
// 失败时返回零值与错误, 调用方(中间件)据此响应 401;
// 成功返回的 claims 可用于获取 Subject(账号名), 例如记录日志或注入 gin.Context。
func (a *Authenticator) ParseToken(raw string) (Claims, error) {
	var claims Claims

	// keyFunc 提供验签密钥; WithValidMethods 固定算法以防算法混淆攻击;
	// WithExpirationRequired 要求 token 必须携带 exp, 否则视为非法。
	if _, err := jwt.ParseWithClaims(raw, &claims,
		func(t *jwt.Token) (any, error) { return []byte(a.secret), nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	); err != nil {
		return Claims{}, err
	}

	return claims, nil
}
