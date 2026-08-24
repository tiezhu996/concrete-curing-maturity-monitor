import { Tag } from 'antd'
import type { CuringState } from '../../types/enums/curing-state'
import { CURING_STATE_LABELS } from '../../types/enums/curing-state'

const colors: Record<CuringState, string> = {
  prepared: 'default', poured: 'geekblue', curing: 'green', suspended: 'orange', threshold_reached: 'gold', closed: 'volcano',
}

export function CuringStateBadge({ state }: { state: CuringState }) {
  return <Tag color={colors[state]} className="state-tag">{CURING_STATE_LABELS[state]}</Tag>
}
