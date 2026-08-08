package util

import "math/rand"

// RandInt 返回 [min, max) 范围内的随机整数。
// Go 1.20+ 自动 seed 全局 PRNG, 无需手动 Seed(deprecated 且非线程安全写)。
func RandInt(min, max int) int {
	return min + rand.Intn(max-min)
}
