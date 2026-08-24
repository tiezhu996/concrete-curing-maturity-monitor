import type { CuringState } from './enums/curing-state'

export interface PourSectionSummary {
  temperature_series_count: number
  forecast_count: number
  latest_strength_mpa: number
  latest_confidence: string
}

export interface PourSection {
  id: number
  section_code: string
  name: string
  structure_part: string
  volume_m3: number
  mix_design_id: number
  mix_code: string
  mix_version: number
  poured_at?: string
  target_strength_mpa: number
  curing_state: CuringState
  owner_team: string
  version: number
  summary: PourSectionSummary
  created_at: string
  updated_at: string
}

export interface CreatePourSection {
  section_code: string
  name: string
  structure_part: string
  volume_m3: number
  mix_design_id: number
  poured_at?: string
  target_strength_mpa: number
  owner_team: string
}
