package util

import (
	"fmt"
	"net"
	"time"
)

func getSpeed(url string) string {
	timeout := time.Duration(5 * time.Second)
	start := time.Now()
	_, err := net.DialTimeout("tcp", url, timeout)
	if err != nil {
		fmt.Println("Error:", err)
		return "timeout"
	}
	elapsed := time.Since(start)
	return elapsed.String()
}
