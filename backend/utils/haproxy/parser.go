package haproxy

import (
	"strconv"
	"strings"
)

// StatRow 表示 show stat 输出的一行
// 参考 HAProxy 官方字段：https://docs.haproxy.org/2.8/management.html#9.1
type StatRow struct {
	PxName   string // # pxname
	SvName   string // svname (FRONTEND / BACKEND / <server-name>)
	Status   string // UP / DOWN / MAINT / no check / OPEN ...
	Scur     uint64 // 当前会话数
	Smax     uint64
	Slim     uint64
	Stot     uint64 // 累计会话数
	Bin      uint64 // 字节入
	Bout     uint64 // 字节出
	Weight   int
	Act      int // active server 数
	Bck      int // backup server 数
	CheckSt  string
	Lastchg  uint64
	Rate     uint64
	ReqRate  uint64
	ReqTot   uint64
}

// ParseStatCSV 解析 show stat 返回的 CSV 文本
func ParseStatCSV(data string) []StatRow {
	var rows []StatRow
	lines := strings.Split(data, "\n")
	if len(lines) == 0 {
		return rows
	}
	var headers []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			headers = strings.Split(strings.TrimPrefix(line, "# "), ",")
			continue
		}
		if headers == nil {
			continue
		}
		fields := strings.Split(line, ",")
		m := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(fields) {
				m[h] = fields[i]
			}
		}
		row := StatRow{
			PxName:  m["pxname"],
			SvName:  m["svname"],
			Status:  m["status"],
			Scur:    atoiU(m["scur"]),
			Smax:    atoiU(m["smax"]),
			Slim:    atoiU(m["slim"]),
			Stot:    atoiU(m["stot"]),
			Bin:     atoiU(m["bin"]),
			Bout:    atoiU(m["bout"]),
			Weight:  atoiI(m["weight"]),
			Act:    atoiI(m["act"]),
			Bck:    atoiI(m["bck"]),
			CheckSt: m["check_status"],
			Lastchg: atoiU(m["lastchg"]),
			Rate:    atoiU(m["rate"]),
			ReqRate: atoiU(m["req_rate"]),
			ReqTot:  atoiU(m["req_tot"]),
		}
		rows = append(rows, row)
	}
	return rows
}

func atoiU(s string) uint64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func atoiI(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return v
}

// ParseVersion 从 "haproxy -v" 的输出中提取版本号
// 典型输出:
// HAProxy version 2.4.22-0ubuntu0.22.04.3 ...
// HAProxy version 2.8.5-1~bpo12+1 ...
func ParseVersion(out string) string {
	out = strings.TrimSpace(out)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		low := strings.ToLower(line)
		if !strings.HasPrefix(low, "haproxy version") && !strings.HasPrefix(low, "ha-proxy version") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			return fields[2]
		}
	}
	return ""
}
