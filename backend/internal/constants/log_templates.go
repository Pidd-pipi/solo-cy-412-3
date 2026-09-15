package constants

var LogTemplates = []string{
	"user.login", "user.profile.update", "user.property.bind", "repair.create", "repair.assign", "repair.status.pending", "repair.status.assigned", "repair.status.processing", "repair.status.done", "repair.status.closed", "repair.comment.create", "payment.bill.create", "payment.pay", "payment.history.read", "announcement.create", "announcement.pin", "announcement.read", "announcement.list", "dashboard.read", "operation_log.read", "role.assign", "permission.grant", "auth.reject", "rate_limit.reject", "error.response", "request.complete", "seed.bootstrap",
	// 访客通行模块：取消、审核、进入、离开、逾期、容量调整与门岗核对均需留痕。
	"visitor.pass.create", "visitor.pass.approve", "visitor.pass.reject", "visitor.pass.cancel", "visitor.pass.checkin", "visitor.pass.checkout", "visitor.pass.expire", "visitor.capacity.update", "visitor.gate.read", "visitor.list", "visitor.capacity.read",
}
