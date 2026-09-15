package constants

// VisitorPass 凭证状态机：
// pending（待审核）→ approved（已通过）→ checked_in（在场）→ completed（已离场）
// pending/approved → cancelled（已取消）；approved/checked_in → expired（已逾期，超时未离场由定时任务兜底）
const (
	PassStatusPending   = "pending"
	PassStatusApproved  = "approved"
	PassStatusCheckedIn = "checked_in"
	PassStatusCompleted = "completed"
	PassStatusCancelled = "cancelled"
	PassStatusRejected  = "rejected"
	PassStatusExpired   = "expired"
)

// ValidPassStatuses 供筛选与校验使用。
var ValidPassStatuses = map[string]bool{
	PassStatusPending:   true,
	PassStatusApproved:  true,
	PassStatusCheckedIn: true,
	PassStatusCompleted: true,
	PassStatusCancelled: true,
	PassStatusRejected:  true,
	PassStatusExpired:   true,
}

// ActivePassStatuses 仍占容量、可被门岗放行或会参与时段重叠判断的“有效凭证”状态。
var ActivePassStatuses = []string{PassStatusPending, PassStatusApproved, PassStatusCheckedIn}

// 访客凭证业务动作（用于访客事件留痕与操作日志 action）。
const (
	PassActionCreate   = "visitor.pass.create"
	PassActionApprove  = "visitor.pass.approve"
	PassActionReject   = "visitor.pass.reject"
	PassActionCancel   = "visitor.pass.cancel"
	PassActionCheckIn  = "visitor.pass.checkin"
	PassActionCheckOut = "visitor.pass.checkout"
	PassActionExpire   = "visitor.pass.expire"
	PassActionCapacity = "visitor.capacity.update"
	PassActionGateRead = "visitor.gate.read"
)

// DefaultBuildingDailyLimit 楼栋当日访客在场容量缺省上限。
const DefaultBuildingDailyLimit = 50
