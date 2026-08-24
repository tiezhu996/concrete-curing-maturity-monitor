import { create } from 'zustand'
import { mixDesignApi } from '../api/mix-design-api'
import type { MixDesign } from '../types/mix-design'
import type { QueryParams } from '../types/common'

interface MixDesignState {
  items: MixDesign[]
  total: number
  loading: boolean
  fetch: (params?: QueryParams) => Promise<void>
}

export const useMixDesignStore = create<MixDesignState>((set) => ({
  items: [], total: 0, loading: false,
  fetch: async (params) => {
    set({ loading: true })
    try {
      const data = await mixDesignApi.list({ page: 1, page_size: 100, ...params })
      set({ items: data.items, total: data.total })
    } finally {
      set({ loading: false })
    }
  },
}))
