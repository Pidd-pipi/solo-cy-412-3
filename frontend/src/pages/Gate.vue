<template>
  <section>
    <header class="page-head">
      <div>
        <p class="eyebrow">GATE CONTROL</p>
        <h2>门岗核对</h2>
        <p>输入凭证号核对，仅时段内有效凭证可放行；重复进入无效。</p>
      </div>
    </header>

    <el-card shadow="never" class="gate-box">
      <div class="gate-input">
        <el-input
          v-model="passNo" placeholder="输入凭证号，如 V09151830A1B2C3" clearable
          size="large" @keyup.enter="verify"
        />
        <el-button type="primary" size="large" :loading="verifying" @click="verify">核对</el-button>
      </div>
      <el-input v-model="checkpoint" placeholder="门岗名称" size="small" style="max-width: 180px; margin-top: 10px" />

      <div v-if="result" class="verify-result">
        <el-result
          :icon="result.allow ? 'success' : 'warning'"
          :title="result.allow ? '凭证有效，可办理进入' : '不可放行'"
          :sub-title="result.reason || result.status_text"
        />
        <el-descriptions :column="2" border>
          <el-descriptions-item label="凭证号">{{ result.pass.pass_no }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <PassStatusBadge :status="result.pass.status" />
          </el-descriptions-item>
          <el-descriptions-item label="访客">{{ result.pass.visitor_name }} {{ result.pass.visitor_phone }}</el-descriptions-item>
          <el-descriptions-item label="到访楼栋">{{ result.pass.building }}</el-descriptions-item>
          <el-descriptions-item label="到访时段" :span="2">
            {{ formatVisit(result.pass.start_time) }} ~ {{ formatVisit(result.pass.end_time) }}
          </el-descriptions-item>
          <el-descriptions-item label="楼栋在场">{{ result.onsite }}</el-descriptions-item>
          <el-descriptions-item label="来访事由">{{ result.pass.reason }}</el-descriptions-item>
        </el-descriptions>
        <div class="gate-actions">
          <el-button
            v-if="result.allow" type="success" size="large" :loading="acting"
            @click="doCheckIn"
          >办理进入</el-button>
          <el-button
            v-if="result.pass.status === 'checked_in'" type="warning" size="large" :loading="acting"
            @click="doCheckOut"
          >办理离开</el-button>
        </div>
      </div>
    </el-card>

    <section class="records-block">
      <h3>门岗记录 <el-button link @click="loadRecords">刷新</el-button></h3>
      <el-table :data="records" size="small" stripe>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ formatVisit(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="动作" width="110">
          <template #default="{ row }">{{ passActionText[row.action] || row.action }}</template>
        </el-table-column>
        <el-table-column prop="detail" label="详情" />
        <el-table-column label="经办人" width="120">
          <template #default="{ row }">{{ row.actor?.nickname || '系统' }}</template>
        </el-table-column>
      </el-table>
    </section>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { verifyPass, gateCheckIn, gateCheckOut, gateRecords } from '../api/visitor'
import { passActionText } from '../constants/visitor'
import type { GateVerifyResult, VisitorEvent } from '../types'
import PassStatusBadge from '../components/common/PassStatusBadge.vue'
import { formatVisit } from '../utils/visitTime'

const passNo = ref('')
const checkpoint = ref('东门')
const verifying = ref(false)
const acting = ref(false)
const result = ref<GateVerifyResult | null>(null)
const records = ref<VisitorEvent[]>([])

async function verify() {
  if (!passNo.value.trim()) {
    ElMessage.warning('请输入凭证号')
    return
  }
  verifying.value = true
  try {
    result.value = await verifyPass(passNo.value.trim())
  } catch (e) {
    result.value = null
    ElMessage.error((e as Error).message)
  } finally {
    verifying.value = false
  }
}
async function afterAction() {
  await verify()
  await loadRecords()
}
async function doCheckIn() {
  if (!result.value) return
  acting.value = true
  try {
    await gateCheckIn(result.value.pass.id, checkpoint.value || '东门')
    ElMessage.success('已办理进入')
    await afterAction()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    acting.value = false
  }
}
async function doCheckOut() {
  if (!result.value) return
  acting.value = true
  try {
    await gateCheckOut(result.value.pass.id, checkpoint.value || '东门')
    ElMessage.success('已办理离开，楼栋容量已恢复')
    await afterAction()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    acting.value = false
  }
}
async function loadRecords() {
  records.value = await gateRecords(50)
}
onMounted(loadRecords)
</script>
<style scoped>
.gate-box { margin-bottom: 20px; }
.gate-input { display: flex; gap: 10px; }
.verify-result { margin-top: 16px; }
.gate-actions { text-align: center; margin-top: 8px; }
.records-block h3 { display: flex; align-items: center; gap: 8px; }
</style>
