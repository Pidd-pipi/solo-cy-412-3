import { request } from '../utils/request'
import type {
  VisitorPass,
  VisitorEvent,
  BuildingCapacity,
  GateVerifyResult,
} from '../types'

export interface CreatePassPayload {
  visitor_name: string
  visitor_phone: string
  building: string
  start_time: string
  end_time: string
  reason: string
}

const qs = (params: Record<string, string>) =>
  Object.entries(params)
    .filter(([, v]) => v)
    .map(([k, v]) => `${k}=${encodeURIComponent(v)}`)
    .join('&')

// 业主端 / 物业共用：凭证列表（业主后端自动仅返回本人凭证）。
export const listPasses = (status = '', building = '') =>
  request<VisitorPass[]>(`/visitor/passes?${qs({ status, building })}`)

export const createPass = (data: CreatePassPayload) =>
  request<VisitorPass>('/visitor/passes', { method: 'POST', body: JSON.stringify(data) })

export const passDetail = (id: number) =>
  request<{ pass: VisitorPass; events: VisitorEvent[] }>(`/visitor/passes/${id}`)

export const cancelPass = (id: number) =>
  request<VisitorPass>(`/visitor/passes/${id}/cancel`, { method: 'POST' })

// 物业审核。
export const approvePass = (id: number, remark = '') =>
  request<VisitorPass>(`/visitor/passes/${id}/approve`, { method: 'POST', body: JSON.stringify({ remark }) })

export const rejectPass = (id: number, remark: string) =>
  request<VisitorPass>(`/visitor/passes/${id}/reject`, { method: 'POST', body: JSON.stringify({ remark }) })

// 楼栋当日容量。
export const listCapacity = () => request<BuildingCapacity[]>('/visitor/capacity')
export const updateCapacity = (building: string, daily_limit: number) =>
  request<BuildingCapacity>('/visitor/capacity', {
    method: 'PUT',
    body: JSON.stringify({ building, daily_limit }),
  })

// 门岗。
export const verifyPass = (pass_no: string) =>
  request<GateVerifyResult>(`/gate/verify?pass_no=${encodeURIComponent(pass_no)}`)

export const gateCheckIn = (id: number, checkpoint: string) =>
  request<VisitorPass>(`/gate/passes/${id}/checkin`, {
    method: 'POST',
    body: JSON.stringify({ checkpoint }),
  })

export const gateCheckOut = (id: number, checkpoint: string) =>
  request<VisitorPass>(`/gate/passes/${id}/checkout`, {
    method: 'POST',
    body: JSON.stringify({ checkpoint }),
  })

export const gateRecords = (limit = 50) =>
  request<VisitorEvent[]>(`/gate/records?limit=${limit}`)
