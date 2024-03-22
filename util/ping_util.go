package util

import (
	"net"
	"time"
)

func GetTCPSpeed(url string, resultChan chan string) {
	timeout := time.Duration(500 * time.Millisecond)
	start := time.Now()
	conn, err := net.DialTimeout("tcp", url, timeout)
	if err != nil {
		resultChan <- "timeout"
		return
	}
	defer conn.Close() // 确保连接在函数返回时被关闭
	elapsed := time.Since(start)
	resultChan <- elapsed.String()
}

// 将任何链接转换为IP+端口的形式
func urlToIPP(rawURL string) string {

	return ""

}
