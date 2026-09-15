// 访客时段格式化。
// 后端统一按社区时区把到访时刻输出为钟面串 "YYYY-MM-DD HH:mm"（不含时区、不做浏览器时区换算），
// 因此这里原样显示，保证业主、物业、门岗在任意时区的浏览器上看到同一本地时刻；
// 若收到旧的 ISO/RFC3339 串则回退按本地解析展示。
export function formatVisit(value?: string): string {
  if (!value) return '—'
  const wall = /^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}/
  if (wall.test(value)) {
    return value.replace('T', ' ').slice(0, 16)
  }
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// datetime-local 输入框值（本地时区，分钟精度）。
export function toLocalInput(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}
