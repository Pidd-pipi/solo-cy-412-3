package main

import (
	"fmt"
	"github.com/smartestate/smartestate/internal/config"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"github.com/smartestate/smartestate/internal/router"
	"github.com/smartestate/smartestate/internal/service"
	"github.com/smartestate/smartestate/internal/util"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"log/slog"
	"time"
)

func main() {
	cfg := config.Load()
	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.AutoMigrate(&model.User{}, &model.Repair{}, &model.Payment{}, &model.Announcement{}, &model.AnnouncementRead{}, &model.OperationLog{}, &model.Role{}, &model.Permission{}, &model.RolePermission{}, &model.VisitorPass{}, &model.VisitorEvent{}, &model.BuildingCapacity{}); err != nil {
		log.Fatal(err)
	}
	if err = seed(db); err != nil {
		log.Fatal(err)
	}
	logger := util.NewLogger()
	ur := repository.NewUserRepository(db)
	rr := repository.NewRepairRepository(db)
	pr := repository.NewPaymentRepository(db)
	ar := repository.NewAnnouncementRepository(db)
	lr := repository.NewOperationLogRepository(db)
	vpr := repository.NewVisitorPassRepository(db)
	ver := repository.NewVisitorEventRepository(db)
	bcr := repository.NewBuildingCapacityRepository(db)
	logSvc := service.NewOperationLogService(lr, logger)
	visitorSvc := service.NewVisitorService(db, vpr, ver, bcr, logSvc, logger)
	sv := router.Services{Users: service.NewUserService(ur, logger), Repairs: service.NewRepairService(rr, ur, logger), Payments: service.NewPaymentService(pr, logger), Announcements: service.NewAnnouncementService(ar, logger), Visitors: visitorSvc, Permissions: service.NewPermissionService(), Logs: logSvc}
	// 逾期兜底：每分钟把超过离场时间仍未离场的凭证标记逾期，并恢复楼栋在场容量。
	go runExpirySweep(visitorSvc, logger)
	log.Printf("SmartEstate server listening on :%s", cfg.Port)
	if err = router.New(cfg, sv, logger).Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
func openDB(c config.Config) (*gorm.DB, error) {
	if c.DBDriver == "mysql" {
		return gorm.Open(mysql.Open(c.DSN), &gorm.Config{})
	}
	return gorm.Open(sqlite.Open(c.DSN), &gorm.Config{})
}
func seed(db *gorm.DB) error {
	var n int64
	if e := db.Model(&model.User{}).Count(&n).Error; e != nil {
		return e
	}
	if n > 0 {
		return nil
	}
	hash, e := util.HashPassword("password123")
	if e != nil {
		return e
	}
	users := []model.User{{Phone: "13800000001", PasswordHash: hash, Nickname: "张业主", Role: constants.UserRoleResident, Building: "1栋", Unit: "2单元", Room: "802"}, {Phone: "13800000002", PasswordHash: hash, Nickname: "王管家", Role: constants.UserRoleStaff}, {Phone: "13800000003", PasswordHash: hash, Nickname: "系统管理员", Role: constants.UserRoleAdmin}}
	if e = db.Create(&users).Error; e != nil {
		return e
	}
	if e = db.Create(&model.Repair{UserID: users[0].ID, Title: "客厅灯具闪烁", Description: "晚间开灯时出现闪烁，请安排师傅检查。", Type: "水电", Status: constants.RepairStatusPending}).Error; e != nil {
		return e
	}
	if e = db.Create(&model.Payment{UserID: users[0].ID, FeeType: "物业费", Amount: 268.50, Month: "2026-08", Status: "unpaid"}).Error; e != nil {
		return e
	}
	if e = db.Create(&model.Announcement{Title: "夏季消防安全提醒", Content: "请勿在楼道堆放杂物，保持消防通道畅通。", Category: "紧急", PublisherID: users[1].ID, PublishAt: time.Now(), Top: true}).Error; e != nil {
		return e
	}
	for _, p := range []model.Permission{{Code: "repair:manage", Name: "报修管理"}, {Code: "payment:manage", Name: "收费管理"}, {Code: "announcement:publish", Name: "公告发布"}, {Code: "log:read", Name: "日志查看"}, {Code: "visitor:review", Name: "访客凭证审核"}, {Code: "visitor:gate", Name: "门岗通行核对"}} {
		if e = db.Create(&p).Error; e != nil {
			return e
		}
	}
	// 预置业主所在楼栋的当日访客容量。
	if e = db.Create(&model.BuildingCapacity{Building: "1栋", DailyLimit: constants.DefaultBuildingDailyLimit}).Error; e != nil {
		return e
	}
	fmt.Print("")
	return nil
}

// runExpirySweep 周期执行访客逾期兜底，避免协程异常退出。
func runExpirySweep(visitors *service.VisitorService, logger *slog.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		n, e := visitors.SweepExpired()
		if e != nil {
			logger.Error("visitor expiry sweep", "error", e)
			continue
		}
		if n > 0 {
			logger.Info("visitor expiry sweep marked", "count", n)
		}
	}
}
