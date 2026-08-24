import type { AuditLog } from '../types/audit'
import type { PageData, QueryParams } from '../types/common'
import { apiClient } from './client'

export const auditApi = {
  list: (params?: QueryParams) => apiClient.get<PageData<AuditLog>>('/audit-logs', params),
}
