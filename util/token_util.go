package util

import (
	"math/rand"
	"time"
)

const letters = "a2b1cdef4gh34ijk453lmn3opqrstuvwxytyza66zABCDmutant4catEFGH22IJK33LMN44OPQR44ST23UVWX12YZ"

// 生成随机字符串
func RandToken(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
