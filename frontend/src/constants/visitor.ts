import type { PassStatus } from '../types'

// 与后端 constants/visitor.go 的 PassStatus 一一对应。
export const PASS_STATUS = {
  PENDING: 'pending',
  APPROVED: 'approved',
  CHECKED_IN: 'checked_in',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
  REJECTED: 'rejected',
  EXPIRED: 'expired',
} as const

export const passStatusText: Record<PassStatus, string> = {
  pending: '待审核',
  approved: '已通过',
  checked_in: '在场',
  completed: '已完成',
  cancelled: '已取消',
  rejected: '已驳回',
  expired: '已逾期',
}

// el-tag 类型映射，被 PassStatusBadge 与凭证列表共用。
export const passStatusKind: Record<PassStatus, string> = {
  pending: 'warning',
  approved: 'success',
  checked_in: 'primary',
  completed: 'info',
  cancelled: 'danger',
  rejected: 'danger',
  expired: 'danger',
}

// 凭证事件动作中文文案（门岗记录 / 凭证时间线）。
export const passActionText: Record<string, string> = {
  'visitor.pass.create': '登记凭证',
  'visitor.pass.approve': '审核通过',
  'visitor.pass.reject': '审核驳回',
  'visitor.pass.cancel': '取消凭证',
  'visitor.pass.checkin': '办理进入',
  'visitor.pass.checkout': '办理离开',
  'visitor.pass.expire': '逾时未离场',
  'visitor.capacity.update': '容量调整',
  'visitor.gate.read': '门岗核对',
}

// 业主端状态筛选（全部 + 主要状态）。
export const passStatusFilters: { value: PassStatus; label: string }[] = [
  { value: 'pending', label: '待审核' },
  { value: 'approved', label: '已通过' },
  { value: 'checked_in', label: '在场' },
  { value: 'completed', label: '已完成' },
  { value: 'cancelled', label: '已取消' },
  { value: 'rejected', label: '已驳回' },
  { value: 'expired', label: '已逾期' },
]
