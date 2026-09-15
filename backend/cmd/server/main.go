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
	"time"
)

func main() {
	cfg := config.Load()
	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.AutoMigrate(&model.User{}, &model.Repair{}, &model.Payment{}, &model.Announcement{}, &model.AnnouncementRead{}, &model.OperationLog{}, &model.Role{}, &model.Permission{}, &model.RolePermission{}); err != nil {
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
	sv := router.Services{Users: service.NewUserService(ur, logger), Repairs: service.NewRepairService(rr, ur, logger), Payments: service.NewPaymentService(pr, logger), Announcements: service.NewAnnouncementService(ar, logger), Permissions: service.NewPermissionService(), Logs: service.NewOperationLogService(lr, logger)}
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
	for _, p := range []model.Permission{{Code: "repair:manage", Name: "报修管理"}, {Code: "payment:manage", Name: "收费管理"}, {Code: "announcement:publish", Name: "公告发布"}, {Code: "log:read", Name: "日志查看"}} {
		if e = db.Create(&p).Error; e != nil {
			return e
		}
	}
	fmt.Print("")
	return nil
}
