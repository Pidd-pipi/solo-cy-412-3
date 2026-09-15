package service

import (
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newConcurrentFixture 使用文件库 + WAL + busy_timeout + 多连接，模拟生产 MySQL 的并发事务语义。
func newConcurrentFixture(t *testing.T) (*VisitorService, *gorm.DB, model.User, model.User) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "conc.db") + "?_busy_timeout=8000&_journal_mode=WAL&_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(8)
	}
	if err = db.AutoMigrate(&model.User{}, &model.VisitorPass{}, &model.VisitorEvent{}, &model.BuildingCapacity{}, &model.OperationLog{}); err != nil {
		t.Fatal(err)
	}
	resident := model.User{Phone: "138000010001", Nickname: "业主", Role: "resident", Building: "1栋"}
	other := model.User{Phone: "138000010002", Nickname: "业主乙", Role: "resident", Building: "1栋"}
	staff := model.User{Phone: "138000010003", Nickname: "物业", Role: "staff"}
	db.Create(&resident)
	db.Create(&other)
	db.Create(&staff)
	logger := slog.Default()
	logs := NewOperationLogService(repository.NewOperationLogRepository(db), logger)
	svc := NewVisitorService(db, repository.NewVisitorPassRepository(db), repository.NewVisitorEventRepository(db), repository.NewBuildingCapacityRepository(db), logs, logger)
	return svc, db, resident, staff
}

func countEvents(db *gorm.DB, passID uint, action string) int64 {
	var n int64
	db.Model(&model.VisitorEvent{}).Where("pass_id = ? AND action = ?", passID, action).Count(&n)
	return n
}

// 两名物业并发审核同一张凭证：只能成功一次，且只有一条审核留痕。
func TestConcurrentApproveSamePass(t *testing.T) {
	svc, db, resident, staff := newConcurrentFixture(t)
	start, end := time.Now().Add(-time.Hour), time.Now().Add(2*time.Hour)
	pass, err := svc.Create(resident.ID, "resident", "访客", "13910000001", "1栋", "探亲", start, end)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var ok, fail int64
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := svc.Approve(pass.ID, staff.ID, ""); e != nil {
				mu.Lock()
				fail++
				mu.Unlock()
			} else {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 || fail != 7 {
		t.Fatalf("approve same pass want exactly 1 success, got ok=%d fail=%d", ok, fail)
	}
	if n := countEvents(db, pass.ID, constants.PassActionApprove); n != 1 {
		t.Fatalf("want 1 approve event, got %d", n)
	}
	var got model.VisitorPass
	db.First(&got, pass.ID)
	if got.Status != constants.PassStatusApproved {
		t.Fatalf("final status=%s", got.Status)
	}
}

// 楼栋上限 1：并发审核一批不同凭证，只有一张能通过，已承诺名额绝不越过上限。
func TestConcurrentApproveBatchCapacity(t *testing.T) {
	svc, db, resident, staff := newConcurrentFixture(t)
	if _, err := svc.SetCapacity("1栋", 1, staff.ID); err != nil {
		t.Fatal(err)
	}
	start, end := time.Now().Add(-time.Hour), time.Now().Add(2*time.Hour)
	const n = 6
	ids := make([]uint, 0, n)
	for i := 0; i < n; i++ {
		p, err := svc.Create(resident.ID, "resident", "访客", "1392000000"+string(rune('1'+i)), "1栋", "送货", start, end)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	var wg sync.WaitGroup
	var ok int64
	var mu sync.Mutex
	for _, id := range ids {
		wg.Add(1)
		go func(pid uint) {
			defer wg.Done()
			if _, e := svc.Approve(pid, staff.ID, ""); e == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(id)
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("capacity=1 want exactly 1 approved, got %d", ok)
	}
	var committed int64
	db.Model(&model.VisitorPass{}).
		Where("building = ? AND status IN ?", "1栋", []string{constants.PassStatusApproved, constants.PassStatusCheckedIn}).
		Count(&committed)
	if committed != 1 {
		t.Fatalf("committed must not exceed limit, got %d", committed)
	}
}

// 同一凭证并发办理进入：只成功一次、一条进入留痕，在场数为 1，终态不被改写。
func TestConcurrentCheckInOnce(t *testing.T) {
	svc, db, resident, staff := newConcurrentFixture(t)
	start, end := time.Now().Add(-time.Hour), time.Now().Add(2*time.Hour)
	pass, _ := svc.Create(resident.ID, "resident", "访客", "13930000001", "1栋", "探亲", start, end)
	if _, err := svc.Approve(pass.ID, staff.ID, ""); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var ok, fail int64
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := svc.CheckIn(pass.ID, staff.ID, "东门"); e != nil {
				mu.Lock()
				fail++
				mu.Unlock()
			} else {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 || fail != 7 {
		t.Fatalf("checkin want exactly 1 success, got ok=%d fail=%d", ok, fail)
	}
	if n := countEvents(db, pass.ID, constants.PassActionCheckIn); n != 1 {
		t.Fatalf("want 1 checkin event, got %d", n)
	}
	if onsite, _ := repository.NewVisitorPassRepository(db).CountInBuilding("1栋", time.Now(), nil); onsite != 1 {
		t.Fatalf("onsite want 1, got %d", onsite)
	}
}

// 同一凭证并发办理离开：只成功一次、一条离开留痕，终态保持 completed。
func TestConcurrentCheckOutOnce(t *testing.T) {
	svc, db, resident, staff := newConcurrentFixture(t)
	start, end := time.Now().Add(-time.Hour), time.Now().Add(2*time.Hour)
	pass, _ := svc.Create(resident.ID, "resident", "访客", "13940000001", "1栋", "探亲", start, end)
	svc.Approve(pass.ID, staff.ID, "")
	if _, err := svc.CheckIn(pass.ID, staff.ID, "东门"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var ok, fail int64
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := svc.CheckOut(pass.ID, staff.ID, "东门"); e != nil {
				mu.Lock()
				fail++
				mu.Unlock()
			} else {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 || fail != 7 {
		t.Fatalf("checkout want exactly 1 success, got ok=%d fail=%d", ok, fail)
	}
	if n := countEvents(db, pass.ID, constants.PassActionCheckOut); n != 1 {
		t.Fatalf("want 1 checkout event, got %d", n)
	}
	if onsite, _ := repository.NewVisitorPassRepository(db).CountInBuilding("1栋", time.Now(), nil); onsite != 0 {
		t.Fatalf("onsite want 0 after checkout, got %d", onsite)
	}
}

// 同一访客同楼栋重叠时段并发登记：只允许一张成功。
func TestConcurrentOverlappingCreate(t *testing.T) {
	svc, db, resident, _ := newConcurrentFixture(t)
	start, end := time.Now().Add(time.Hour), time.Now().Add(3*time.Hour)
	var wg sync.WaitGroup
	var ok, fail int64
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := svc.Create(resident.ID, "resident", "同访客", "13950000001", "1栋", "探亲", start, end); e != nil {
				mu.Lock()
				fail++
				mu.Unlock()
			} else {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if ok != 1 || fail != 7 {
		t.Fatalf("overlapping create want exactly 1 success, got ok=%d fail=%d", ok, fail)
	}
	var cnt int64
	db.Model(&model.VisitorPass{}).Where("visitor_phone = ? AND building = ?", "13950000001", "1栋").Count(&cnt)
	if cnt != 1 {
		t.Fatalf("want exactly 1 pass row, got %d", cnt)
	}
}

// 授权：登记仅业主；取消仅凭证登记业主本人。
func TestVisitorAuthorization(t *testing.T) {
	svc, _, resident, staff := newConcurrentFixture(t)
	start, end := time.Now().Add(time.Hour), time.Now().Add(3*time.Hour)
	pass, err := svc.Create(resident.ID, "resident", "访客", "13960000001", "1栋", "探亲", start, end)
	if err != nil {
		t.Fatal(err)
	}
	// 物业不能代替业主登记。
	if _, e := svc.Create(staff.ID, "staff", "访客", "13960000002", "1栋", "代登记", start, end); !errors.Is(e, ErrPassForbidden) {
		t.Fatalf("staff create should be forbidden, got %v", e)
	}
	// 物业不能取消他人凭证。
	if _, e := svc.Cancel(pass.ID, staff.ID, "staff"); !errors.Is(e, ErrPassForbidden) {
		t.Fatalf("staff cancel should be forbidden, got %v", e)
	}
	// 审核后物业仍不能取消业主凭证（只能业主本人取消）。
	if _, e := svc.Approve(pass.ID, staff.ID, ""); e != nil {
		t.Fatal(e)
	}
	if _, e := svc.Cancel(pass.ID, staff.ID, "admin"); !errors.Is(e, ErrPassForbidden) {
		t.Fatalf("admin cancel should be forbidden, got %v", e)
	}
	// 业主本人可取消自己的凭证。
	if _, e := svc.Cancel(pass.ID, resident.ID, "resident"); e != nil {
		t.Fatalf("owner cancel should succeed, got %v", e)
	}
}
