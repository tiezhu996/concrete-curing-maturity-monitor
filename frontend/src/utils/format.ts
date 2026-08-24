export const formatDateTime = (value?: string): string => {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false,
  }).format(new Date(value))
}

export const formatNumber = (value: number | undefined, digits = 1): string =>
  value === undefined || Number.isNaN(value) ? '—' : value.toFixed(digits)

export const shortHash = (value?: string): string => value ? `${value.slice(0, 8)}…${value.slice(-6)}` : '—'
