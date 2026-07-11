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
	// 去掉前面有没有http://或者https://,先做长度判断避免切片越界
	if len(rawURL) >= 7 && rawURL[:7] == "http://" {
		rawURL = rawURL[7:]
	} else if len(rawURL) >= 8 && rawURL[:8] == "https://" {
		rawURL = rawURL[8:]
	}
	// 去掉用户信息(如 user:pass@host)以及路径,只保留 host:port
	if idx := strings.Index(rawURL, "@"); idx != -1 {
		rawURL = rawURL[idx+1:]
	}
	if idx := strings.Index(rawURL, "/"); idx != -1 {
		rawURL = rawURL[:idx]
	}

	return rawURL
}
