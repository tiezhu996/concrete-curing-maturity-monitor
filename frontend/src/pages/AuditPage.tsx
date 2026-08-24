import { useEffect, useMemo, useState } from 'react'
import { App, Button, Input, Select, Table, Tag, type TableColumnsType } from 'antd'
import { FileJson } from 'lucide-react'
import { PageHeader } from '../components/common/PageHeader'
import { CalculationDrawer } from '../components/common/CalculationDrawer'
import { auditApi } from '../api/audit-api'
import { errorMessage } from '../api/client'
import type { AuditLog } from '../types/audit'
import { formatDateTime, shortHash } from '../utils/format'
import { useAuth } from '../hooks/useAuth'
import { can } from '../utils/permissions'

function parseEvidence(entry: AuditLog): unknown {
  const parse = (value: string) => { try { return JSON.parse(value) as unknown } catch { return value } }
  return { before: parse(entry.before_json), after: parse(entry.after_json), metadata: parse(entry.metadata_json) }
}

export function AuditPage() {
  const { message } = App.useApp()
  const { user } = useAuth()
  const [items, setItems] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [evidence, setEvidence] = useState<unknown>()
  const [entityType, setEntityType] = useState('')
  const [requestId, setRequestId] = useState('')

  const load = async (entity = entityType, request = requestId) => {
    if (!can(user?.role, 'audit:read')) return
    setLoading(true)
    try {
      const data = await auditApi.list({ page: 1, page_size: 100, entity_type: entity, request_id: request })
      setItems(data.items); setTotal(data.total)
    } catch (reason) { message.error(errorMessage(reason)) } finally { setLoading(false) }
  }
  useEffect(() => { void load() }, [])

  const actors = useMemo(() => new Set(items.map((item) => item.actor_name)).size, [items])
  const requests = useMemo(() => new Set(items.map((item) => item.request_id)).size, [items])
  const entityLabels: Record<string, string> = { pour_section: '浇筑区段', mix_design: '配合比', temperature_series: '温度序列', strength_forecast: '强度预测' }
  const actionColors: Record<string, string> = { create: 'blue', import: 'cyan', transition: 'gold', forecast_complete: 'green', review: 'purple', confirm: 'green', void: 'volcano', invalidate: 'volcano' }

  const columns: TableColumnsType<AuditLog> = [
    { title: '时间', dataIndex: 'created_at', fixed: 'left', width: 145, render: formatDateTime },
    { title: '操作者', width: 155, render: (_, item) => <div className="entity-cell"><strong>{item.actor_name}</strong><span>{item.actor_role}</span></div> },
    { title: '实体', dataIndex: 'entity_type', width: 130, render: (value: string) => entityLabels[value] ?? value },
    { title: '实体 ID', dataIndex: 'entity_id', width: 85, render: (value: number) => <span className="mono">#{value}</span> },
    { title: '动作', dataIndex: 'action', width: 150, render: (value: string) => <Tag color={actionColors[value] ?? 'default'}>{value}</Tag> },
    { title: 'Request ID', dataIndex: 'request_id', width: 180, render: (value: string) => <span className="mono">{shortHash(value)}</span> },
    { title: '证据', fixed: 'right', width: 100, render: (_, item) => <Button type="link" icon={<FileJson size={15} />} onClick={() => setEvidence(parseEvidence(item))}>查看</Button> },
  ]

  if (!can(user?.role, 'audit:read')) {
    return <div className="page-stack"><PageHeader eyebrow="AUDIT TRAIL" title="审计中心" description="当前角色无权读取工程审计记录。" /></div>
  }

  return (
    <div className="page-stack">
      <PageHeader eyebrow="AUDIT TRAIL" title="审计中心" description="按请求追踪四个核心实体的前后快照、操作者和计算摘要。" />
      <div className="metric-strip"><div><span>审计事件</span><strong>{total}</strong></div><div><span>独立请求</span><strong>{requests}</strong></div><div><span>操作者</span><strong>{actors}</strong></div><div><span>保留策略</span><strong><small>不可覆写</small></strong></div></div>
      <section className="work-section">
        <div className="table-toolbar">
          <Select placeholder="全部实体" allowClear value={entityType || undefined} onChange={(value) => { setEntityType(value ?? ''); void load(value ?? '', requestId) }} options={Object.entries(entityLabels).map(([value, label]) => ({ value, label }))} />
          <Input.Search placeholder="精确查找 Request ID" allowClear value={requestId} onChange={(event) => setRequestId(event.target.value)} onSearch={(value) => void load(entityType, value)} />
        </div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={items} pagination={{ total, pageSize: 20, showSizeChanger: false }} scroll={{ x: 1000 }} />
      </section>
      <CalculationDrawer open={evidence !== undefined} onClose={() => setEvidence(undefined)} rawEvidence={evidence} />
    </div>
  )
}
