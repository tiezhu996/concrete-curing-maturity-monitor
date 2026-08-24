export type MixDesignState = 'draft' | 'validated' | 'published' | 'retired'

export interface CalibrationPoint {
  maturity_degree_hours: number
  strength_mpa: number
}

export interface MixDesign {
  id: number
  mix_code: string
  version: number
  cement_type: string
  water_binder_ratio: number
  datum_temperature_c: number
  calibration_points: CalibrationPoint[]
  valid_from?: string
  valid_to?: string
  design_state: MixDesignState
  created_by: number
  created_by_name: string
  lock_version: number
  referenced_sections: number
  created_at: string
  updated_at: string
}

export interface CreateMixDesign {
  mix_code: string
  version: number
  cement_type: string
  water_binder_ratio: number
  datum_temperature_c: number
  calibration_points: CalibrationPoint[]
}
