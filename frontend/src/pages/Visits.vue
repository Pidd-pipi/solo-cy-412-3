<template>
  <section>
    <header class="page-head">
      <div>
        <p class="eyebrow">VISITOR PASS</p>
        <h2>访客通行</h2>
        <p>为到访亲友登记，凭证仅在所选时段内有效。</p>
      </div>
      <el-button type="primary" @click="openCreate">登记访客</el-button>
    </header>

    <div class="toolbar">
      <el-select v-model="status" placeholder="全部状态" clearable @change="load" style="width: 160px">
        <el-option v-for="f in passStatusFilters" :key="f.value" :label="f.label" :value="f.value" />
      </el-select>
    </div>

    <div class="pass-list">
      <PassCard v-for="p in items" :key="p.id" :pass="p">
        <template #actions="{ pass }">
          <el-button
            v-if="pass.status === 'pending' || pass.status === 'approved'"
            size="small" type="danger" plain
            @click="onCancel(pass.id)"
          >取消凭证</el-button>
          <el-button size="small" @click="showTimeline(pass.id)">留痕记录</el-button>
        </template>
      </PassCard>
      <EmptyState v-if="!items.length" description="还没有登记访客凭证" />
    </div>

    <!-- 登记弹窗 -->
    <el-dialog v-model="dialog" title="登记访客凭证" width="480px">
      <el-form :model="form" label-width="84px">
        <el-form-item label="访客姓名"><el-input v-model="form.visitor_name" placeholder="真实姓名" /></el-form-item>
        <el-form-item label="手机号"><el-input v-model="form.visitor_phone" maxlength="11" placeholder="11 位手机号" /></el-form-item>
        <el-form-item label="来访楼栋">
          <el-input v-model="form.building" placeholder="如 1栋；默认本人楼栋">
            <template #append>
              <el-button @click="form.building = myBuilding">用我的楼栋</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="到访开始"><el-date-picker v-model="form.start_time" type="datetime" value-format="YYYY-MM-DDTHH:mm" placeholder="选择开始时间" /></el-form-item>
        <el-form-item label="到访结束"><el-date-picker v-model="form.end_time" type="datetime" value-format="YYYY-MM-DDTHH:mm" placeholder="选择结束时间" /></el-form-item>
        <el-form-item label="来访事由"><el-input v-model="form.reason" type="textarea" placeholder="如 探亲、送货、维修" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">提交登记</el-button>
      </template>
    </el-dialog>

    <!-- 留痕时间线 -->
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
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listPasses, createPass, cancelPass, passDetail } from '../api/visitor'
import { passStatusFilters, passActionText } from '../constants/visitor'
import type { VisitorPass, VisitorEvent } from '../types'
import { authStore } from '../stores/authStore'
import PassCard from '../components/common/PassCard.vue'
import EmptyState from '../components/common/EmptyState.vue'
import { formatVisit, toLocalInput } from '../utils/visitTime'

const items = ref<VisitorPass[]>([])
const status = ref('')
const dialog = ref(false)
const saving = ref(false)
const myBuilding = computed(() => authStore.user?.building || '1栋')
const emptyForm = () => {
  const start = new Date(Date.now() + 60 * 60 * 1000)
  const end = new Date(Date.now() + 3 * 60 * 60 * 1000)
  return {
    visitor_name: '', visitor_phone: '', building: myBuilding.value,
    start_time: toLocalInput(start), end_time: toLocalInput(end), reason: '',
  }
}
const form = ref(emptyForm())

const timelineDialog = ref(false)
const events = ref<VisitorEvent[]>([])

async function load() {
  items.value = await listPasses(status.value)
}
function openCreate() {
  form.value = emptyForm()
  dialog.value = true
}
async function submit() {
  if (!/^\d{11}$/.test(form.value.visitor_phone)) {
    ElMessage.warning('请输入 11 位手机号')
    return
  }
  if (new Date(form.value.end_time) <= new Date(form.value.start_time)) {
    ElMessage.warning('结束时间必须晚于开始时间')
    return
  }
  saving.value = true
  try {
    await createPass(form.value)
    ElMessage.success('访客通行凭证已登记，等待物业审核')
    dialog.value = false
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}
async function onCancel(id: number) {
  try {
    await ElMessageBox.confirm('取消后该凭证立即失效，确定取消吗？', '取消凭证', { type: 'warning' })
    await cancelPass(id)
    ElMessage.success('凭证已取消')
    load()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error((e as Error).message)
  }
}
async function showTimeline(id: number) {
  const data = await passDetail(id)
  events.value = data.events
  timelineDialog.value = true
}
onMounted(load)
</script>
