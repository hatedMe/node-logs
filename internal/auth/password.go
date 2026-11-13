// Package auth 提供密码哈希、JWT 签发校验与认证中间件。
package auth

import "golang.org/x/crypto/bcrypt"

// Hash 用 bcrypt 对密码求哈希。
func Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Verify 校验密码与哈希是否匹配。
func Verify(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
