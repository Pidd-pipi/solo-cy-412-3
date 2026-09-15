package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"github.com/smartestate/smartestate/internal/util"
	"gorm.io/gorm"
)

// 业务哨兵错误：handler 据此映射 HTTP 状态与统一错误码，错误信息层层透传。
var (
	ErrPassNotFound    = errors.New("visitor pass not found")
	ErrPassOverlap     = errors.New("overlapping valid pass exists")
	ErrPassState       = errors.New("visitor pass state invalid")
	ErrCapacityReached = errors.New("building daily capacity reached")
	ErrPassForbidden   = errors.New("visitor pass forbidden")
	ErrPassTimeInvalid = errors.New("visitor pass time range invalid")
	ErrGateNotInWindow = errors.New("visitor pass not in visit window")
	ErrGateAlreadyIn   = errors.New("visitor already checked in")
	ErrGateNotIn       = errors.New("visitor not checked in")
)

type VisitorService struct {
	db     *gorm.DB
	passes *repository.VisitorPassRepository
	events *repository.VisitorEventRepository
	caps   *repository.BuildingCapacityRepository
	logs   *OperationLogService
	logger *slog.Logger
}

func NewVisitorService(db *gorm.DB, p *repository.VisitorPassRepository, e *repository.VisitorEventRepository, c *repository.BuildingCapacityRepository, logs *OperationLogService, l *slog.Logger) *VisitorService {
	return &VisitorService{db: db, passes: p, events: e, caps: c, logs: logs, logger: l}
}

// withTx 在事务中执行，事件与凭证状态变更原子提交。
func (s *VisitorService) withTx(fn func(tx *gorm.DB) error) error {
	return s.db.Transaction(fn)
}

// recordEvent 在同一事务内写访客留痕与系统操作日志，保证取消/审核/进出/逾期必留痕且与状态变更同提交。
func (s *VisitorService) recordEvent(tx *gorm.DB, passID, actorID uint, action, from, to, detail string) error {
	if e := s.events.Create(&model.VisitorEvent{PassID: passID, ActorID: actorID, Action: action, FromStatus: from, ToStatus: to, Detail: detail}, tx); e != nil {
		s.logger.Error("write visitor event", "pass_id", passID, "action", action, "error", e)
		return e
	}
	if s.logs != nil {
		if e := s.logs.AddTx(actorID, action, detail, tx); e != nil {
			return e
		}
	}
	return nil
}

// ParseVisitTime 兼容 datetime-local（T 分隔）与 "2006-01-02 15:04"。
func ParseVisitTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	layouts := []string{"2006-01-02T15:04", "2006-01-02 15:04", time.RFC3339}
	var last error
	for _, l := range layouts {
		if t, e := time.ParseInLocation(l, v, time.Local); e == nil {
			return t, nil
		} else {
			last = e
		}
	}
	return time.Time{}, fmt.Errorf("%w: %v", ErrPassTimeInvalid, last)
}

func genPassNo() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("V%s%02X%02X%02X", time.Now().Format("01021504"), b[0], b[1], b[2])
}

// Create 业主登记访客凭证。
func (s *VisitorService) Create(residentID uint, name, phone, building, reason string, start, end time.Time) (model.VisitorPass, error) {
	if !end.After(start) {
		return model.VisitorPass{}, fmt.Errorf("VisitorPass[resident=%d] create failed: end before start, role=%s: %w", residentID, constants.UserRoleResident, ErrPassTimeInvalid)
	}
	pass := model.VisitorPass{
		PassNo:       genPassNo(),
		ResidentID:   residentID,
		VisitorName:  name,
		VisitorPhone: phone,
		Building:     building,
		Reason:       reason,
		StartTime:    start,
		EndTime:      end,
		Status:       constants.PassStatusPending,
	}
	e := s.withTx(func(tx *gorm.DB) error {
		n, e := s.passes.FindOverlap(phone, building, start, end, 0, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[phone=%s] overlap query failed: %w", phone, e)
		}
		if n > 0 {
			return fmt.Errorf("VisitorPass[phone=%s building=%s] create rejected: %w", phone, building, ErrPassOverlap)
		}
		if e = s.passes.Create(&pass, tx); e != nil {
			return fmt.Errorf("VisitorPass[resident=%d] create failed: %w", residentID, e)
		}
		if e = s.recordEvent(tx, pass.ID, residentID, constants.PassActionCreate, "", constants.PassStatusPending,
			fmt.Sprintf("登记凭证 %s：访客 %s(%s) 到访 %s %s~%s 事由 %s", pass.PassNo, name, phone, building, util.Date(start), util.Date(end), reason)); e != nil {
			return fmt.Errorf("VisitorPass[resident=%d] event failed: %w", residentID, e)
		}
		return nil
	})
	if e != nil {
		return model.VisitorPass{}, e
	}
	return s.passes.ByID(pass.ID, nil)
}

// List 业主仅见本人凭证；staff/admin 可按状态、楼栋筛选。
func (s *VisitorService) List(residentID uint, role, status, building string) ([]model.VisitorPass, error) {
	scopeID := uint(0)
	if role == constants.UserRoleResident {
		scopeID = residentID
	}
	return s.passes.List(scopeID, status, building)
}

func (s *VisitorService) Detail(id, uid uint, role string) (model.VisitorPass, []model.VisitorEvent, error) {
	v, e := s.passes.ByID(id, nil)
	if e != nil {
		return v, nil, fmt.Errorf("VisitorPass[id=%d] detail failed: %w", id, ErrPassNotFound)
	}
	if role == constants.UserRoleResident && v.ResidentID != uid {
		return v, nil, fmt.Errorf("VisitorPass[id=%d] detail forbidden: owner=%d user=%d: %w", id, v.ResidentID, uid, ErrPassForbidden)
	}
	events, e := s.events.ListByPass(id)
	if e != nil {
		return v, nil, fmt.Errorf("VisitorPass[id=%d] events failed: %w", id, e)
	}
	return v, events, nil
}

// Approve 物业审核通过；楼栋在场容量已达上限时暂停审核。
func (s *VisitorService) Approve(id, reviewerID uint, remark string) (model.VisitorPass, error) {
	e := s.withTx(func(tx *gorm.DB) error {
		v, e := s.passes.ByID(id, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[id=%d] approve failed: %w", id, ErrPassNotFound)
		}
		if v.Status != constants.PassStatusPending {
			return fmt.Errorf("VisitorPass[id=%d] approve rejected: status=%s: %w", id, v.Status, ErrPassState)
		}
		now := time.Now()
		if v.EndTime.Before(now) {
			return fmt.Errorf("VisitorPass[id=%d] approve rejected: window expired: %w", id, ErrPassState)
		}
		capCfg, e := s.caps.GetOrCreate(v.Building, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[id=%d] capacity query failed: %w", id, e)
		}
		onsite, e := s.passes.CountInBuilding(v.Building, now, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[id=%d] onsite count failed: %w", id, e)
		}
		if capCfg.DailyLimit > 0 && int(onsite) >= capCfg.DailyLimit {
			return fmt.Errorf("VisitorPass[id=%d building=%s] approve suspended: onsite=%d limit=%d: %w", id, v.Building, onsite, capCfg.DailyLimit, ErrCapacityReached)
		}
		v.Status = constants.PassStatusApproved
		v.ReviewerID = &reviewerID
		v.ReviewRemark = remark
		v.ReviewedAt = &now
		if e = s.passes.Update(&v, tx); e != nil {
			return fmt.Errorf("VisitorPass[id=%d] approve failed: %w", id, e)
		}
		if e = s.recordEvent(tx, id, reviewerID, constants.PassActionApprove, constants.PassStatusPending, constants.PassStatusApproved,
			fmt.Sprintf("凭证 %s 审核通过，楼栋 %s 当前在场 %d/%d", v.PassNo, v.Building, onsite, capCfg.DailyLimit)); e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return model.VisitorPass{}, e
	}
	return s.passes.ByID(id, nil)
}

// Reject 物业驳回待审核凭证。
func (s *VisitorService) Reject(id, reviewerID uint, remark string) (model.VisitorPass, error) {
	e := s.withTx(func(tx *gorm.DB) error {
		v, e := s.passes.ByID(id, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[id=%d] reject failed: %w", id, ErrPassNotFound)
		}
		if v.Status != constants.PassStatusPending {
			return fmt.Errorf("VisitorPass[id=%d] reject rejected: status=%s: %w", id, v.Status, ErrPassState)
		}
		now := time.Now()
		v.Status = constants.PassStatusRejected
		v.ReviewerID = &reviewerID
		v.ReviewRemark = remark
		v.ReviewedAt = &now
		if e = s.passes.Update(&v, tx); e != nil {
			return fmt.Errorf("VisitorPass[id=%d] reject failed: %w", id, e)
		}
		if e = s.recordEvent(tx, id, reviewerID, constants.PassActionReject, constants.PassStatusPending, constants.PassStatusRejected,
			fmt.Sprintf("凭证 %s 审核驳回：%s", v.PassNo, remark)); e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return model.VisitorPass{}, e
	}
	return s.passes.ByID(id, nil)
}

// Cancel 业主取消本人凭证，或物业取消任意凭证；仅待审核/已通过可取消。
func (s *VisitorService) Cancel(id, uid uint, role string) (model.VisitorPass, error) {
	e := s.withTx(func(tx *gorm.DB) error {
		v, e := s.passes.ByID(id, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[id=%d] cancel failed: %w", id, ErrPassNotFound)
		}
		if role == constants.UserRoleResident && v.ResidentID != uid {
			return fmt.Errorf("VisitorPass[id=%d] cancel forbidden: owner=%d user=%d: %w", id, v.ResidentID, uid, ErrPassForbidden)
		}
		if v.Status != constants.PassStatusPending && v.Status != constants.PassStatusApproved {
			return fmt.Errorf("VisitorPass[id=%d] cancel rejected: status=%s: %w", id, v.Status, ErrPassState)
		}
		from := v.Status
		v.Status = constants.PassStatusCancelled
		if e = s.passes.Update(&v, tx); e != nil {
			return fmt.Errorf("VisitorPass[id=%d] cancel failed: %w", id, e)
		}
		if e = s.recordEvent(tx, id, uid, constants.PassActionCancel, from, constants.PassStatusCancelled,
			fmt.Sprintf("凭证 %s 被 %s 取消", v.PassNo, util.RoleText(role))); e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return model.VisitorPass{}, e
	}
	return s.passes.ByID(id, nil)
}

// CheckIn 门岗办理进入：仅已通过且在到访时段内可放行；重复进入无效。
func (s *VisitorService) CheckIn(id, guardID uint, checkpoint string) (model.VisitorPass, error) {
	e := s.withTx(func(tx *gorm.DB) error {
		v, e := s.passes.ByID(id, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[id=%d] checkin failed: %w", id, ErrPassNotFound)
		}
		now := time.Now()
		switch {
		case v.Status == constants.PassStatusCheckedIn:
			return fmt.Errorf("VisitorPass[id=%s] checkin rejected: already in: %w", v.PassNo, ErrGateAlreadyIn)
		case v.Status == constants.PassStatusPending:
			return fmt.Errorf("VisitorPass[%s] checkin rejected: pending review: %w", v.PassNo, ErrPassState)
		case v.Status == constants.PassStatusCancelled:
			return fmt.Errorf("VisitorPass[%s] checkin rejected: cancelled: %w", v.PassNo, ErrPassState)
		case v.Status == constants.PassStatusCompleted:
			return fmt.Errorf("VisitorPass[%s] checkin rejected: completed: %w", v.PassNo, ErrPassState)
		case v.Status == constants.PassStatusRejected:
			return fmt.Errorf("VisitorPass[%s] checkin rejected: rejected: %w", v.PassNo, ErrPassState)
		case v.Status == constants.PassStatusExpired:
			return fmt.Errorf("VisitorPass[%s] checkin rejected: expired: %w", v.PassNo, ErrPassState)
		case v.Status != constants.PassStatusApproved:
			return fmt.Errorf("VisitorPass[%s] checkin rejected: status=%s: %w", v.PassNo, v.Status, ErrPassState)
		}
		if now.Before(v.StartTime) {
			return fmt.Errorf("VisitorPass[%s] checkin rejected: before window %s: %w", v.PassNo, util.Date(v.StartTime), ErrGateNotInWindow)
		}
		if now.After(v.EndTime) {
			return fmt.Errorf("VisitorPass[%s] checkin rejected: past window %s: %w", v.PassNo, util.Date(v.EndTime), ErrPassState)
		}
		// 进入瞬间再次确认在场容量，避免审核后并发超限。
		capCfg, e := s.caps.GetOrCreate(v.Building, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[%s] capacity query failed: %w", v.PassNo, e)
		}
		onsite, e := s.passes.CountInBuilding(v.Building, now, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[%s] onsite count failed: %w", v.PassNo, e)
		}
		if capCfg.DailyLimit > 0 && int(onsite) >= capCfg.DailyLimit {
			return fmt.Errorf("VisitorPass[%s building=%s] checkin suspended: onsite=%d limit=%d: %w", v.PassNo, v.Building, onsite, capCfg.DailyLimit, ErrCapacityReached)
		}
		v.Status = constants.PassStatusCheckedIn
		v.CheckInAt = &now
		v.Checkpoint = checkpoint
		if e = s.passes.Update(&v, tx); e != nil {
			return fmt.Errorf("VisitorPass[%s] checkin failed: %w", v.PassNo, e)
		}
		if e = s.recordEvent(tx, id, guardID, constants.PassActionCheckIn, constants.PassStatusApproved, constants.PassStatusCheckedIn,
			fmt.Sprintf("门岗 %s 放行进入，楼栋 %s 在场 %d/%d", checkpoint, v.Building, onsite+1, capCfg.DailyLimit)); e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return model.VisitorPass{}, e
	}
	return s.passes.ByID(id, nil)
}

// CheckOut 门岗办理离开；离开后实时容量恢复（在场数统计即减一）。
func (s *VisitorService) CheckOut(id, guardID uint, checkpoint string) (model.VisitorPass, error) {
	e := s.withTx(func(tx *gorm.DB) error {
		v, e := s.passes.ByID(id, tx)
		if e != nil {
			return fmt.Errorf("VisitorPass[id=%d] checkout failed: %w", id, ErrPassNotFound)
		}
		if v.Status != constants.PassStatusCheckedIn {
			return fmt.Errorf("VisitorPass[%s] checkout rejected: status=%s: %w", v.PassNo, v.Status, ErrGateNotIn)
		}
		now := time.Now()
		v.Status = constants.PassStatusCompleted
		v.CheckOutAt = &now
		if checkpoint != "" {
			v.Checkpoint = checkpoint
		}
		if e = s.passes.Update(&v, tx); e != nil {
			return fmt.Errorf("VisitorPass[%s] checkout failed: %w", v.PassNo, e)
		}
		if e = s.recordEvent(tx, id, guardID, constants.PassActionCheckOut, constants.PassStatusCheckedIn, constants.PassStatusCompleted,
			fmt.Sprintf("门岗 %s 办理离开 %s，楼栋 %s 容量已恢复", checkpoint, v.PassNo, v.Building)); e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return model.VisitorPass{}, e
	}
	return s.passes.ByID(id, nil)
}

// GateVerify 门岗核对凭证：返回凭证与是否可放行及原因，不改变状态；核对动作写操作日志。
func (s *VisitorService) GateVerify(passNo string, guardID uint) (model.VisitorPass, map[string]any, error) {
	v, e := s.passes.ByPassNo(strings.TrimSpace(passNo), nil)
	if e != nil {
		return v, nil, fmt.Errorf("VisitorPass[no=%s] verify failed: %w", passNo, ErrPassNotFound)
	}
	if s.logs != nil {
		s.logs.Add(guardID, constants.PassActionGateRead, fmt.Sprintf("门岗核对凭证 %s（%s）", v.PassNo, util.PassStatusText(v.Status)))
	}
	now := time.Now()
	allow := false
	reason := ""
	switch v.Status {
	case constants.PassStatusApproved:
		switch {
		case now.Before(v.StartTime):
			reason = constants.MessageGateNotInWindow
		case now.After(v.EndTime):
			reason = constants.MessageGateExpired
		default:
			allow = true
		}
	case constants.PassStatusCheckedIn:
		reason = constants.MessageGateAlreadyIn
	case constants.PassStatusPending:
		reason = constants.MessageGatePending
	case constants.PassStatusCancelled:
		reason = constants.MessageGateCancelled
	case constants.PassStatusCompleted:
		reason = constants.MessageGateCompleted
	case constants.PassStatusRejected:
		reason = constants.MessageGateRejected
	case constants.PassStatusExpired:
		reason = constants.MessageGateExpired
	}
	onsite, _ := s.passes.CountInBuilding(v.Building, now, nil)
	info := map[string]any{
		"allow":        allow,
		"reason":       reason,
		"status_text":  util.PassStatusText(v.Status),
		"onsite":       onsite,
		"current_time": now,
	}
	return v, info, nil
}

// SweepExpired 定时兜底：已过结束时段仍未离场的在场/已通过凭证标记逾期并恢复容量。
func (s *VisitorService) SweepExpired() (int, error) {
	due, e := s.passes.ExpiredDue(time.Now())
	if e != nil {
		return 0, fmt.Errorf("VisitorPass sweep query failed: %w", e)
	}
	marked := 0
	for _, p := range due {
		id := p.ID
		if e = s.withTx(func(tx *gorm.DB) error {
			v, e := s.passes.ByID(id, tx)
			if e != nil {
				return e
			}
			now := time.Now()
			from := v.Status
			v.Status = constants.PassStatusExpired
			v.ExpireMarkedAt = &now
			if e = s.passes.Update(&v, tx); e != nil {
				return e
			}
			if e = s.recordEvent(tx, id, 0, constants.PassActionExpire, from, constants.PassStatusExpired,
				fmt.Sprintf("凭证 %s 超过离场时间 %s 未离场，系统标记逾期，容量恢复", v.PassNo, util.Date(v.EndTime))); e != nil {
				return e
			}
			return nil
		}); e != nil {
			s.logger.Error("sweep visitor pass", "pass_id", id, "error", e)
			continue
		}
		marked++
	}
	return marked, nil
}

// CapacityOverview 物业工作台：各楼栋容量上限、当前在场、剩余数量。
func (s *VisitorService) CapacityOverview() ([]map[string]any, error) {
	cfgs, e := s.caps.List()
	if e != nil {
		return nil, e
	}
	now := time.Now()
	out := make([]map[string]any, 0, len(cfgs))
	seen := map[string]bool{}
	for _, c := range cfgs {
		onsite, e := s.passes.CountInBuilding(c.Building, now, nil)
		if e != nil {
			return nil, e
		}
		remaining := c.DailyLimit - int(onsite)
		if c.DailyLimit == 0 {
			remaining = -1 // 0 表示不限
		}
		out = append(out, map[string]any{"building": c.Building, "daily_limit": c.DailyLimit, "onsite": onsite, "remaining": remaining})
		seen[c.Building] = true
	}
	return out, nil
}

// SetCapacity 物业调整楼栋当日容量上限。
func (s *VisitorService) SetCapacity(building string, limit int, operatorID uint) (model.BuildingCapacity, error) {
	c, e := s.caps.Upsert(building, limit, operatorID)
	if e != nil {
		return c, fmt.Errorf("BuildingCapacity[%s] upsert failed: %w", building, e)
	}
	if s.logs != nil {
		s.logs.Add(operatorID, constants.PassActionCapacity, fmt.Sprintf("楼栋 %s 当日容量上限调整为 %d", building, limit))
	}
	return c, nil
}

// PendingCount 工作台待审核凭证数。
func (s *VisitorService) PendingCount() (int64, error) { return s.passes.CountPending() }

// RecentEvents 门岗记录 / 工作台最近进出留痕。
func (s *VisitorService) RecentEvents(limit int) ([]model.VisitorEvent, error) {
	return s.events.ListRecent(limit)
}
