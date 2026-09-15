package util

import (
	"os"
	"time"
)

// 社区统一时区：浏览器 datetime-local 提交的是不含时区的本地钟面时刻，
// 后端必须按"小区所在时区"（而非服务器 time.Local，容器内常为 UTC）来解释，
// 否则跨时区部署时开始/结束时刻会整体偏移、跨日还会错一天。
var communityLocation = loadCommunityLocation()

func loadCommunityLocation() *time.Location {
	name := os.Getenv("APP_TIMEZONE")
	if name == "" {
		name = "Asia/Shanghai"
	}
	if loc, e := time.LoadLocation(name); e == nil {
		return loc
	}
	// 极简容器可能没有 tzdata：回退到固定 UTC+8（中国时区无夏令时）。
	return time.FixedZone("UTC+8", 8*60*60)
}

// CommunityLocation 返回访客时段统一使用的社区时区。
func CommunityLocation() *time.Location { return communityLocation }

// InCommunity 把某一绝对时刻转到社区时区，用于按统一钟面渲染与比较。
func InCommunity(t time.Time) time.Time { return t.In(communityLocation) }

// CommunityNow 返回社区时区下的当前时刻。
func CommunityNow() time.Time { return time.Now().In(communityLocation) }

// FormatVisit 按社区时区把到访时刻格式化为分钟精度的本地钟面（YYYY-MM-DD HH:mm）。
// 存储的是绝对时刻，渲染统一在此转换，保证业主、物业、门岗看到同一本地时刻；旧记录原值照显。
func FormatVisit(t time.Time) string { return InCommunity(t).Format("2006-01-02 15:04") }
