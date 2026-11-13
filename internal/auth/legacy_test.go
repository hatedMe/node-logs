package auth

import (
	"testing"
)

func TestMD5Upper(t *testing.T) {
	// 与旧 Node md5(str).toUpperCase() 对齐。
	if got := MD5Upper("123456"); got != "E10ADC3949BA59ABBE56E057F20F883E" {
		t.Errorf("MD5Upper(123456) 错误，实际为 %q", got)
	}
}

func TestIsMD5Hash(t *testing.T) {
	if !IsMD5Hash("E10ADC3949BA59ABBE56E057F20F883E") {
		t.Error("大写 32 位十六进制应判定为 MD5 哈希")
	}
	if !IsMD5Hash("e10adc3949ba59abbe56e057f20f883e") {
		t.Error("小写 32 位十六进制应判定为 MD5 哈希")
	}
	for _, not := range []string{"", "short", "$2a$10$" + "abcdefghijklmnopqrstuu", "E10ADC3949BA59ABBE56E057F20F883EX"} {
		if IsMD5Hash(not) {
			t.Errorf("%q 不应判定为 MD5 哈希", not)
		}
	}
}
