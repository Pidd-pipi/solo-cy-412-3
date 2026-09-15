package constants

const (
	MessageOK             = "ok"
	MessageUnauthorized   = "登录已失效"
	MessageForbidden      = "无权限执行此操作"
	MessageValidation     = "请求参数不合法"
	MessageNotFound       = "资源不存在"
	MessagePaymentSuccess = "支付宝沙箱支付成功"
	MessageRepairCreated  = "报修工单已提交"
)

// 访客通行模块业务提示文案（业主端 / 门岗 / 物业工作台共用，改动会同时影响返回与日志）。
const (
	MessagePassCreated       = "访客通行凭证已登记，等待物业审核"
	MessagePassApproved      = "凭证已审核通过"
	MessagePassRejected      = "凭证已驳回"
	MessagePassCancelled     = "凭证已取消"
	MessagePassCheckedIn     = "已办理进入"
	MessagePassCheckedOut    = "已办理离开，楼栋容量已恢复"
	MessagePassExpired       = "凭证已逾期，容量已恢复"
	MessageCapacityUpdated   = "楼栋当日容量已更新"
	MessageGateNotInWindow   = "未到到访时段，暂不能放行"
	MessageGateExpired       = "凭证已过期，不能放行"
	MessageGateCancelled     = "凭证已取消，不能放行"
	MessageGateCompleted     = "凭证已完成，不能重复放行"
	MessageGateRejected      = "凭证已被驳回，不能放行"
	MessageGatePending       = "凭证尚未审核通过，不能放行"
	MessageGateAlreadyIn     = "已办理过进入，请勿重复办理"
	MessageGateNotIn         = "访客尚未进入，无需办理离开"
	MessageOverlapConflict   = "同一访客在该楼栋已有时段重叠的有效凭证"
	MessageCapacityReached   = "楼栋当日剩余容量不足，已暂停新凭证审核"
	MessagePassNotCancelable = "当前状态的凭证不能取消"
	MessagePassNotReviewable = "仅待审核凭证可审核"
)
