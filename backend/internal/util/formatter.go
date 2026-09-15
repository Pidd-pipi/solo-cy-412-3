package util

import (
	"fmt"
	"github.com/smartestate/smartestate/internal/constants"
	"time"
)

func Date(v time.Time) string { return v.Format("2006-01-02 15:04") }
func Money(v float64) string  { return fmt.Sprintf("¥%.2f", v) }
func StatusText(v string) string {
	m := map[string]string{constants.RepairStatusPending: "待受理", constants.RepairStatusAssigned: "已分派", constants.RepairStatusProcessing: "处理中", constants.RepairStatusDone: "已完成", constants.RepairStatusClosed: "已关闭"}
	return m[v]
}
func PassStatusText(v string) string {
	m := map[string]string{
		constants.PassStatusPending:   "待审核",
		constants.PassStatusApproved:  "已通过",
		constants.PassStatusCheckedIn: "在场",
		constants.PassStatusCompleted: "已完成",
		constants.PassStatusCancelled: "已取消",
		constants.PassStatusRejected:  "已驳回",
		constants.PassStatusExpired:   "已逾期",
	}
	return m[v]
}
func PassActionText(v string) string {
	m := map[string]string{
		constants.PassActionCreate:   "登记凭证",
		constants.PassActionApprove:  "审核通过",
		constants.PassActionReject:   "审核驳回",
		constants.PassActionCancel:   "取消凭证",
		constants.PassActionCheckIn:  "办理进入",
		constants.PassActionCheckOut: "办理离开",
		constants.PassActionExpire:   "逾时标记",
		constants.PassActionCapacity: "容量调整",
		constants.PassActionGateRead: "门岗核对",
	}
	return m[v]
}
func RoleText(v string) string {
	m := map[string]string{constants.UserRoleResident: "业主", constants.UserRoleStaff: "物业人员", constants.UserRoleAdmin: "管理员"}
	return m[v]
}
