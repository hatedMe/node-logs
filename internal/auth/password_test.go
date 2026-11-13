package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("s3cret-password")
	if err != nil {
		t.Fatalf("Hash 失败: %v", err)
	}
	if hash == "" || hash == "s3cret-password" {
		t.Error("哈希结果不得为空或明文")
	}
	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("应为 bcrypt 哈希，实际为 %q", hash)
	}
	if err := Verify(hash, "s3cret-password"); err != nil {
		t.Errorf("正确密码应通过验证，实际错误: %v", err)
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	hash, err := Hash("right-password")
	if err != nil {
		t.Fatalf("Hash 失败: %v", err)
	}
	if err := Verify(hash, "wrong-password"); err == nil {
		t.Error("错误密码应验证失败，实际通过")
	}
}

func TestVerifyBadHash(t *testing.T) {
	if err := Verify("not-a-hash", "whatever"); err == nil {
		t.Error("非法哈希应验证失败，实际通过")
	}
}
