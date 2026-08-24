import type { PageData, QueryParams } from '../types/common'
import type { CreatePourSection, PourSection } from '../types/pour-section'
import type { CuringState } from '../types/enums/curing-state'
import { apiClient } from './client'

export const pourSectionApi = {
  list: (params?: QueryParams) => apiClient.get<PageData<PourSection>>('/pour-sections', params),
  get: (id: number) => apiClient.get<PourSection>(`/pour-sections/${id}`),
  create: (payload: CreatePourSection) => apiClient.post<PourSection>('/pour-sections', payload),
  update: (id: number, payload: Partial<CreatePourSection> & { version: number }) => apiClient.put<PourSection>(`/pour-sections/${id}`, payload),
  transition: (id: number, toState: CuringState, version: number, note: string) =>
    apiClient.post<PourSection>(`/pour-sections/${id}/transition`, { to_state: toState, version, note }),
}
