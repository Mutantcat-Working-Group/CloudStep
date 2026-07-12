package util

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// HMACSHA256Hex 以 key 对 message 做 HMAC-SHA256,返回小写 hex 字符串。
// 用于 way 请求层签名校验(salted mode): 客户端把同样的 hex 放在 salt= 参数里,
// 服务端重算后做不区分大小写的比对。
func HMACSHA256Hex(key string, message string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
