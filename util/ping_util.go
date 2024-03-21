package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
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

func urlToIPP(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("解析URL出错:", err)
		return ""
	}

	// 获取主机部分（IP地址和端口）
	host := parsedURL.Host

	// 查找第一个斜杠的位置
	slashIndex := strings.Index(host, "/")
	if slashIndex != -1 {
		// 去除斜杠后面的路径部分
		host = host[:slashIndex]
	}
	return host

}
