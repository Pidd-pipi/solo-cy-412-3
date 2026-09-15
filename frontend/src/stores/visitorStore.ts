import { reactive } from 'vue'
import type { VisitorPass, BuildingCapacity, VisitorEvent } from '../types'

// 与 repairStore/paymentStore 一致的轻量 reactive 存储：
// 跨页面共享凭证列表、楼栋容量与门岗最近记录。
export const visitorStore = reactive<{
  passes: VisitorPass[]
  capacities: BuildingCapacity[]
  records: VisitorEvent[]
  pendingCount: number
}>({
  passes: [],
  capacities: [],
  records: [],
  pendingCount: 0,
})
