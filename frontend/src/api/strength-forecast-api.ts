import type { PageData, QueryParams } from '../types/common'
import type { StrengthForecast } from '../types/strength-forecast'
import { apiClient } from './client'

export const strengthForecastApi = {
  list: (params?: QueryParams) => apiClient.get<PageData<StrengthForecast>>('/strength-forecasts', params),
  get: (id: number) => apiClient.get<StrengthForecast>(`/strength-forecasts/${id}`),
  run: (pourSectionId: number, temperatureSeriesId: number, idempotencyKey: string) =>
    apiClient.post<StrengthForecast>(
      '/strength-forecasts',
      { pour_section_id: pourSectionId, temperature_series_id: temperatureSeriesId },
      { 'Idempotency-Key': idempotencyKey },
    ),
  action: (id: number, action: 'review' | 'confirm' | 'void', note: string) =>
    apiClient.post<StrengthForecast>(`/strength-forecasts/${id}/${action}`, { note }),
  replay: (id: number) => apiClient.post<StrengthForecast>(`/strength-forecasts/${id}/replay`),
}
