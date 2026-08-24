import { create } from 'zustand'
import { temperatureSeriesApi } from '../api/temperature-series-api'
import type { TemperatureSeries } from '../types/temperature-series'
import type { QueryParams } from '../types/common'

interface TemperatureSeriesState {
  items: TemperatureSeries[]
  total: number
  loading: boolean
  fetch: (params?: QueryParams) => Promise<void>
}

export const useTemperatureSeriesStore = create<TemperatureSeriesState>((set) => ({
  items: [], total: 0, loading: false,
  fetch: async (params) => {
    set({ loading: true })
    try {
      const data = await temperatureSeriesApi.list({ page: 1, page_size: 100, ...params })
      set({ items: data.items, total: data.total })
    } finally {
      set({ loading: false })
    }
  },
}))
