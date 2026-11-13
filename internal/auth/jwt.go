package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claims 是 JWT 载荷。
type Claims struct {
	Username  string
	ExpiresAt time.Time
}

var b64 = base64.RawURLEncoding

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

// Sign 签发 HS256 JWT，ttl 后过期；空密钥拒绝签发。
func Sign(secret, username string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("auth: JWT 密钥为空，拒绝签发")
	}
	header, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(jwtPayload{Sub: username, Exp: time.Now().Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	unsigned := b64.EncodeToString(header) + "." + b64.EncodeToString(payload)
	sig := hmacSHA256(secret, unsigned)
	return unsigned + "." + b64.EncodeToString(sig), nil
}

// VerifyToken 校验 HS256 JWT 并返回载荷；签名错误、过期或格式非法时返回错误。
func VerifyToken(token, secret string) (Claims, error) {
	if secret == "" {
		return Claims{}, errors.New("auth: JWT 密钥为空，拒绝校验")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("auth: token 格式非法")
	}
	var header jwtHeader
	if raw, err := b64.DecodeString(parts[0]); err != nil {
		return Claims{}, errors.New("auth: token 头部非法")
	} else if err := json.Unmarshal(raw, &header); err != nil || header.Alg != "HS256" {
		return Claims{}, errors.New("auth: 不支持的签名算法")
	}
	want := hmacSHA256(secret, parts[0]+"."+parts[1])
	got, err := b64.DecodeString(parts[2])
	if err != nil || subtle.ConstantTimeCompare(got, want) != 1 {
		return Claims{}, errors.New("auth: token 签名无效")
	}
	raw, err := b64.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("auth: token 载荷非法")
	}
	var payload jwtPayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Sub == "" {
		return Claims{}, errors.New("auth: token 载荷非法")
	}
	exp := time.Unix(payload.Exp, 0)
	if !time.Now().Before(exp) {
		return Claims{}, fmt.Errorf("auth: token 已过期")
	}
	return Claims{Username: payload.Sub, ExpiresAt: exp}, nil
}

func hmacSHA256(secret, message string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return mac.Sum(nil)
}
