package util

import (
	"crypto/rand"
)

const letters = "a2b1cdef4gh34ijk453lmn3opqrstuvwxytyza66zABCDmutant4catEFGH22IJK33LMN44OPQR44ST23UVWX12YZ"

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
