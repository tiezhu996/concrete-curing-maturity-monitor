export type UserRole = 'admin' | 'lab_engineer' | 'site_engineer' | 'reviewer' | 'auditor'

export interface UserSession {
  id: number
  username: string
  display_name: string
  role: UserRole
}

export interface LoginResponse {
  token: string
  expires_at: string
  user: UserSession
}
