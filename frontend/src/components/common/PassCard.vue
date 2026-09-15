<template>
  <el-card class="pass-card" shadow="hover">
    <div class="pass-head">
      <div>
        <b>{{ pass.visitor_name }}</b>
        <span class="phone">{{ pass.visitor_phone }}</span>
      </div>
      <PassStatusBadge :status="pass.status" />
    </div>
    <div class="pass-no">凭证号：{{ pass.pass_no }}</div>
    <dl class="pass-grid">
      <div><dt>到访楼栋</dt><dd>{{ pass.building }}</dd></div>
      <div><dt>到访时段</dt><dd>{{ formatVisit(pass.start_time) }} ~ {{ formatVisit(pass.end_time) }}</dd></div>
      <div><dt>来访事由</dt><dd>{{ pass.reason }}</dd></div>
      <div v-if="pass.resident"><dt>登记业主</dt><dd>{{ pass.resident.nickname }}</dd></div>
      <div v-if="pass.check_in_at"><dt>进入时间</dt><dd>{{ formatVisit(pass.check_in_at) }}</dd></div>
      <div v-if="pass.check_out_at"><dt>离开时间</dt><dd>{{ formatVisit(pass.check_out_at) }}</dd></div>
      <div v-if="pass.review_remark"><dt>审核备注</dt><dd>{{ pass.review_remark }}</dd></div>
    </dl>
    <div v-if="$slots.actions" class="pass-actions">
      <slot name="actions" :pass="pass" />
    </div>
  </el-card>
</template>
<script setup lang="ts">
import type { VisitorPass } from '../../types'
import PassStatusBadge from './PassStatusBadge.vue'
import { formatVisit } from '../../utils/visitTime'
defineProps<{ pass: VisitorPass }>()
</script>
<style scoped>
.pass-head { display: flex; justify-content: space-between; align-items: center; }
.pass-head .phone { margin-left: 8px; color: var(--el-text-color-secondary); font-size: 13px; }
.pass-no { margin: 6px 0; color: var(--el-text-color-secondary); font-size: 12px; }
.pass-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 16px; margin: 0; }
.pass-grid dt { color: var(--el-text-color-secondary); font-size: 12px; }
.pass-grid dd { margin: 0; }
.pass-actions { margin-top: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
</style>
