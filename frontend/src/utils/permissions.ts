import type { UserRole } from '../types/auth'

export type Permission = 'section:write' | 'section:transition' | 'mix:write' | 'mix:publish' | 'temperature:write' | 'forecast:run' | 'forecast:review' | 'forecast:confirm' | 'audit:read'

const grants: Record<UserRole, Permission[]> = {
  admin: ['section:write', 'section:transition', 'mix:write', 'mix:publish', 'temperature:write', 'forecast:run', 'forecast:review', 'forecast:confirm', 'audit:read'],
  lab_engineer: ['mix:write', 'temperature:write', 'forecast:run'],
  site_engineer: ['section:write', 'section:transition', 'forecast:run'],
  reviewer: ['section:transition', 'mix:publish', 'temperature:write', 'forecast:review', 'forecast:confirm', 'audit:read'],
  auditor: ['audit:read'],
}

export const can = (role: UserRole | undefined, permission: Permission): boolean => Boolean(role && grants[role].includes(permission))
