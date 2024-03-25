package util

import (
	"net"
	"strings"
	"time"
)

func GetTCPSpeed(url string, resultChan chan string) {
	// 先把地址形式转换为ip:port
	url = urlToIPP(url)
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
	// 去过前面有没有http://或者https://
	if rawURL[:7] == "http://" {
		rawURL = rawURL[7:]
	}
	if rawURL[:8] == "https://" {
		rawURL = rawURL[8:]
	}
	// 如果后面有任何路径，去掉路径
	if strings.Contains(rawURL, "/") {
		rawURL = rawURL[:strings.Index(rawURL, "/")]
	}

	return rawURL
}
