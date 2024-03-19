package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

func GetTCPSpeed(url string) string {
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
