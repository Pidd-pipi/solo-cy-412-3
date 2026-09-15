<template><el-tag :type="kind" effect="light">{{ passStatusText[status] }}</el-tag></template>
<script setup lang="ts">
import { computed } from 'vue'
import { passStatusText, passStatusKind } from '../../constants/visitor'
import type { PassStatus } from '../../types'
const p = defineProps<{ status: PassStatus }>()
const kind = computed(() => passStatusKind[p.status] as 'success')
</script>
