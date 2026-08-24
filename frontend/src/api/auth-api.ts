import type { LoginResponse } from '../types/auth'
import { apiClient } from './client'

export const authApi = {
  login: (username: string, password: string) => apiClient.post<LoginResponse>('/auth/login', { username, password }),
}
