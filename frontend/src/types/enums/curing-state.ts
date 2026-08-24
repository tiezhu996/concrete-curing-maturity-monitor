export const CURING_STATES = ['prepared', 'poured', 'curing', 'suspended', 'threshold_reached', 'closed'] as const
export type CuringState = (typeof CURING_STATES)[number]

export const CURING_STATE_LABELS: Record<CuringState, string> = {
  prepared: '待浇筑',
  poured: '已浇筑',
  curing: '养护中',
  suspended: '已暂停',
  threshold_reached: '达到阈值',
  closed: '已关闭',
}

export const CURING_TRANSITIONS: Record<CuringState, CuringState[]> = {
  prepared: ['poured', 'suspended'],
  poured: ['curing', 'suspended'],
  curing: ['threshold_reached', 'suspended'],
  threshold_reached: ['closed', 'suspended'],
  suspended: ['curing', 'closed'],
  closed: [],
}
