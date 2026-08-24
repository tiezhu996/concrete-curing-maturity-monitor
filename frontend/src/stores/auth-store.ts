import { create } from 'zustand'
import type { UserSession } from '../types/auth'

interface AuthState {
  user: UserSession | null
  expiresAt: string | null
  setSession: (token: string, expiresAt: string, user: UserSession) => void
  clearSession: () => void
}

function readUser(): UserSession | null {
  try {
    const raw = localStorage.getItem('curing-user')
    return raw ? (JSON.parse(raw) as UserSession) : null
  } catch {
    return null
  }
}

export const useAuthStore = create<AuthState>((set) => ({
  user: readUser(),
  expiresAt: localStorage.getItem('curing-token-expiry'),
  setSession: (token, expiresAt, user) => {
    localStorage.setItem('curing-access-token', token)
    localStorage.setItem('curing-token-expiry', expiresAt)
    localStorage.setItem('curing-user', JSON.stringify(user))
    set({ user, expiresAt })
  },
  clearSession: () => {
    localStorage.removeItem('curing-access-token')
    localStorage.removeItem('curing-token-expiry')
    localStorage.removeItem('curing-user')
    set({ user: null, expiresAt: null })
  },
}))
