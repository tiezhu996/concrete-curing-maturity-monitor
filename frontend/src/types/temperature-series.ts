export type TemperatureSeriesState = 'imported' | 'usable' | 'invalid'

export interface TemperaturePoint {
  timestamp: string
  temperature_c: number
}

export interface TemperatureSeries {
  id: number
  pour_section_id: number
  section_code: string
  sensor_code: string
  sample_interval_min: number
  points: TemperaturePoint[]
  started_at: string
  ended_at: string
  source_checksum: string
  missing_ratio: number
  series_state: TemperatureSeriesState
  quality_note: string
  imported_by: number
  imported_by_name: string
  created_at: string
  updated_at: string
}

export interface ImportTemperatureSeries {
  pour_section_id: number
  sensor_code: string
  sample_interval_min: number
  points: TemperaturePoint[]
  quality_note?: string
}
