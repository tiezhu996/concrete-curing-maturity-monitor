import type { PageData, QueryParams } from '../types/common'
import type { CreateMixDesign, MixDesign } from '../types/mix-design'
import { apiClient } from './client'

export const mixDesignApi = {
  list: (params?: QueryParams) => apiClient.get<PageData<MixDesign>>('/mix-designs', params),
  get: (id: number) => apiClient.get<MixDesign>(`/mix-designs/${id}`),
  create: (payload: CreateMixDesign) => apiClient.post<MixDesign>('/mix-designs', payload),
  update: (id: number, payload: Partial<CreateMixDesign> & { lock_version: number }) => apiClient.put<MixDesign>(`/mix-designs/${id}`, payload),
  transition: (id: number, action: 'validate' | 'publish' | 'retire', lockVersion: number, note: string) =>
    apiClient.post<MixDesign>(`/mix-designs/${id}/${action}`, { lock_version: lockVersion, note }),
}
