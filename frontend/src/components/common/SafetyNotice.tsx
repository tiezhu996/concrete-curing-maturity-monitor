import { ShieldAlert } from 'lucide-react'

export function SafetyNotice({ compact = false }: { compact?: boolean }) {
  return (
    <div className={compact ? 'safety-notice compact' : 'safety-notice'} role="note">
      <ShieldAlert size={16} aria-hidden="true" />
      <span>离线决策支持：预测不替代实体试块强度报告、现场核验与持证工程师签认。</span>
    </div>
  )
}
