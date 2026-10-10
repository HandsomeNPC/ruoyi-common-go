package ip

import (
	_ "embed"
	"net"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// v4Content 内嵌的 IPv4 xdb 地址库。
//
//go:embed ip2region_v4.xdb
var v4Content []byte

// ResolveRealAddressByIP 离线查询 IP 的行政区域。
// ipStr: 待查 IP。
// 返回: 内网（私网/回环/链路本地/未指定）返回 "内网IP"；非 IPv4、解析失败或无结果返回 "未知"。
// 仅 IPv4 带库，IPv6 暂不支持。每次调用新建 Searcher 复用只读 buffer，避免并发写 ioCount 的竞争。
func ResolveRealAddressByIP(ipStr string) string {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return "未知"
	}
	p := net.ParseIP(ipStr)
	if p == nil {
		return "未知"
	}
	if p.IsPrivate() || p.IsLoopback() || p.IsLinkLocalUnicast() || p.IsUnspecified() {
		return "内网IP"
	}
	v4 := p.To4()
	if v4 == nil {
		return "未知"
	}
	s, err := xdb.NewWithBuffer(xdb.IPv4, v4Content)
	if err != nil {
		return "未知"
	}
	region, err := s.Search(v4.String())
	if err != nil || region == "" {
		return "未知"
	}
	return region
}
