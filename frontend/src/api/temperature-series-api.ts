import type { PageData, QueryParams } from '../types/common'
import type { ImportTemperatureSeries, TemperatureSeries } from '../types/temperature-series'
import { apiClient } from './client'

export const temperatureSeriesApi = {
  list: (params?: QueryParams) => apiClient.get<PageData<TemperatureSeries>>('/temperature-series', params),
  get: (id: number) => apiClient.get<TemperatureSeries>(`/temperature-series/${id}`),
  import: (payload: ImportTemperatureSeries) => apiClient.post<TemperatureSeries>('/temperature-series', payload),
  confirm: (id: number, reason: string) => apiClient.post<TemperatureSeries>(`/temperature-series/${id}/confirm`, { reason }),
  invalidate: (id: number, reason: string) => apiClient.post<TemperatureSeries>(`/temperature-series/${id}/invalidate`, { reason }),
}
