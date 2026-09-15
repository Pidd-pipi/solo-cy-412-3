<template>
  <section>
    <header class="page-head">
      <div>
        <p class="eyebrow">VISITOR REVIEW</p>
        <h2>凭证审核 · 楼栋容量</h2>
        <p>审核业主登记的访客凭证；楼栋在场达到当日上限时自动暂停新审核。</p>
      </div>
      <el-button type="primary" @click="loadAll">刷新</el-button>
    </header>

    <section class="capacity-block">
      <h3>楼栋当日容量</h3>
      <div class="cap-grid">
        <el-card v-for="c in capacities" :key="c.building" class="cap-card" shadow="never">
          <div class="cap-title">{{ c.building }}</div>
          <div class="cap-num" :class="{ full: c.daily_limit > 0 && c.remaining <= 0 }">
            剩余 {{ c.daily_limit === 0 ? '不限' : c.remaining }}
          </div>
          <div class="cap-sub">在场 {{ c.onsite }} / 上限 {{ c.daily_limit === 0 ? '不限' : c.daily_limit }}</div>
          <el-button size="small" link @click="editCap(c)">调整上限</el-button>
        </el-card>
        <el-card class="cap-card add" shadow="never" @click="addCap">
          <span class="plus">＋</span>
          <span>新增楼栋容量</span>
        </el-card>
      </div>
    </section>

    <div class="toolbar">
      <el-select v-model="status" placeholder="全部状态" clearable @change="load" style="width: 160px">
        <el-option v-for="f in passStatusFilters" :key="f.value" :label="f.label" :value="f.value" />
      </el-select>
      <el-input v-model="building" placeholder="按楼栋筛选" clearable style="width: 160px; margin-left: 8px" @change="load" />
    </div>

    <div class="pass-list">
      <PassCard v-for="p in items" :key="p.id" :pass="p">
        <template #actions="{ pass }">
          <template v-if="pass.status === 'pending'">
            <el-button size="small" type="success" @click="onApprove(pass.id)">通过</el-button>
            <el-button size="small" type="danger" plain @click="onReject(pass.id)">驳回</el-button>
          </template>
          <el-tag v-else-if="pass.status === 'checked_in'" type="primary" size="small">访客在场中</el-tag>
          <el-button size="small" @click="showTimeline(pass.id)">留痕</el-button>
        </template>
      </PassCard>
      <EmptyState v-if="!items.length" description="暂无凭证" />
    </div>

    <el-dialog v-model="timelineDialog" title="凭证留痕" width="520px">
      <el-timeline>
        <el-timeline-item v-for="e in events" :key="e.id" :timestamp="formatVisit(e.created_at)" placement="top">
          <b>{{ passActionText[e.action] || e.action }}</b>
          <div>{{ e.detail }}</div>
        </el-timeline-item>
      </el-timeline>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listPasses, listCapacity, updateCapacity, approvePass, rejectPass, passDetail,
} from '../api/visitor'
import { passStatusFilters, passActionText } from '../constants/visitor'
import type { VisitorPass, VisitorEvent, BuildingCapacity } from '../types'
import PassCard from '../components/common/PassCard.vue'
import EmptyState from '../components/common/EmptyState.vue'
import { formatVisit } from '../utils/visitTime'

const items = ref<VisitorPass[]>([])
const capacities = ref<BuildingCapacity[]>([])
const status = ref('pending')
const building = ref('')
const timelineDialog = ref(false)
const events = ref<VisitorEvent[]>([])

async function load() {
  items.value = await listPasses(status.value, building.value)
}
async function loadCaps() {
  capacities.value = await listCapacity()
}
async function loadAll() {
  await Promise.all([load(), loadCaps()])
}
async function onApprove(id: number) {
  try {
    const { value } = await ElMessageBox.prompt('审核备注（可选）', '通过凭证', {
      confirmButtonText: '通过', inputValue: '',
    }).catch(() => ({ value: undefined }))
    if (value === undefined) return
    await approvePass(id, value || '')
    ElMessage.success('凭证已审核通过')
    loadAll()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}
async function onReject(id: number) {
  try {
    const { value } = await ElMessageBox.prompt('请填写驳回原因', '驳回凭证', {
      confirmButtonText: '驳回', inputValidator: (v) => !!v?.trim() || '驳回原因必填',
    })
    await rejectPass(id, value)
    ElMessage.success('凭证已驳回')
    loadAll()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error((e as Error).message)
  }
}
async function editCap(c: BuildingCapacity) {
  const { value } = await ElMessageBox.prompt(`设置 ${c.building} 当日在场上限（0 表示不限）`, '调整容量', {
    inputValue: String(c.daily_limit), inputPattern: /^\d+$/, inputErrorMessage: '请输入非负整数',
  }).catch(() => ({ value: undefined }))
  if (value === undefined) return
  await updateCapacity(c.building, Number(value))
  ElMessage.success('楼栋当日容量已更新')
  loadCaps()
}
async function addCap() {
  const form = await ElMessageBox.prompt('输入楼栋名称与上限，格式：楼栋,上限（如 2栋,30）', '新增楼栋容量', {
    inputPattern: /^.+,\d+$/, inputErrorMessage: '格式不正确',
  }).catch(() => ({ value: undefined }))
  if (!form.value) return
  const [b, limit] = form.value.split(',')
  await updateCapacity(b.trim(), Number(limit))
  ElMessage.success('已新增楼栋容量')
  loadCaps()
}
async function showTimeline(id: number) {
  const data = await passDetail(id)
  events.value = data.events
  timelineDialog.value = true
}
onMounted(loadAll)
</script>
<style scoped>
.cap-grid { display: flex; gap: 12px; flex-wrap: wrap; }
.cap-card { width: 180px; text-align: center; }
.cap-card.add { display: flex; align-items: center; justify-content: center; gap: 6px; color: var(--el-text-color-secondary); cursor: pointer; border-style: dashed; }
.cap-card.add .plus { font-size: 22px; line-height: 1; }
.cap-title { color: var(--el-text-color-secondary); }
.cap-num { font-size: 24px; font-weight: 700; margin: 4px 0; }
.cap-num.full { color: var(--el-color-danger); }
.cap-sub { font-size: 12px; color: var(--el-text-color-secondary); }
</style>
