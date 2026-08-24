import { useEffect, useMemo, useState } from 'react'
import { App, Button, Dropdown, Form, Input, InputNumber, Modal, Select, Space, Table, type TableColumnsType } from 'antd'
import { ChevronRight, Plus, Search } from 'lucide-react'
import { PageHeader } from '../components/common/PageHeader'
import { CuringStateBadge } from '../components/common/CuringStateBadge'
import { usePourSectionStore } from '../stores/pour-section-store'
import { useMixDesignStore } from '../stores/mix-design-store'
import { pourSectionApi } from '../api/pour-section-api'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { can } from '../utils/permissions'
import { CURING_STATE_LABELS, CURING_TRANSITIONS, type CuringState } from '../types/enums/curing-state'
import type { CreatePourSection, PourSection } from '../types/pour-section'
import { formatDateTime, formatNumber } from '../utils/format'

interface SectionFormValues extends Omit<CreatePourSection, 'poured_at'> { poured_at_local?: string }

export function SectionsPage() {
  const { message, modal } = App.useApp()
  const { user } = useAuth()
  const { items, total, loading, fetch } = usePourSectionStore()
  const { items: mixes, fetch: fetchMixes } = useMixDesignStore()
  const [createOpen, setCreateOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<SectionFormValues>()

  const load = async (search = '') => {
    try { await fetch({ search }) } catch (reason) { message.error(errorMessage(reason)) }
  }
  useEffect(() => { void load(); void fetchMixes({ design_state: 'published' }).catch((reason) => message.error(errorMessage(reason))) }, [])

  const summary = useMemo(() => ({
    active: items.filter((item) => item.curing_state === 'curing').length,
    threshold: items.filter((item) => item.curing_state === 'threshold_reached').length,
    volume: items.reduce((sum, item) => sum + item.volume_m3, 0),
  }), [items])

  const create = async () => {
    const values = await form.validateFields()
    setSaving(true)
    try {
      const payload: CreatePourSection = {
        ...values,
        poured_at: values.poured_at_local ? new Date(values.poured_at_local).toISOString() : undefined,
      }
      delete (payload as CreatePourSection & { poured_at_local?: string }).poured_at_local
      await pourSectionApi.create(payload)
      message.success('浇筑区段已建档')
      setCreateOpen(false); form.resetFields(); await load()
    } catch (reason) {
      if (reason instanceof Error) message.error(errorMessage(reason))
    } finally { setSaving(false) }
  }

  const transition = (section: PourSection, next: CuringState) => {
    modal.confirm({
      title: `确认变更为“${CURING_STATE_LABELS[next]}”`,
      content: `${section.section_code} 的状态变化将写入审计记录。`,
      okText: '确认变更', cancelText: '取消',
      onOk: async () => {
        try {
          await pourSectionApi.transition(section.id, next, section.version, `工程工作台状态变更：${CURING_STATE_LABELS[next]}`)
          message.success('区段状态已更新'); await load()
        } catch (reason) { message.error(errorMessage(reason)); throw reason }
      },
    })
  }

  const columns: TableColumnsType<PourSection> = [
    { title: '区段', key: 'section', fixed: 'left', width: 190, render: (_, item) => <div className="entity-cell"><strong>{item.section_code}</strong><span>{item.name}</span></div> },
    { title: '结构部位', dataIndex: 'structure_part', width: 180, ellipsis: true },
    { title: '配合比', width: 130, render: (_, item) => <span className="mono">{item.mix_code} / v{item.mix_version}</span> },
    { title: '目标', width: 110, render: (_, item) => <strong>{formatNumber(item.target_strength_mpa)} MPa</strong> },
    { title: '最新预测', width: 130, render: (_, item) => item.summary.latest_strength_mpa ? `${formatNumber(item.summary.latest_strength_mpa)} MPa` : '尚未计算' },
    { title: '状态', dataIndex: 'curing_state', width: 110, render: (state: CuringState) => <CuringStateBadge state={state} /> },
    { title: '浇筑时间', dataIndex: 'poured_at', width: 130, render: formatDateTime },
    { title: '责任班组', dataIndex: 'owner_team', width: 130 },
    { title: '操作', key: 'actions', fixed: 'right', width: 104, render: (_, item) => {
      const transitions = CURING_TRANSITIONS[item.curing_state]
      return can(user?.role, 'section:transition') && transitions.length ? (
        <Dropdown menu={{ items: transitions.map((state) => ({ key: state, label: CURING_STATE_LABELS[state] })), onClick: ({ key }) => transition(item, key as CuringState) }}>
          <Button type="link" size="small">推进 <ChevronRight size={14} /></Button>
        </Dropdown>
      ) : <span className="muted">只读</span>
    } },
  ]

  return (
    <div className="page-stack">
      <PageHeader eyebrow="POUR SECTIONS" title="浇筑区段" description="维护浇筑上下文、目标强度和受控养护状态。" actions={can(user?.role, 'section:write') ? <Button type="primary" icon={<Plus size={16} />} onClick={() => setCreateOpen(true)}>新建区段</Button> : undefined} />
      <div className="metric-strip">
        <div><span>全部区段</span><strong>{total}</strong></div><div><span>养护中</span><strong>{summary.active}</strong></div><div><span>达到阈值</span><strong>{summary.threshold}</strong></div><div><span>记录方量</span><strong>{formatNumber(summary.volume)} <small>m³</small></strong></div>
      </div>
      <section className="work-section">
        <div className="table-toolbar"><Input.Search prefix={<Search size={16} />} placeholder="搜索区段编号、名称或结构部位" allowClear onSearch={load} /></div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={items} pagination={{ total, pageSize: 20, showSizeChanger: false }} scroll={{ x: 1250 }} />
      </section>
      <Modal title="新建浇筑区段" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={() => void create()} confirmLoading={saving} okText="建档" width={680} destroyOnClose>
        <Form form={form} layout="vertical" className="two-column-form" initialValues={{ target_strength_mpa: 28, volume_m3: 60 }}>
          <Form.Item label="区段编号" name="section_code" rules={[{ required: true }, { min: 2 }]}><Input placeholder="例如 B2-W08" /></Form.Item>
          <Form.Item label="区段名称" name="name" rules={[{ required: true }, { min: 2 }]}><Input /></Form.Item>
          <Form.Item label="结构部位" name="structure_part" rules={[{ required: true }, { min: 2 }]}><Input /></Form.Item>
          <Form.Item label="责任班组" name="owner_team" rules={[{ required: true }, { min: 2 }]}><Input /></Form.Item>
          <Form.Item label="方量 (m³)" name="volume_m3" rules={[{ required: true }]}><InputNumber min={0.1} max={100000} precision={1} /></Form.Item>
          <Form.Item label="目标强度 (MPa)" name="target_strength_mpa" rules={[{ required: true }]}><InputNumber min={0.1} max={200} precision={1} /></Form.Item>
          <Form.Item label="已发布配合比" name="mix_design_id" rules={[{ required: true }]}><Select options={mixes.map((mix) => ({ value: mix.id, label: `${mix.mix_code} / v${mix.version}` }))} /></Form.Item>
          <Form.Item label="浇筑时间（可选）" name="poured_at_local"><Input type="datetime-local" /></Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
