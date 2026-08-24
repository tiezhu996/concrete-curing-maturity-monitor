import { create } from 'zustand'
import { strengthForecastApi } from '../api/strength-forecast-api'
import type { StrengthForecast } from '../types/strength-forecast'
import type { QueryParams } from '../types/common'

interface StrengthForecastState {
  items: StrengthForecast[]
  total: number
  loading: boolean
  fetch: (params?: QueryParams) => Promise<void>
}

export const useStrengthForecastStore = create<StrengthForecastState>((set) => ({
  items: [], total: 0, loading: false,
  fetch: async (params) => {
    set({ loading: true })
    try {
      const data = await strengthForecastApi.list({ page: 1, page_size: 100, ...params })
      set({ items: data.items, total: data.total })
    } finally {
      set({ loading: false })
    }
  },
}))
