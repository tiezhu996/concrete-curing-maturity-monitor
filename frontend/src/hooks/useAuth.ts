import { useCallback, useEffect } from 'react'
import { authApi } from '../api/auth-api'
import { useAuthStore } from '../stores/auth-store'

export function useAuth() {
  const { user, expiresAt, setSession, clearSession } = useAuthStore()

  useEffect(() => {
    const expire = () => clearSession()
    window.addEventListener('auth-expired', expire)
    if (expiresAt && new Date(expiresAt).getTime() <= Date.now()) clearSession()
    return () => window.removeEventListener('auth-expired', expire)
  }, [clearSession, expiresAt])

  const login = useCallback(async (username: string, password: string) => {
    const session = await authApi.login(username, password)
    setSession(session.token, session.expires_at, session.user)
    return session.user
  }, [setSession])

  return {
    user,
    authenticated: Boolean(user && localStorage.getItem('curing-access-token')),
    login,
    logout: clearSession,
  }
}
