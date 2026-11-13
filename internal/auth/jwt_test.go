package auth

import (
	"strings"
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	token, err := Sign("test-secret", "alice", 2*time.Hour)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("token 应为三段式 JWT，实际为 %q", token)
	}

	claims, err := VerifyToken(token, "test-secret")
	if err != nil {
		t.Fatalf("VerifyToken 失败: %v", err)
	}
	if claims.Username != "alice" {
		t.Errorf("用户名应为 alice，实际为 %q", claims.Username)
	}
	if time.Until(claims.ExpiresAt) <= time.Hour {
		t.Errorf("过期时间应约 2 小时后，实际为 %v", claims.ExpiresAt)
	}
}

func TestVerifyExpired(t *testing.T) {
	token, err := Sign("test-secret", "alice", -time.Second)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	if _, err := VerifyToken(token, "test-secret"); err == nil {
		t.Error("过期 token 应验证失败，实际通过")
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	token, err := Sign("secret-a", "alice", time.Hour)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	if _, err := VerifyToken(token, "secret-b"); err == nil {
		t.Error("错误密钥应验证失败，实际通过")
	}
}

func TestVerifyTampered(t *testing.T) {
	token, err := Sign("test-secret", "alice", time.Hour)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	parts := strings.Split(token, ".")
	// 篡改载荷最后一位。
	last := parts[1][:len(parts[1])-1] + "A"
	tampered := parts[0] + "." + last + "." + parts[2]
	if _, err := VerifyToken(tampered, "test-secret"); err == nil {
		t.Error("篡改后的 token 应验证失败，实际通过")
	}
}

func TestEmptySecretRefused(t *testing.T) {
	if _, err := Sign("", "alice", time.Hour); err == nil {
		t.Error("空密钥不得签发 token，实际成功")
	}
	token, err := Sign("test-secret", "alice", time.Hour)
	if err != nil {
		t.Fatalf("Sign 失败: %v", err)
	}
	if _, err := VerifyToken(token, ""); err == nil {
		t.Error("空密钥不得校验 token，实际通过")
	}
}

func TestVerifyMalformed(t *testing.T) {
	for _, bad := range []string{"", "abc", "a.b", "a.b.c.d"} {
		if _, err := VerifyToken(bad, "test-secret"); err == nil {
			t.Errorf("非法 token %q 应验证失败，实际通过", bad)
		}
	}
}
