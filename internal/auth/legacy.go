package auth

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

// MD5Upper 计算与旧 Node 端一致的 MD5 大写十六进制摘要。
// 旧实现：crypto.createHash('md5').update(str, 'utf8').digest('hex').toUpperCase()。
func MD5Upper(password string) string {
	sum := md5.Sum([]byte(password))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// IsMD5Hash 判断存储哈希是否为旧 MD5 格式（32 位十六进制，不分大小写）。
// bcrypt 哈希以 $2 开头，不会误判。
func IsMD5Hash(hash string) bool {
	if len(hash) != 32 {
		return false
	}
	for _, c := range hash {
		if !isHexChar(c) {
			return false
		}
	}
	return true
}

func isHexChar(c rune) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}
