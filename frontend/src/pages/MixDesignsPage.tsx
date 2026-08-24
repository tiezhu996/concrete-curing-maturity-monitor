import { useEffect, useMemo, useState } from 'react'
import { App, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, type TableColumnsType } from 'antd'
import { FileSearch, Plus, Trash2 } from 'lucide-react'
import { PageHeader } from '../components/common/PageHeader'
import { CalculationDrawer } from '../components/common/CalculationDrawer'
import { useMixDesignStore } from '../stores/mix-design-store'
import { mixDesignApi } from '../api/mix-design-api'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { can } from '../utils/permissions'
import type { CalibrationPoint, CreateMixDesign, MixDesign, MixDesignState } from '../types/mix-design'
import { formatDateTime } from '../utils/format'

const stateLabels: Record<MixDesignState, string> = { draft: '草稿', validated: '已校验', published: '已发布', retired: '已废止' }
const stateColors: Record<MixDesignState, string> = { draft: 'default', validated: 'blue', published: 'green', retired: 'volcano' }

export function MixDesignsPage() {
  const { message } = App.useApp()
  const { user } = useAuth()
  const { items, total, loading, fetch } = useMixDesignStore()
  const [createOpen, setCreateOpen] = useState(false)
  const [evidence, setEvidence] = useState<CalibrationPoint[] | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<CreateMixDesign>()
  const load = async (search = '', designState = '') => {
    try { await fetch({ search, design_state: designState }) } catch (reason) { message.error(errorMessage(reason)) }
  }
  useEffect(() => { void load() }, [])

  const counts = useMemo(() => ({
    published: items.filter((item) => item.design_state === 'published').length,
    drafts: items.filter((item) => item.design_state === 'draft').length,
    references: items.reduce((sum, item) => sum + item.referenced_sections, 0),
  }), [items])

  const create = async () => {
    const values = await form.validateFields()
    setSaving(true)
    try {
      await mixDesignApi.create(values)
      message.success('配合比草稿已创建'); form.resetFields(); setCreateOpen(false); await load()
    } catch (reason) { message.error(errorMessage(reason)) } finally { setSaving(false) }
  }

  const transition = async (item: MixDesign, action: 'validate' | 'publish' | 'retire') => {
    try {
      await mixDesignApi.transition(item.id, action, item.lock_version, `工程工作台执行${action}`)
      message.success('配合比状态已更新'); await load()
    } catch (reason) { message.error(errorMessage(reason)) }
  }

  const actionFor = (item: MixDesign) => {
    if (item.design_state === 'draft' && can(user?.role, 'mix:write')) return { action: 'validate' as const, label: '校验' }
    if (item.design_state === 'validated' && can(user?.role, 'mix:publish')) return { action: 'publish' as const, label: '发布' }
    if (item.design_state === 'published' && can(user?.role, 'mix:publish')) return { action: 'retire' as const, label: '废止' }
    return null
  }

  const columns: TableColumnsType<MixDesign> = [
    { title: '配合比 / 版本', fixed: 'left', width: 170, render: (_, item) => <div className="entity-cell"><strong>{item.mix_code}</strong><span>版本 {item.version}</span></div> },
    { title: '水泥类型', dataIndex: 'cement_type', width: 220, ellipsis: true },
    { title: '水胶比', dataIndex: 'water_binder_ratio', width: 90, render: (value: number) => value.toFixed(2) },
    { title: '基准温度', dataIndex: 'datum_temperature_c', width: 110, render: (value: number) => `${value.toFixed(1)} °C` },
    { title: '标定点', width: 100, render: (_, item) => <Button type="link" icon={<FileSearch size={15} />} onClick={() => setEvidence(item.calibration_points)}>{item.calibration_points.length} 点</Button> },
    { title: '区段引用', dataIndex: 'referenced_sections', width: 100 },
    { title: '状态', dataIndex: 'design_state', width: 100, render: (state: MixDesignState) => <Tag color={stateColors[state]}>{stateLabels[state]}</Tag> },
    { title: '创建人', dataIndex: 'created_by_name', width: 130 },
    { title: '更新时间', dataIndex: 'updated_at', width: 130, render: formatDateTime },
    { title: '操作', fixed: 'right', width: 90, render: (_, item) => {
      const action = actionFor(item)
      return action ? <Popconfirm title={`确认${action.label}该版本？`} description="状态变化不可由界面直接撤销。" onConfirm={() => void transition(item, action.action)}><Button type="link" size="small">{action.label}</Button></Popconfirm> : <span className="muted">只读</span>
    } },
  ]

  return (
    <div className="page-stack">
      <PageHeader eyebrow="MIX CALIBRATION" title="配合比版本" description="发布单调标定曲线并冻结养护计算所依据的版本。" actions={can(user?.role, 'mix:write') ? <Button type="primary" icon={<Plus size={16} />} onClick={() => setCreateOpen(true)}>新建版本</Button> : undefined} />
      <div className="metric-strip"><div><span>全部版本</span><strong>{total}</strong></div><div><span>已发布</span><strong>{counts.published}</strong></div><div><span>待完善草稿</span><strong>{counts.drafts}</strong></div><div><span>区段引用</span><strong>{counts.references}</strong></div></div>
      <section className="work-section">
        <div className="table-toolbar">
          <Input.Search placeholder="搜索配合比编号或水泥类型" allowClear onSearch={(value) => void load(value)} />
          <Select placeholder="全部状态" allowClear onChange={(value) => void load('', value ?? '')} options={Object.entries(stateLabels).map(([value, label]) => ({ value, label }))} />
        </div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={items} pagination={{ total, pageSize: 20, showSizeChanger: false }} scroll={{ x: 1200 }} />
      </section>
      <Modal title="新建配合比版本" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={() => void create()} confirmLoading={saving} okText="创建草稿" width={760} destroyOnClose>
        <Form form={form} layout="vertical" className="two-column-form" initialValues={{ version: 1, water_binder_ratio: 0.42, datum_temperature_c: 0, calibration_points: [{ maturity_degree_hours: 0, strength_mpa: 0 }, { maturity_degree_hours: 120, strength_mpa: 8.5 }, { maturity_degree_hours: 360, strength_mpa: 18.2 }, { maturity_degree_hours: 720, strength_mpa: 28.5 }] }}>
          <Form.Item label="配合比编号" name="mix_code" rules={[{ required: true }, { min: 2 }]}><Input placeholder="例如 C40-P45" /></Form.Item>
          <Form.Item label="版本" name="version" rules={[{ required: true }]}><InputNumber min={1} max={1000} /></Form.Item>
          <Form.Item label="水泥类型" name="cement_type" rules={[{ required: true }, { min: 2 }]}><Input /></Form.Item>
          <Form.Item label="水胶比" name="water_binder_ratio" rules={[{ required: true }]}><InputNumber min={0.01} max={1} step={0.01} precision={2} /></Form.Item>
          <Form.Item label="基准温度 (°C)" name="datum_temperature_c" rules={[{ required: true }]}><InputNumber min={-30} max={40} precision={1} /></Form.Item>
          <div />
          <div className="form-list-span">
            <span className="field-label">成熟度—强度标定点</span>
            <Form.List name="calibration_points">
              {(fields, { add, remove }) => <div className="calibration-list">
                {fields.map((field) => <Space key={field.key} align="baseline">
                  <Form.Item {...field} name={[field.name, 'maturity_degree_hours']} rules={[{ required: true }]}><InputNumber min={0} placeholder="成熟度 °C·h" /></Form.Item>
                  <Form.Item {...field} name={[field.name, 'strength_mpa']} rules={[{ required: true }]}><InputNumber min={0} placeholder="强度 MPa" /></Form.Item>
                  <Button type="text" danger icon={<Trash2 size={16} />} onClick={() => remove(field.name)} aria-label="删除标定点" disabled={fields.length <= 2} />
                </Space>)}
                <Button type="dashed" onClick={() => add()} block>增加标定点</Button>
              </div>}
            </Form.List>
          </div>
        </Form>
      </Modal>
      <CalculationDrawer open={Boolean(evidence)} onClose={() => setEvidence(null)} calibration={evidence ?? undefined} />
    </div>
  )
}
