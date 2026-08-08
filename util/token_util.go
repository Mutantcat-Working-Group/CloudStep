package util

import (
	"crypto/rand"
)

// 64 字符字母表(A-Z a-z 0-9 + 2 填充); 256 % 64 == 0 → v%lettersLen 无偏差。
const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"

// 生成随机字符串
func RandToken(n int) string {
	b := make([]byte, n)
	// crypto/rand 读取失败时回退到空 token,由调用方决定如何处理
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	lettersLen := byte(len(letters))
	for i, v := range b {
		b[i] = letters[v%lettersLen]
	}
	return string(b)
}
