package util

import (
	"net/netip"
	"strings"
)

// 私有/保留地址类别,命中则表示该地址不可被公开代理。
const (
	// PrivateCategory 内网私有地址,如 10/8、172.16/12、192.168/16
	PrivateCategory = "private"
	// LoopbackCategory 回环地址 127.0.0.0/8、::1
	LoopbackCategory = "loopback"
	// LinkLocalCategory 链路本地地址 169.254.0.0/16(含云厂商 metadata)、fe80::/10
	LinkLocalCategory = "linklocal"
	// UnspecifiedCategory 未指定地址 0.0.0.0、::
	UnspecifiedCategory = "unspecified"
)

// ClassifyIP 对 host 字符串做 IP 分类。若 host 不是合法 IP(如域名),返回 ("", false)。
// 命中内网/回环/链路本地/未指定时返回对应类别与 true。
func ClassifyIP(host string) (string, bool) {
	// 去掉端口,netip.ParseAddr 不接受 host:port
	if h, _, err := splitHostPort(host); err == nil {
		host = h
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return "", false
	}
	switch {
	case addr.IsUnspecified():
		return UnspecifiedCategory, true
	case addr.IsLoopback():
		return LoopbackCategory, true
	case addr.IsLinkLocalUnicast(), addr.IsLinkLocalMulticast():
		return LinkLocalCategory, true
	case addr.IsPrivate():
		return PrivateCategory, true
	default:
		return "", false
	}
}

// splitHostPort 简单拆分 host:port。
// 支持: "host:port"、"[::1]:8080"、"::1"(裸 IPv6 无端口)。
// 对包含多个 ':' 且不以 ']' 结尾的输入,视为裸 IPv6 地址(不拆分端口)。
func splitHostPort(hostPort string) (string, string, error) {
	// 带方括号的 IPv6:[host]:port
	if strings.HasPrefix(hostPort, "[") {
		if i := strings.LastIndex(hostPort, "]"); i != -1 {
			host := hostPort[1:i]
			rest := hostPort[i+1:]
			if strings.HasPrefix(rest, ":") {
				return host, rest[1:], nil
			}
			return host, "", nil
		}
		return hostPort, "", nil
	}
	// 只有一个冒号是 host:port;多个冒号是裸 IPv6
	if strings.Count(hostPort, ":") == 1 {
		idx := strings.Index(hostPort, ":")
		return hostPort[:idx], hostPort[idx+1:], nil
	}
	return hostPort, "", nil
}
