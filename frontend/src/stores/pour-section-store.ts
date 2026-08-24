import { create } from 'zustand'
import { pourSectionApi } from '../api/pour-section-api'
import type { PourSection } from '../types/pour-section'
import type { QueryParams } from '../types/common'

interface PourSectionState {
  items: PourSection[]
  total: number
  loading: boolean
  fetch: (params?: QueryParams) => Promise<void>
}

export const usePourSectionStore = create<PourSectionState>((set) => ({
  items: [], total: 0, loading: false,
  fetch: async (params) => {
    set({ loading: true })
    try {
      const data = await pourSectionApi.list({ page: 1, page_size: 100, ...params })
      set({ items: data.items, total: data.total })
    } finally {
      set({ loading: false })
    }
  },
}))
