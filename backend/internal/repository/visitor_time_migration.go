package repository

import (
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

// NormalizeVisitorTimes 升级期一次性、幂等的数据迁移：
// 旧版本按进程/社区时区把到访时刻存成带偏移的字符串（如 ...T23:30:00+08:00），
// 新版本统一以 UTC（...Z）入库。读取会把带偏移串解析为正确的绝对时刻，
// 这里再借模型 BeforeSave 钩子把存量行回写为 UTC，使升级后跨新旧记录的
// 重叠/到期等时间范围比较在 SQLite（字符串比较）与 MySQL 上结果一致。
// 钟面时刻不变（绝对时刻不变，仅规范化存储表示），可安全重复执行。
func NormalizeVisitorTimes(db *gorm.DB) (int64, error) {
	var passes []model.VisitorPass
	if e := db.Find(&passes).Error; e != nil {
		return 0, e
	}
	var n int64
	for i := range passes {
		tx := db.Save(&passes[i])
		if tx.Error != nil {
			return n, tx.Error
		}
		n += tx.RowsAffected
	}
	var events []model.VisitorEvent
	if e := db.Find(&events).Error; e != nil {
		return n, e
	}
	for i := range events {
		if e := db.Save(&events[i]).Error; e != nil {
			return n, e
		}
	}
	return n, nil
}
