export const CONFIDENCE_LEVELS = ['low', 'medium', 'high'] as const
export type ConfidenceLevel = (typeof CONFIDENCE_LEVELS)[number]

export const CONFIDENCE_LABELS: Record<ConfidenceLevel, string> = {
  low: '低置信',
  medium: '中置信',
  high: '高置信',
}
