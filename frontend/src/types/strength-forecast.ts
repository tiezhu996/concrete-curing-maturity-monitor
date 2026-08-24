import type { ConfidenceLevel } from './enums/confidence-level'

export type ForecastState = 'queued' | 'calculating' | 'completed' | 'failed' | 'reviewed' | 'confirmed' | 'voided'

export interface MaturityStep {
  index: number
  from: string
  to: string
  average_temperature_c: number
  duration_hours: number
  contribution_degree_hours: number
  running_maturity_degree_hours: number
  below_datum_clamped: boolean
}

export interface ForecastExplanation {
  formula_version: string
  formula: string
  below_datum_rule: string
  maturity: { degree_hours: number; duration_hours: number; steps: MaturityStep[] }
  interpolation: Record<string, unknown>
  data_coverage_percent: number
  quality_warnings: string[]
  boundary_note: string
  target_strength_mpa: number
  recent_maturity_rate_per_hour: number
}

export interface StrengthForecast {
  id: number
  pour_section_id: number
  section_code: string
  temperature_series_id: number
  sensor_code: string
  mix_design_version: number
  formula_version: string
  input_hash: string
  input_snapshot: Record<string, unknown>
  maturity_degree_hours: number
  predicted_strength_mpa: number
  threshold_eta?: string
  confidence_level: ConfidenceLevel
  forecast_state: ForecastState
  explanation: ForecastExplanation
  calculated_by: number
  calculated_by_name: string
  calculated_at: string
  reviewed_by?: number
  reviewed_by_name?: string
  confirmed_by?: number
  idempotency_key: string
  duration_milliseconds: number
  failure_reason?: string
  determinism_replay_passed?: boolean
  created_at: string
}
