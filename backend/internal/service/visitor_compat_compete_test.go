package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"github.com/smartestate/smartestate/internal/util"
	"gorm.io/gorm"
)

// 本文件覆盖两个主题：
//  A. 存量兼容：用上一版（进程/社区时区 +08:00）已经落库的本地钟面记录做升级前样本，
//     跑升级归一化后，列表、详情、门岗核对仍按原钟面展示；新记录的本地时段也不变。
//  B. 容量竞争：不同凭证并发争抢最后一个审核名额，只有一张通过，容量页剩余与审批结果一致，
//     失败请求不留成功留痕。
// 全程使用磁盘 SQLite + WAL + busy_timeout + 多连接的真实持久化与事务链路，
// 不用内存替身、不串行化请求。

// insertLegacyPass 按"上一版"的真实落库格式写入一条访客记录：
// 旧 GORM SQLite 驱动把 time.Time 存成带 +08:00 偏移的字符串（非新版的 UTC 规范形式）。
func insertLegacyPass(db *gorm.DB, pass model.VisitorPass, startWall, endWall time.Time, loc *time.Location) uint {
	startStr := startWall.In(loc).Format("2006-01-02 15:04:05-07:00")
	endStr := endWall.In(loc).Format("2006-01-02 15:04:05-07:00")
	nowStr := time.Now().In(loc).Format("2006-01-02 15:04:05-07:00")
	if e := db.Exec(`INSERT INTO visitor_passes
		(pass_no, resident_id, visitor_name, visitor_phone, building, reason, start_time, end_time, status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		pass.PassNo, pass.ResidentID, pass.VisitorName, pass.VisitorPhone, pass.Building, pass.Reason,
		startStr, endStr, pass.Status, nowStr, nowStr).Error; e != nil {
		panic(e)
	}
	var got model.VisitorPass
	db.Where("pass_no = ?", pass.PassNo).First(&got)
	return got.ID
}

// 主题 A：存量本地钟面记录升级后按原值展示，新记录时段不变。
func TestLegacy_LocalWallSurvivesUpgrade(t *testing.T) {
	env := newFlowEnv(t)
	loc := util.CommunityLocation()
	building := "存量兼容栋"
	if _, err := env.svc.SetCapacity(building, 10, env.staff.ID); err != nil {
		t.Fatalf("[存量读取] 容量预置失败: %v", err)
	}

	// 旧样本 1：跨零点、日期固定的已完成凭证。无论今天几号，升级后都必须逐分按原值展示。
	legStart := time.Date(2026, 1, 15, 23, 30, 0, 0, loc)
	legEnd := time.Date(2026, 1, 16, 1, 0, 0, 0, loc)
	idDone := insertLegacyPass(env.db, model.VisitorPass{
		PassNo: "V-LEGACY-DONE", ResidentID: env.resident.ID, VisitorName: "存量已离场",
		VisitorPhone: "13700009001", Building: building, Reason: "存量跨日记录", Status: constants.PassStatusCompleted,
	}, legStart, legEnd, loc)

	// 旧样本 2：当前时段内、已审核通过的凭证（用于门岗核对）。
	actStart := util.CommunityNow().Add(-30 * time.Minute)
	actEnd := util.CommunityNow().Add(2 * time.Hour)
	idActive := insertLegacyPass(env.db, model.VisitorPass{
		PassNo: "V-LEGACY-ACT", ResidentID: env.resident.ID, VisitorName: "存量在场",
		VisitorPhone: "13700009002", Building: building, Reason: "存量当前记录", Status: constants.PassStatusApproved,
	}, actStart, actEnd, loc)

	// 执行与服务启动相同的一次性、幂等升级归一化。
	if _, err := repository.NormalizeVisitorTimes(env.db); err != nil {
		t.Fatalf("[存量读取] 升级归一化失败: %v", err)
	}
	// 幂等：再跑一次不应改变任何结果。
	if _, err := repository.NormalizeVisitorTimes(env.db); err != nil {
		t.Fatalf("[存量读取] 归一化重复执行失败: %v", err)
	}

	wantWall := func(x time.Time) string { return x.In(loc).Format("2006-01-02 15:04") }

	// —— 回读阶段：固定跨日旧记录的钟面必须与原值逐日逐分一致 ——
	done, err := env.svc.passes.ByID(idDone, nil)
	if err != nil {
		t.Fatalf("[回读阶段] 读取旧已完成凭证失败: %v", err)
	}
	if util.FormatVisit(done.StartTime) != wantWall(legStart) || util.FormatVisit(done.EndTime) != wantWall(legEnd) {
		t.Fatalf("[回读阶段] 旧跨日记录钟面被改写: 期望 %s~%s 实际 %s~%s",
			wantWall(legStart), wantWall(legEnd), util.FormatVisit(done.StartTime), util.FormatVisit(done.EndTime))
	}
	if done.Status != constants.PassStatusCompleted {
		t.Fatalf("[回读阶段] 旧终态不应改变，实际 %s", done.Status)
	}

	// —— 存量读取阶段：列表按 building 过滤能取到两条旧记录且钟面不变 ——
	list, err := env.svc.List(0, "staff", "", building)
	if err != nil {
		t.Fatalf("[存量读取] 列表读取失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("[存量读取] 应读到 2 条存量记录，实际 %d", len(list))
	}
	for _, p := range list {
		switch p.ID {
		case idDone:
			if util.FormatVisit(p.StartTime) != wantWall(legStart) || util.FormatVisit(p.EndTime) != wantWall(legEnd) {
				t.Fatalf("[存量读取] 列表中旧跨日钟面偏移: %s~%s", util.FormatVisit(p.StartTime), util.FormatVisit(p.EndTime))
			}
		case idActive:
			if util.FormatVisit(p.StartTime) != wantWall(actStart) || util.FormatVisit(p.EndTime) != wantWall(actEnd) {
				t.Fatalf("[存量读取] 列表中当前旧记录钟面偏移: %s~%s", util.FormatVisit(p.StartTime), util.FormatVisit(p.EndTime))
			}
		}
	}

	// —— 详情阶段：详情读取的绝对时刻与旧样本一致 ——
	det, _, err := env.svc.Detail(idDone, env.staff.ID, "staff")
	if err != nil || !det.StartTime.Equal(legStart) || !det.EndTime.Equal(legEnd) {
		t.Fatalf("[详情展示] 详情时刻与原值绝对时刻不一致: %v", err)
	}

	// —— 门岗核对阶段：旧的当前时段凭证应可放行，且门岗看到同一钟面 ——
	act, info, err := env.svc.GateVerify("V-LEGACY-ACT", env.staff.ID)
	if err != nil {
		t.Fatalf("[门岗核对] 旧凭证核对失败: %v", err)
	}
	if info["allow"] != true {
		t.Fatalf("[门岗核对] 当前时段的旧凭证应放行，实际 allow=false reason=%v", info["reason"])
	}
	if util.FormatVisit(act.StartTime) != wantWall(actStart) || util.FormatVisit(act.EndTime) != wantWall(actEnd) {
		t.Fatalf("[门岗核对] 门岗钟面与业主原值不一致: %s~%s", util.FormatVisit(act.StartTime), util.FormatVisit(act.EndTime))
	}

	// —— 新记录：升级后新登记的本地时段落库、回读、门岗展示也不能变化 ——
	newStart := util.CommunityNow().Add(time.Hour)
	newEnd := util.CommunityNow().Add(3 * time.Hour)
	np := env.register(building, env.uniquePhone(), newStart, newEnd)
	nDetail, _, err := env.svc.Detail(np.ID, env.staff.ID, "staff")
	if err != nil || !nDetail.StartTime.Equal(newStart) || !nDetail.EndTime.Equal(newEnd) {
		t.Fatalf("[回读阶段] 新记录绝对时刻被改变: %v", err)
	}
	if util.FormatVisit(nDetail.StartTime) != wantWall(newStart) || util.FormatVisit(nDetail.EndTime) != wantWall(newEnd) {
		t.Fatalf("[回读阶段] 新记录本地钟面偏移: 期望 %s~%s 实际 %s~%s",
			wantWall(newStart), wantWall(newEnd), util.FormatVisit(nDetail.StartTime), util.FormatVisit(nDetail.EndTime))
	}
}

// 主题 B：不同凭证并发争抢最后一个审核名额。
func TestCapacity_ContendForFinalSlot(t *testing.T) {
	env := newFlowEnv(t)
	building := "末位竞争栋"
	limit := 3
	if _, err := env.svc.SetCapacity(building, limit, env.staff.ID); err != nil {
		t.Fatalf("[审核阶段] 容量预置失败: %v", err)
	}

	// 先用 K-1 个不同访客占掉名额（一张在场、一张已通过待进入），使剩余恰好 1。
	pre1 := env.checkInMust(env.approveMust(env.reg(building, env.uniquePhone(), false).ID).ID)
	pre2 := env.approveMust(env.reg(building, env.uniquePhone(), false).ID)
	if pre1.Status != constants.PassStatusCheckedIn {
		t.Fatalf("[审核阶段] 前置在场状态错误: %s", pre1.Status)
	}
	if pre2.Status != constants.PassStatusApproved {
		t.Fatalf("[审核阶段] 前置待进入状态错误: %s", pre2.Status)
	}
	cv := env.capacity(building)
	if cv.committed != int64(limit-1) || cv.remaining != 1 {
		t.Fatalf("[回读阶段] 竞争前应 committed=%d remaining=1，实际 committed=%d remaining=%d",
			limit-1, cv.committed, cv.remaining)
	}

	// 多张不同凭证（不同访客，互不重叠）同时争抢最后一个名额。
	const contenders = 6
	pendingIDs := make([]uint, 0, contenders)
	for i := 0; i < contenders; i++ {
		pendingIDs = append(pendingIDs, env.reg(building, env.uniquePhone(), false).ID)
	}

	var mu sync.Mutex
	var success, rejected int
	runConcurrent(contenders, func(i int) {
		_, err := env.svc.Approve(pendingIDs[i], env.staff.ID, "")
		mu.Lock()
		defer mu.Unlock()
		if err == nil {
			success++
			return
		}
		if !errors.Is(err, ErrCapacityReached) {
			t.Errorf("[审核阶段] 末位竞争的失败应是容量错误，凭证 %d 实际: %v", pendingIDs[i], err)
		}
		rejected++
	})

	if success != 1 || rejected != contenders-1 {
		t.Fatalf("[审核阶段] 最后一个名额应恰好 1 张通过、%d 张容量拒绝，实际通过=%d 拒绝=%d",
			contenders-1, success, rejected)
	}

	// —— 回读阶段：容量页剩余必须与审批结果一致（满额 remaining=0，committed=上限）——
	final := env.capacity(building)
	if final.committed != int64(limit) || final.remaining != 0 {
		t.Fatalf("[回读阶段] 竞争后应 committed=%d remaining=0，实际 committed=%d remaining=%d",
			limit, final.committed, final.remaining)
	}
	// 页面剩余=0 时，再审核任一仍待审的输家都必须失败（页面数字与"还能审核数"一致）。
	var stillPending uint
	for _, id := range pendingIDs {
		if env.statusOf(id) == constants.PassStatusPending {
			stillPending = id
			break
		}
	}
	if stillPending == 0 {
		t.Fatalf("[回读阶段] 应有待审输家可供复验")
	}
	if _, err := env.svc.Approve(stillPending, env.staff.ID, ""); !errors.Is(err, ErrCapacityReached) {
		t.Fatalf("[审核阶段] 容量页剩余为 0 时不应再通过，实际 %v", err)
	}

	// —— 留痕回读：唯一成功者有 1 条审核留痕；所有失败者不得留下成功留痕、状态仍 pending ——
	var winner int
	for _, id := range pendingIDs {
		n := env.eventCount(id, constants.PassActionApprove)
		st := env.statusOf(id)
		if st == constants.PassStatusApproved {
			winner++
			if n != 1 {
				t.Fatalf("[回读阶段] 成功凭证应有 1 条审核留痕，实际 %d", n)
			}
			continue
		}
		if st != constants.PassStatusPending {
			t.Fatalf("[回读阶段] 竞争失败凭证应保持 pending，凭证 %d 实际 %s", id, st)
		}
		if n != 0 {
			t.Fatalf("[回读阶段] 容量拒绝的凭证不得留下审核成功留痕，凭证 %d 实际 %d 条", id, n)
		}
	}
	if winner != 1 {
		t.Fatalf("[回读阶段] 落库的通过凭证应恰好 1 张，实际 %d", winner)
	}

	// —— 进入阶段：赢家可进入；仍是 pending 的输家门岗进入必须失败，不改变容量 ——
	var winID uint
	for _, id := range pendingIDs {
		if env.statusOf(id) == constants.PassStatusApproved {
			winID = id
		}
	}
	if _, err := env.svc.CheckIn(winID, env.staff.ID, "东门"); err != nil {
		t.Fatalf("[进入阶段] 抢到名额的凭证应可进入，实际 %v", err)
	}
	loserID := uint(0)
	for _, id := range pendingIDs {
		if env.statusOf(id) == constants.PassStatusPending {
			loserID = id
			break
		}
	}
	if _, err := env.svc.CheckIn(loserID, env.staff.ID, "东门"); !errors.Is(err, ErrPassState) {
		t.Fatalf("[进入阶段] 未通过凭证进入应失败，实际 %v", err)
	}
	post := env.capacity(building)
	if post.onsite != 2 {
		t.Fatalf("[进入阶段] 赢家进入后在场应为 2（前置 1 + 赢家 1），实际 %d", post.onsite)
	}
}
