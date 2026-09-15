package service

import (
	"log/slog"
	"testing"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newVisitorFixture(t *testing.T) (*VisitorService, *gorm.DB, model.User, model.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// SQLite :memory: 为每个连接保留独立内存库，强制单连接保证 DDL 与后续查询同库。
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err = db.AutoMigrate(&model.User{}, &model.VisitorPass{}, &model.VisitorEvent{}, &model.BuildingCapacity{}, &model.OperationLog{}); err != nil {
		t.Fatal(err)
	}
	resident := model.User{Phone: "13800000001", Nickname: "业主", Role: "resident", Building: "1栋"}
	staff := model.User{Phone: "13800000002", Nickname: "物业", Role: "staff"}
	db.Create(&resident)
	db.Create(&staff)
	logger := slog.Default()
	logs := NewOperationLogService(repository.NewOperationLogRepository(db), logger)
	svc := NewVisitorService(db, repository.NewVisitorPassRepository(db), repository.NewVisitorEventRepository(db), repository.NewBuildingCapacityRepository(db), logs, logger)
	return svc, db, resident, staff
}

func TestVisitorPassLifecycleAndCapacity(t *testing.T) {
	svc, db, resident, staff := newVisitorFixture(t)
	building := "1栋"
	// 容量上限设为 1。
	if _, err := svc.SetCapacity(building, 1, staff.ID); err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-time.Hour)
	end := time.Now().Add(2 * time.Hour)

	pass, err := svc.Create(resident.ID, "resident", "张三", "13900000001", building, "探亲", start, end)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if pass.Status != constants.PassStatusPending {
		t.Fatalf("want pending, got %s", pass.Status)
	}

	// 重叠时段登记同一访客同一楼栋应被拒绝。
	if _, err = svc.Create(resident.ID, "resident", "张三", "13900000001", building, "探亲", start.Add(30*time.Minute), end.Add(time.Hour)); err == nil {
		t.Fatal("expected overlap conflict")
	}

	// 审核通过。
	pass, err = svc.Approve(pass.ID, staff.ID, "")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}

	// 办理进入，在场容量变为 1。
	pass, err = svc.CheckIn(pass.ID, staff.ID, "东门")
	if err != nil {
		t.Fatalf("checkin: %v", err)
	}
	if n, _ := repository.NewVisitorPassRepository(db).CountInBuilding(building, time.Now(), nil); n != 1 {
		t.Fatalf("onsite want 1 got %d", n)
	}

	// 重复进入无效。
	if _, err = svc.CheckIn(pass.ID, staff.ID, "东门"); err == nil {
		t.Fatal("expected duplicate checkin rejected")
	}

	// 第二张凭证：不同访客，审核时因容量已满被暂停。
	start2 := time.Now().Add(-30 * time.Minute)
	end2 := time.Now().Add(3 * time.Hour)
	pass2, err := svc.Create(resident.ID, "resident", "李四", "13900000002", building, "访友", start2, end2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Approve(pass2.ID, staff.ID, ""); err == nil {
		t.Fatal("expected capacity reached on approve")
	}

	// 办理离开，容量恢复，第二张凭证可审核通过。
	if _, err = svc.CheckOut(pass.ID, staff.ID, "东门"); err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if n, _ := repository.NewVisitorPassRepository(db).CountInBuilding(building, time.Now(), nil); n != 0 {
		t.Fatalf("onsite want 0 got %d", n)
	}
	if _, err = svc.Approve(pass2.ID, staff.ID, ""); err != nil {
		t.Fatalf("approve after release: %v", err)
	}

	// 已完成的凭证不能再放行。
	if _, err = svc.CheckIn(pass.ID, staff.ID, "东门"); err == nil {
		t.Fatal("completed pass must not be re-checkin")
	}
}

func TestVisitorGateWindowAndCancel(t *testing.T) {
	svc, _, resident, staff := newVisitorFixture(t)
	// 未到时段。
	future := time.Now().Add(2 * time.Hour)
	pass, err := svc.Create(resident.ID, "resident", "王五", "13900000003", "2栋", "送货", future, future.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if pass, err = svc.Approve(pass.ID, staff.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.CheckIn(pass.ID, staff.ID, "西门"); err == nil {
		t.Fatal("before window must be rejected")
	}
	// 业主取消后不能放行。
	if _, err = svc.Cancel(pass.ID, resident.ID, "resident"); err != nil {
		t.Fatal(err)
	}
	if _, info, verr := svc.GateVerify(pass.PassNo, staff.ID); verr != nil || info["allow"] != false {
		t.Fatalf("cancelled pass allow=false expected, err=%v info=%v", verr, info)
	}
}

func TestVisitorExpirySweepRestoresCapacity(t *testing.T) {
	svc, db, resident, staff := newVisitorFixture(t)
	building := "3栋"
	if _, err := svc.SetCapacity(building, 1, staff.ID); err != nil {
		t.Fatal(err)
	}
	// 已过期的在场凭证。
	pass, err := svc.Create(resident.ID, "resident", "赵六", "13900000004", building, "维修", time.Now().Add(-3*time.Hour), time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	// 直接构造 approved→checkin 以模拟已进入（绕过时间窗校验）。
	if err = db.Model(&model.VisitorPass{}).Where("id = ?", pass.ID).Updates(map[string]any{"status": "checked_in", "check_in_at": time.Now().Add(-2 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	if n, _ := repository.NewVisitorPassRepository(db).CountInBuilding(building, time.Now(), nil); n != 1 {
		t.Fatalf("onsite want 1 got %d", n)
	}
	marked, err := svc.SweepExpired()
	if err != nil || marked != 1 {
		t.Fatalf("sweep marked=%d err=%v", marked, err)
	}
	if n, _ := repository.NewVisitorPassRepository(db).CountInBuilding(building, time.Now(), nil); n != 0 {
		t.Fatalf("onsite after sweep want 0 got %d", n)
	}
	got, _, err := svc.Detail(pass.ID, staff.ID, "staff")
	if err != nil || got.Status != constants.PassStatusExpired {
		t.Fatalf("status want expired got %s err=%v", got.Status, err)
	}
}
