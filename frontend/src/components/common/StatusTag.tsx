import { Tag } from 'antd'
import type { ConfidenceLevel } from '../../types/enums/confidence-level'
import { CONFIDENCE_LABELS } from '../../types/enums/confidence-level'

export function ConfidenceTag({ level }: { level: ConfidenceLevel }) {
  return <Tag color={{ low: 'volcano', medium: 'gold', high: 'green' }[level]}>{CONFIDENCE_LABELS[level]}</Tag>
}

export function SeriesStateTag({ state }: { state: 'imported' | 'usable' | 'invalid' }) {
  const labels = { imported: '待确认', usable: '可用于计算', invalid: '已作废' }
  return <Tag color={{ imported: 'gold', usable: 'green', invalid: 'volcano' }[state]}>{labels[state]}</Tag>
}

export function ForecastStateTag({ state }: { state: string }) {
  const labels: Record<string, string> = { queued: '排队中', calculating: '计算中', completed: '已完成', failed: '失败', reviewed: '已评审', confirmed: '已确认', voided: '已作废' }
  const color: Record<string, string> = { queued: 'default', calculating: 'blue', completed: 'cyan', failed: 'volcano', reviewed: 'gold', confirmed: 'green', voided: 'default' }
  return <Tag color={color[state] ?? 'default'}>{labels[state] ?? state}</Tag>
}
