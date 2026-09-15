package service

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"github.com/smartestate/smartestate/internal/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 本套测试全程走真实持久化链路：磁盘 SQLite + WAL + busy_timeout + 多连接（8），
// 与生产 MySQL 一样存在真实的事务竞争；不使用 :memory:、mock、单连接，也不串行化请求。
//
// 每个用例的失败信息都以阶段前缀（登记/审核/进入/离开/逾期）标注失败发生环节，
// 并在结束时回读校验：凭证状态、楼栋容量（在场/已承诺/剩余）与留痕条数。

type flowEnv struct {
	t        *testing.T
	svc      *VisitorService
	db       *gorm.DB
	resident model.User
	staff    model.User
	phones   atomic.Int64
}

func newFlowEnv(t *testing.T) *flowEnv {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "flow.db") + "?_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开真实数据库失败: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		// 多连接，制造与生产一致的并发事务竞争（不退回单连接）。
		sqlDB.SetMaxOpenConns(8)
	}
	if err = db.AutoMigrate(&model.User{}, &model.VisitorPass{}, &model.VisitorEvent{}, &model.BuildingCapacity{}, &model.OperationLog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	resident := model.User{Phone: "13809000001", Nickname: "业主", Role: "resident", Building: "1栋"}
	staff := model.User{Phone: "13809000002", Nickname: "门岗物业", Role: "staff"}
	if err = db.Create(&resident).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&staff).Error; err != nil {
		t.Fatal(err)
	}
	logger := slog.Default()
	logs := NewOperationLogService(repository.NewOperationLogRepository(db), logger)
	svc := NewVisitorService(db, repository.NewVisitorPassRepository(db), repository.NewVisitorEventRepository(db), repository.NewBuildingCapacityRepository(db), logs, logger)
	env := &flowEnv{t: t, svc: svc, db: db, resident: resident, staff: staff}
	env.phones.Store(time.Now().UnixNano() % 1000000)
	return env
}

// uniquePhone 返回 11 位、用例内唯一的访客手机号（重叠测试除外，那种场景刻意复用同一号码）。
func (e *flowEnv) uniquePhone() string {
	n := e.phones.Add(1) % 1000000000
	return fmt.Sprintf("13%09d", n)
}

func (e *flowEnv) register(building, phone string, start, end time.Time) model.VisitorPass {
	e.t.Helper()
	p, err := e.svc.Create(e.resident.ID, "resident", "访客张三", phone, building, "来访事由测试", start, end)
	if err != nil {
		e.t.Fatalf("[登记阶段] 创建凭证失败: %v", err)
	}
	return p
}

func (e *flowEnv) approveMust(id uint) model.VisitorPass {
	e.t.Helper()
	p, err := e.svc.Approve(id, e.staff.ID, "")
	if err != nil {
		e.t.Fatalf("[审核阶段] 凭证 %d 应可审核通过，实际失败: %v", id, err)
	}
	return p
}

func (e *flowEnv) checkInMust(id uint) model.VisitorPass {
	e.t.Helper()
	p, err := e.svc.CheckIn(id, e.staff.ID, "东门")
	if err != nil {
		e.t.Fatalf("[进入阶段] 凭证 %d 应可进入，实际失败: %v", id, err)
	}
	return p
}

// runConcurrent 用 barrier 让 n 个 goroutine 真正同时发起，制造竞争而非排队串行。
func runConcurrent(n int, fn func(i int)) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
}

func (e *flowEnv) eventCount(id uint, action string) int64 {
	var n int64
	if err := e.db.Model(&model.VisitorEvent{}).Where("pass_id = ? AND action = ?", id, action).Count(&n).Error; err != nil {
		e.t.Fatalf("回读留痕失败: %v", err)
	}
	return n
}

func (e *flowEnv) rowsByPhoneBuilding(phone, building string) int64 {
	var n int64
	e.db.Model(&model.VisitorPass{}).Where("visitor_phone = ? AND building = ?", phone, building).Count(&n)
	return n
}

type capView struct {
	limit     int
	onsite    int64
	committed int64
	remaining int
}

func (e *flowEnv) capacity(building string) capView {
	rows, err := e.svc.CapacityOverview()
	if err != nil {
		e.t.Fatalf("[容量回读] 失败: %v", err)
	}
	for _, r := range rows {
		if r["building"] == building {
			return capView{
				limit:     int(r["daily_limit"].(int)),
				onsite:    r["onsite"].(int64),
				committed: r["committed"].(int64),
				remaining: r["remaining"].(int),
			}
		}
	}
	e.t.Fatalf("[容量回读] 找不到楼栋 %s", building)
	return capView{}
}

func (e *flowEnv) statusOf(id uint) string {
	var p model.VisitorPass
	if err := e.db.First(&p, id).Error; err != nil {
		e.t.Fatalf("回读凭证 %d 失败: %v", id, err)
	}
	return p.Status
}

func windowActive() (time.Time, time.Time) {
	return time.Now().Add(-time.Hour), time.Now().Add(3 * time.Hour)
}
func windowFuture() (time.Time, time.Time) {
	return time.Now().Add(2 * time.Hour), time.Now().Add(5 * time.Hour)
}

// reg 以当前有效（或未来）时段登记一张凭证。
func (e *flowEnv) reg(building, phone string, future bool) model.VisitorPass {
	s, en := windowActive()
	if future {
		s, en = windowFuture()
	}
	return e.register(building, phone, s, en)
}

// 用例 1：并发登记同一访客同一楼栋的重叠时段，只能成功一张（失败发生在登记阶段）。
func TestFlow_ConcurrentCreateOverlap(t *testing.T) {
	env := newFlowEnv(t)
	building := "并发登记栋"
	phone := "13900001111"
	start, end := windowFuture()

	var mu sync.Mutex
	var success, fail int
	var successID uint
	runConcurrent(8, func(int) {
		p, err := env.svc.Create(env.resident.ID, "resident", "同一位访客", phone, building, "来访事由测试", start, end)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if !errors.Is(err, ErrPassOverlap) {
				t.Errorf("[登记阶段] 期望重叠冲突 ErrPassOverlap，实际: %v", err)
			}
			fail++
			return
		}
		success++
		successID = p.ID
	})
	if success != 1 || fail != 7 {
		t.Fatalf("[登记阶段] 并发重叠登记应恰好 1 成功 7 失败，实际成功=%d 失败=%d", success, fail)
	}
	if got := env.rowsByPhoneBuilding(phone, building); got != 1 {
		t.Fatalf("[登记阶段] 回读凭证行数应为 1，实际 %d", got)
	}
	if n := env.eventCount(successID, constants.PassActionCreate); n != 1 {
		t.Fatalf("[登记阶段] 成功者留痕应为 1 条，实际 %d", n)
	}
	if env.statusOf(successID) != constants.PassStatusPending {
		t.Fatalf("[登记阶段] 新凭证应为 pending，实际 %s", env.statusOf(successID))
	}
}

// 用例 2：两名物业并发审核同一张凭证，只能成功一次（失败发生在审核阶段）。
func TestFlow_ConcurrentApproveSamePass(t *testing.T) {
	env := newFlowEnv(t)
	building := "并发审核栋"
	p := env.reg(building, env.uniquePhone(), false)

	var mu sync.Mutex
	var success, fail int
	runConcurrent(8, func(int) {
		_, err := env.svc.Approve(p.ID, env.staff.ID, "")
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if !errors.Is(err, ErrPassState) && !errors.Is(err, ErrStateLost) {
				t.Errorf("[审核阶段] 期望状态冲突，实际: %v", err)
			}
			fail++
			return
		}
		success++
	})
	if success != 1 || fail != 7 {
		t.Fatalf("[审核阶段] 同凭证并发审核应 1 成功 7 失败，实际成功=%d 失败=%d", success, fail)
	}
	if env.statusOf(p.ID) != constants.PassStatusApproved {
		t.Fatalf("[审核阶段] 终态应为 approved，实际 %s", env.statusOf(p.ID))
	}
	if n := env.eventCount(p.ID, constants.PassActionApprove); n != 1 {
		t.Fatalf("[审核阶段] 审核留痕应为 1 条，实际 %d（出现重复留痕）", n)
	}
	cv := env.capacity(building)
	if cv.committed != 1 || cv.remaining != cv.limit-1 {
		t.Fatalf("[审核阶段] 已承诺应=1、剩余应=%d，实际 committed=%d remaining=%d", cv.limit-1, cv.committed, cv.remaining)
	}
}

// 用例 3：同一凭证并发办理进入，只能成功一次（失败发生在进入阶段）。
func TestFlow_ConcurrentCheckIn(t *testing.T) {
	env := newFlowEnv(t)
	building := "并发进入栋"
	p := env.approveMust(env.reg(building, env.uniquePhone(), false).ID)

	var mu sync.Mutex
	var success, fail int
	runConcurrent(8, func(int) {
		_, err := env.svc.CheckIn(p.ID, env.staff.ID, "东门")
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if !errors.Is(err, ErrGateAlreadyIn) && !errors.Is(err, ErrStateLost) {
				t.Errorf("[进入阶段] 期望重复进入冲突，实际: %v", err)
			}
			fail++
			return
		}
		success++
	})
	if success != 1 || fail != 7 {
		t.Fatalf("[进入阶段] 并发进入应 1 成功 7 失败，实际成功=%d 失败=%d", success, fail)
	}
	if env.statusOf(p.ID) != constants.PassStatusCheckedIn {
		t.Fatalf("[进入阶段] 终态应为 checked_in，实际 %s", env.statusOf(p.ID))
	}
	if n := env.eventCount(p.ID, constants.PassActionCheckIn); n != 1 {
		t.Fatalf("[进入阶段] 进入留痕应为 1 条，实际 %d（重复办理产生重复记录）", n)
	}
	cv := env.capacity(building)
	if cv.onsite != 1 {
		t.Fatalf("[进入阶段] 在场数应为 1，实际 %d", cv.onsite)
	}
	// 再补一次串行的重复进入，仍应失败、不新增留痕。
	if _, err := env.svc.CheckIn(p.ID, env.staff.ID, "东门"); !errors.Is(err, ErrGateAlreadyIn) {
		t.Fatalf("[进入阶段] 串行重复进入应报 ErrGateAlreadyIn，实际 %v", err)
	}
	if n := env.eventCount(p.ID, constants.PassActionCheckIn); n != 1 {
		t.Fatalf("[进入阶段] 重复进入后留痕仍应为 1，实际 %d", n)
	}
}

// 用例 4：同一凭证并发办理离开，只能成功一次并恢复容量（失败发生在离开阶段）。
func TestFlow_ConcurrentCheckOut(t *testing.T) {
	env := newFlowEnv(t)
	building := "并发离开栋"
	pid := env.checkInMust(env.approveMust(env.reg(building, env.uniquePhone(), false).ID).ID).ID
	if env.capacity(building).onsite != 1 {
		t.Fatalf("[离开阶段] 前置在场数应为 1")
	}

	var mu sync.Mutex
	var success, fail int
	runConcurrent(8, func(int) {
		_, err := env.svc.CheckOut(pid, env.staff.ID, "东门")
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			if !errors.Is(err, ErrGateNotIn) && !errors.Is(err, ErrStateLost) {
				t.Errorf("[离开阶段] 期望重复离开冲突，实际: %v", err)
			}
			fail++
			return
		}
		success++
	})
	if success != 1 || fail != 7 {
		t.Fatalf("[离开阶段] 并发离开应 1 成功 7 失败，实际成功=%d 失败=%d", success, fail)
	}
	if env.statusOf(pid) != constants.PassStatusCompleted {
		t.Fatalf("[离开阶段] 终态应为 completed，实际 %s", env.statusOf(pid))
	}
	if n := env.eventCount(pid, constants.PassActionCheckOut); n != 1 {
		t.Fatalf("[离开阶段] 离开留痕应为 1 条，实际 %d", n)
	}
	cv := env.capacity(building)
	if cv.onsite != 0 || cv.committed != 0 || cv.remaining != cv.limit {
		t.Fatalf("[离开阶段] 离场后在场=0、已承诺=0、剩余应恢复为上限 %d，实际 onsite=%d committed=%d remaining=%d",
			cv.limit, cv.onsite, cv.committed, cv.remaining)
	}
	// 终态不能被重复离开改写。
	if _, err := env.svc.CheckOut(pid, env.staff.ID, "东门"); !errors.Is(err, ErrGateNotIn) {
		t.Fatalf("[离开阶段] 对已完成凭证再次离开应失败，实际 %v", err)
	}
}

// 用例 5：容量已满时，审核被明确拒绝（失败发生在审核阶段），不写成功留痕。
func TestFlow_CapacityFullBlocksApprove(t *testing.T) {
	env := newFlowEnv(t)
	building := "满员栋"
	if _, err := env.svc.SetCapacity(building, 1, env.staff.ID); err != nil {
		t.Fatal(err)
	}
	a := env.reg(building, env.uniquePhone(), false)
	env.approveMust(a.ID)
	b := env.reg(building, env.uniquePhone(), false)
	c := env.reg(building, env.uniquePhone(), false)

	for _, id := range []uint{b.ID, c.ID} {
		if _, err := env.svc.Approve(id, env.staff.ID, ""); !errors.Is(err, ErrCapacityReached) {
			t.Fatalf("[审核阶段] 容量已满应拒绝凭证 %d（ErrCapacityReached），实际 %v", id, err)
		}
		if env.statusOf(id) != constants.PassStatusPending {
			t.Fatalf("[审核阶段] 被容量拒绝后凭证 %d 应保持 pending，实际 %s", id, env.statusOf(id))
		}
		if n := env.eventCount(id, constants.PassActionApprove); n != 0 {
			t.Fatalf("[审核阶段] 被容量拒绝不应留下审核成功留痕，实际 %d 条", n)
		}
	}
	cv := env.capacity(building)
	if cv.committed != 1 || cv.remaining != 0 {
		t.Fatalf("[审核阶段] 已承诺应=1、剩余应=0，实际 committed=%d remaining=%d", cv.committed, cv.remaining)
	}
}

// 用例 6：取消已占名额的凭证会释放名额，随后待审凭证可通过（取消阶段）。
func TestFlow_CancelReleasesCapacity(t *testing.T) {
	env := newFlowEnv(t)
	building := "取消释放栋"
	if _, err := env.svc.SetCapacity(building, 1, env.staff.ID); err != nil {
		t.Fatal(err)
	}
	a := env.approveMust(env.reg(building, env.uniquePhone(), false).ID)
	b := env.reg(building, env.uniquePhone(), false)
	if env.capacity(building).remaining != 0 {
		t.Fatalf("[取消阶段] 前置剩余应为 0")
	}
	if _, err := env.svc.Approve(b.ID, env.staff.ID, ""); !errors.Is(err, ErrCapacityReached) {
		t.Fatalf("[取消阶段] 释放前 B 应被容量拒绝，实际 %v", err)
	}
	// 业主取消本人凭证，名额释放。
	if _, err := env.svc.Cancel(a.ID, env.resident.ID, "resident"); err != nil {
		t.Fatalf("[取消阶段] 业主取消本人凭证失败: %v", err)
	}
	if env.statusOf(a.ID) != constants.PassStatusCancelled {
		t.Fatalf("[取消阶段] A 应为 cancelled，实际 %s", env.statusOf(a.ID))
	}
	if n := env.eventCount(a.ID, constants.PassActionCancel); n != 1 {
		t.Fatalf("[取消阶段] 取消留痕应为 1，实际 %d", n)
	}
	cv := env.capacity(building)
	if cv.committed != 0 || cv.remaining != 1 {
		t.Fatalf("[取消阶段] 取消后已承诺=0、剩余=1，实际 committed=%d remaining=%d", cv.committed, cv.remaining)
	}
	// 释放后 B 可审核通过。
	env.approveMust(b.ID)
	if env.statusOf(b.ID) != constants.PassStatusApproved || env.eventCount(b.ID, constants.PassActionApprove) != 1 {
		t.Fatalf("[取消阶段] B 释放后应审核通过且有 1 条留痕")
	}
}

// 用例 7：容量页剩余数量必须与"还能审核通过的数量"一致。
func TestFlow_RemainingEqualsApprovable(t *testing.T) {
	env := newFlowEnv(t)
	building := "一致校验栋"
	limit := 3
	if _, err := env.svc.SetCapacity(building, limit, env.staff.ID); err != nil {
		t.Fatal(err)
	}
	// 通过两张（其中一张进入在场，一张待进入），已承诺=2，剩余=1。
	a := env.approveMust(env.reg(building, env.uniquePhone(), false).ID)
	env.approveMust(env.reg(building, env.uniquePhone(), false).ID)
	env.checkInMust(a.ID)

	cv := env.capacity(building)
	if cv.onsite != 1 || cv.committed != 2 || cv.remaining != 1 {
		t.Fatalf("[审核阶段] 期望 onsite=1 committed=2 remaining=1，实际 onsite=%d committed=%d remaining=%d",
			cv.onsite, cv.committed, cv.remaining)
	}
	// 准备 3 张待审，实测在剩余名额耗尽前能通过几张——必须恰好等于页面 remaining。
	pending := make([]uint, 0, 3)
	for i := 0; i < 3; i++ {
		pending = append(pending, env.reg(building, env.uniquePhone(), false).ID)
	}
	approved := 0
	for _, id := range pending {
		if _, err := env.svc.Approve(id, env.staff.ID, ""); err == nil {
			approved++
		} else if !errors.Is(err, ErrCapacityReached) {
			t.Fatalf("[审核阶段] 超额审核应返回容量错误，实际 %v", err)
		}
	}
	if approved != cv.remaining {
		t.Fatalf("[审核阶段] 实际可审核数=%d 与容量页剩余=%d 不一致", approved, cv.remaining)
	}
	if final := env.capacity(building); final.committed != int64(limit) || final.remaining != 0 {
		t.Fatalf("[审核阶段] 满额后 committed 应=%d remaining=0，实际 committed=%d remaining=%d",
			limit, final.committed, final.remaining)
	}
}

// 用例 8：浏览器本地到访时段保存后，门岗核对与列表/详情读到的是同一时刻。
func TestFlow_BrowserLocalTimeConsistency(t *testing.T) {
	env := newFlowEnv(t)
	building := "时区一致栋"
	// 模拟中文浏览器 datetime-local 提交的钟面串（无时区，T 分隔），按社区时区解释。
	startStr := util.CommunityNow().Add(-30 * time.Minute).Format("2006-01-02T15:04")
	endStr := util.CommunityNow().Add(2 * time.Hour).Format("2006-01-02T15:04")
	start, err := ParseVisitTime(startStr)
	if err != nil {
		t.Fatalf("[登记阶段] 解析开始时间失败: %v", err)
	}
	end, err := ParseVisitTime(endStr)
	if err != nil {
		t.Fatalf("[登记阶段] 解析结束时间失败: %v", err)
	}
	p := env.approveMust(env.register(building, env.uniquePhone(), start, end).ID)

	// 列表与详情回读的绝对时刻必须与提交一致。
	list, err := env.svc.List(0, "staff", "", building)
	if err != nil || len(list) != 1 || !list[0].StartTime.Equal(start) || !list[0].EndTime.Equal(end) {
		t.Fatalf("[列表展示] 列表时段与提交时刻不一致: n=%d err=%v", len(list), err)
	}
	detail, events, err := env.svc.Detail(p.ID, env.staff.ID, "staff")
	if err != nil || !detail.StartTime.Equal(start) || !detail.EndTime.Equal(end) {
		t.Fatalf("[详情展示] 详情时段与提交时刻不一致: %v", err)
	}
	// 渲染钟面必须等于业主提交的钟面（跨时区/跨部署都不偏移），业主/物业/门岗看到同一时刻。
	wantWall := func(s string) string { return strings.Replace(s, "T", " ", 1) }
	if got := util.FormatVisit(detail.StartTime); got != wantWall(startStr) {
		t.Fatalf("[详情展示] 钟面偏移: 提交=%s 渲染=%s", wantWall(startStr), got)
	}
	if len(events) < 2 {
		t.Fatalf("[留痕回读] 至少应有登记+审核两条，实际 %d", len(events))
	}
	// 门岗核对基于同一存储时刻，当前在时段内应放行，且渲染钟面一致。
	verify, info, err := env.svc.GateVerify(p.PassNo, env.staff.ID)
	if err != nil || info["allow"] != true || !verify.StartTime.Equal(start) {
		t.Fatalf("[门岗核对] 时段内应 allow=true 且时刻一致: allow=%v err=%v", info["allow"], err)
	}
	if util.FormatVisit(verify.StartTime) != wantWall(startStr) || util.FormatVisit(verify.EndTime) != wantWall(endStr) {
		t.Fatalf("[门岗核对] 门岗看到的钟面与业主提交不一致")
	}

	// 未到时段的凭证：门岗必须拒绝且不改变状态（失败发生在进入阶段之前的核对环节）。
	fs := util.CommunityNow().Add(2 * time.Hour)
	futureStart, futureEnd := fs, fs.Add(2*time.Hour)
	fp := env.approveMust(env.register(building, env.uniquePhone(), futureStart, futureEnd).ID)
	if _, info, err = env.svc.GateVerify(fp.PassNo, env.staff.ID); err != nil || info["allow"] != false || info["reason"] != constants.MessageGateNotInWindow {
		t.Fatalf("[门岗核对] 未到时段应 allow=false 且原因正确: info=%v err=%v", info, err)
	}
	if _, err = env.svc.CheckIn(fp.ID, env.staff.ID, "东门"); !errors.Is(err, ErrGateNotInWindow) {
		t.Fatalf("[进入阶段] 未到时段进入应报 ErrGateNotInWindow，实际 %v", err)
	}
	if env.statusOf(fp.ID) != constants.PassStatusApproved {
		t.Fatalf("[进入阶段] 未到时段被拒后凭证应保持 approved，实际 %s", env.statusOf(fp.ID))
	}
}

// 用例 9：跨日（跨零点）到访时段的解析、存储与回读一致。
func TestFlow_CrossDayOvernightWindow(t *testing.T) {
	env := newFlowEnv(t)
	building := "跨日时段栋"
	loc := util.CommunityLocation()
	now := util.CommunityNow()
	// 取社区时区下未来最近的一个 23:00 作为开始，结束为次日 01:00，保证跨零点。
	startDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, loc)
	if !startDay.After(now) {
		startDay = startDay.AddDate(0, 0, 1)
	}
	overnightEnd := startDay.Add(2 * time.Hour)
	if overnightEnd.Day() == startDay.Day() {
		t.Fatalf("[跨日校验] 构造的时段未跨零点: %s ~ %s", startDay, overnightEnd)
	}
	// 从 datetime-local 钟面串往返解析，验证跨日字符串被按社区时区正确还原。
	startWall := startDay.Format("2006-01-02T15:04")
	endWall := overnightEnd.Format("2006-01-02T15:04")
	s, e1 := ParseVisitTime(startWall)
	en, e2 := ParseVisitTime(endWall)
	if e1 != nil || e2 != nil || !s.Equal(startDay) || !en.Equal(overnightEnd) {
		t.Fatalf("[跨日校验] 跨日时段解析不一致: %v %v", e1, e2)
	}
	p := env.register(building, env.uniquePhone(), s, en)
	detail, _, err := env.svc.Detail(p.ID, env.staff.ID, "staff")
	if err != nil || !detail.StartTime.Equal(s) || !detail.EndTime.Equal(en) {
		t.Fatalf("[跨日校验] 回读的跨日时段与提交不一致: %v", err)
	}
	if detail.EndTime.Sub(detail.StartTime) != 2*time.Hour {
		t.Fatalf("[跨日校验] 跨日时长应=2h，实际 %s", detail.EndTime.Sub(detail.StartTime))
	}
	// 渲染钟面必须与提交钟面逐日逐分一致（跨日不偏移、不错到前一天/后一天）。
	if util.FormatVisit(detail.StartTime) != strings.Replace(startWall, "T", " ", 1) ||
		util.FormatVisit(detail.EndTime) != strings.Replace(endWall, "T", " ", 1) {
		t.Fatalf("[跨日校验] 钟面偏移: 提交 %s~%s 渲染 %s~%s",
			startWall, endWall, util.FormatVisit(detail.StartTime), util.FormatVisit(detail.EndTime))
	}
}

// 用例 10：跨日且已过离场时间未离场 → 逾期标记并恢复容量（失败/兜底发生在逾期阶段）。
func TestFlow_CrossDayExpiryRestoresCapacity(t *testing.T) {
	env := newFlowEnv(t)
	building := "跨日逾期栋"
	if _, err := env.svc.SetCapacity(building, 1, env.staff.ID); err != nil {
		t.Fatal(err)
	}
	// 构造确定已过去的跨零点时段：社区时区昨天 23:00 ~ 今天 01:00。
	loc := util.CommunityLocation()
	now := util.CommunityNow()
	expiredStart := time.Date(now.Year(), now.Month(), now.Day()-1, 23, 0, 0, 0, loc)
	expiredEnd := expiredStart.Add(2 * time.Hour)
	if !expiredEnd.Before(now) {
		t.Fatalf("[逾期阶段] 测试前置时段必须已过期")
	}
	p := env.register(building, env.uniquePhone(), expiredStart, expiredEnd)
	// 经真实持久化把已审核凭证置为在场（模拟访客已进入但跨日未离场）。
	if err := env.db.Model(&model.VisitorPass{}).Where("id = ?", p.ID).
		Updates(map[string]any{"status": constants.PassStatusCheckedIn, "check_in_at": expiredStart.UTC()}).Error; err != nil {
		t.Fatalf("[逾期阶段] 前置写入在场状态失败: %v", err)
	}
	before := env.capacity(building)
	if before.onsite != 1 || before.committed != 1 || before.remaining != 0 {
		t.Fatalf("[逾期阶段] 前置应 onsite=1 committed=1 remaining=0，实际 %+v", before)
	}

	marked, err := env.svc.SweepExpired()
	if err != nil || marked != 1 {
		t.Fatalf("[逾期阶段] 逾期扫描应标记 1 张，实际 marked=%d err=%v", marked, err)
	}
	if st := env.statusOf(p.ID); st != constants.PassStatusExpired {
		t.Fatalf("[逾期阶段] 终态应为 expired，实际 %s", st)
	}
	if n := env.eventCount(p.ID, constants.PassActionExpire); n != 1 {
		t.Fatalf("[逾期阶段] 逾期留痕应为 1 条，实际 %d", n)
	}
	after := env.capacity(building)
	if after.onsite != 0 || after.committed != 0 || after.remaining != after.limit {
		t.Fatalf("[逾期阶段] 逾期后应恢复为 onsite=0 committed=0 remaining=上限，实际 %+v", after)
	}
	// 逾期凭证门岗核对必须拒绝放行。
	full, err := env.svc.passes.ByID(p.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, info, err := env.svc.GateVerify(full.PassNo, env.staff.ID); err != nil || info["allow"] != false || info["reason"] != constants.MessageGateExpired {
		t.Fatalf("[门岗核对] 逾期凭证应拒绝并提示过期: info=%v err=%v", info, err)
	}
}
