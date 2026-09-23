package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testPasswordHash = "$2a$10$MqkF//ZWuwkD2z1gJdPYZuURfCrSs4VLa7.dNX1VnBTYg28.lrCka"
	testSecret       = "test-secret-for-unit-test"
)

func TestVerifyPassword(t *testing.T) {
	user := NewUser("zhangsan", testPasswordHash)

	tests := []struct {
		name     string
		userName string
		password string
		want     bool
	}{
		{"账号与口令均正确", "zhangsan", "123456", true},
		{"口令错误", "zhangsan", "654321", false},
		{"账号错误", "lisi", "123456", false},
		{"账号为空", "", "123456", false},
		{"口令为空", "zhangsan", "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := user.VerifyPassword(test.userName, test.password)

			if got != test.want {
				t.Errorf("VerifyPassword(%q, %q) 期望 %v, 实际得到 %v", test.userName, test.password, test.want, got)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	a := NewAuthenticator(testSecret, time.Hour)

	token, err := a.GenerateToken("zhangsan")

	if err != nil {
		t.Fatalf("GenerateToken() 期望 nil, 得到 %v", err)
	}

	if token == "" {
		t.Fatalf("GenerateToken() 返回了空字符串")
	}

	// JWT 由 header.payload.signature 三段组成
	if parts := strings.Split(token, "."); len(parts) != 3 {
		t.Errorf("令牌应由三段组成, 实际得到 %d 段", len(parts))
	}
}

func TestParseToken(t *testing.T) {
	a := NewAuthenticator(testSecret, time.Hour)

	token, err := a.GenerateToken("zhangsan")

	if err != nil {
		t.Fatalf("生成测试令牌失败: %v", err)
	}

	claims, err := a.ParseToken(token)

	if err != nil {
		t.Fatalf("ParseToken() 期望 nil, 得到 %v", err)
	}

	if claims.Subject != "zhangsan" {
		t.Errorf("Subject 期望 %q, 实际得到 %q", "zhangsan", claims.Subject)
	}

	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatalf("签发时间或过期时间缺失")
	}

	// 有效期为 1 小时: 允许 1 分钟的误差
	validFor := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)

	if validFor < 59*time.Minute || validFor > 61*time.Minute {
		t.Errorf("有效期期望约 1 小时, 实际得到 %v", validFor)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	a := NewAuthenticator(testSecret, time.Hour)

	validToken, err := a.GenerateToken("zhangsan")

	if err != nil {
		t.Fatalf("生成测试令牌失败: %v", err)
	}

	parts := strings.Split(validToken, ".")

	if len(parts) != 3 {
		t.Fatalf("测试令牌格式异常: %q", validToken)
	}

	// 篡改载荷(第二段): 内容与签名不再匹配
	payload := []byte(parts[1])

	if payload[0] == 'e' {
		payload[0] = 'f'
	} else {
		payload[0] = 'e'
	}

	tamperedPayload := parts[0] + "." + string(payload) + "." + parts[2]

	// 用其他密钥签发的令牌
	otherAuthenticator := NewAuthenticator("another-secret-for-unit-test", time.Hour)

	otherToken, err := otherAuthenticator.GenerateToken("zhangsan")

	if err != nil {
		t.Fatalf("生成其他密钥的令牌失败: %v", err)
	}

	otherParts := strings.Split(otherToken, ".")

	if len(otherParts) != 3 {
		t.Fatalf("其他令牌格式异常: %q", otherToken)
	}

	// 替换签名段(第三段): 签名必然校验失败
	forgedSignature := parts[0] + "." + parts[1] + "." + otherParts[2]

	// 已过期的令牌: 有效期设为负数, 签发出来即为过期状态
	expiredAuthenticator := NewAuthenticator(testSecret, -time.Minute)

	expiredToken, err := expiredAuthenticator.GenerateToken("zhangsan")

	if err != nil {
		t.Fatalf("生成过期令牌失败: %v", err)
	}

	// 缺少 exp 的令牌: 用 MapClaims 签发, 只带 sub 不带过期时间
	noExpToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "zhangsan"}).
		SignedString([]byte(testSecret))

	if err != nil {
		t.Fatalf("生成缺少过期时间的令牌失败: %v", err)
	}

	// 使用 none 算法的令牌: 手工拼接三段, 第三段(签名)为空, 模拟算法混淆攻击
	encode := base64.RawURLEncoding.EncodeToString

	noneToken := encode([]byte(`{"alg":"none","typ":"JWT"}`)) + "." +
		encode([]byte(`{"sub":"zhangsan","exp":4102444800}`)) + "."

	tests := []struct {
		name string
		raw  string
	}{
		{"空字符串", ""},
		{"非 JWT 格式", "not-a-jwt"},
		{"缺少段落的令牌", parts[0] + "." + parts[1]},
		{"载荷被篡改的令牌", tamperedPayload},
		{"签名被替换的令牌", forgedSignature},
		{"其他密钥签发的令牌", otherToken},
		{"已过期的令牌", expiredToken},
		{"缺少过期时间的令牌", noExpToken},
		{"使用 none 算法签名的令牌", noneToken},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := a.ParseToken(test.raw); err == nil {
				t.Errorf("ParseToken(%q) 期望返回错误, 实际得到 nil", test.raw)
			}
		})
	}
}
