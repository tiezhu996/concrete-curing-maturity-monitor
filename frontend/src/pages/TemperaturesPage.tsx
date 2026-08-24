import { useEffect, useMemo, useState } from 'react'
import { App, Button, Form, Input, InputNumber, Modal, Popconfirm, Select, Table, type TableColumnsType } from 'antd'
import { FileUp, LineChart, Plus } from 'lucide-react'
import { PageHeader } from '../components/common/PageHeader'
import { MaturityChart } from '../components/common/MaturityChart'
import { CuringStateBadge } from '../components/common/CuringStateBadge'
import { SeriesStateTag } from '../components/common/StatusTag'
import { useTemperatureSeriesStore } from '../stores/temperature-series-store'
import { usePourSectionStore } from '../stores/pour-section-store'
import { temperatureSeriesApi } from '../api/temperature-series-api'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { can } from '../utils/permissions'
import type { ImportTemperatureSeries, TemperaturePoint, TemperatureSeries } from '../types/temperature-series'
import { formatDateTime, shortHash } from '../utils/format'

interface TemperatureFormValues {
  pour_section_id: number
  sensor_code: string
  sample_interval_min: number
  quality_note?: string
  points_json: string
}

const samplePoints = JSON.stringify([
  { timestamp: '2026-08-22T01:00:00Z', temperature_c: 22.4 },
  { timestamp: '2026-08-22T03:00:00Z', temperature_c: 25.1 },
  { timestamp: '2026-08-22T05:00:00Z', temperature_c: 28.7 },
  { timestamp: '2026-08-22T07:00:00Z', temperature_c: 30.2 },
], null, 2)

export function TemperaturesPage() {
  const { message } = App.useApp()
  const { user } = useAuth()
  const { items, total, loading, fetch } = useTemperatureSeriesStore()
  const { items: sections, fetch: fetchSections } = usePourSectionStore()
  const [importOpen, setImportOpen] = useState(false)
  const [chartSeries, setChartSeries] = useState<TemperatureSeries | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<TemperatureFormValues>()
  const load = async () => {
    try { await fetch() } catch (reason) { message.error(errorMessage(reason)) }
  }
  useEffect(() => { void load(); void fetchSections().catch((reason) => message.error(errorMessage(reason))) }, [])

  const sectionById = useMemo(() => new Map(sections.map((section) => [section.id, section])), [sections])
  const counts = useMemo(() => ({ usable: items.filter((item) => item.series_state === 'usable').length, pending: items.filter((item) => item.series_state === 'imported').length, sensors: new Set(items.map((item) => item.sensor_code)).size }), [items])

  const importSeries = async () => {
    const values = await form.validateFields()
    let points: TemperaturePoint[]
    try {
      points = JSON.parse(values.points_json) as TemperaturePoint[]
      if (!Array.isArray(points) || points.length < 2) throw new Error('至少需要两个温度点')
    } catch (reason) {
      message.error(reason instanceof Error ? `温度 JSON 无效：${reason.message}` : '温度 JSON 无效')
      return
    }
    const payload: ImportTemperatureSeries = { ...values, points }
    delete (payload as ImportTemperatureSeries & { points_json?: string }).points_json
    setSaving(true)
    try {
      await temperatureSeriesApi.import(payload)
      message.success('温度序列已导入，等待确认'); setImportOpen(false); form.resetFields(); await load()
    } catch (reason) { message.error(errorMessage(reason)) } finally { setSaving(false) }
  }

  const changeState = async (item: TemperatureSeries, action: 'confirm' | 'invalidate') => {
    try {
      if (action === 'confirm') await temperatureSeriesApi.confirm(item.id, '工程试验室核对时间顺序与数据覆盖后确认')
      else await temperatureSeriesApi.invalidate(item.id, '工程工作台人工判定该序列不再用于计算')
      message.success(action === 'confirm' ? '序列已确认可用' : '序列已作废'); await load()
    } catch (reason) { message.error(errorMessage(reason)) }
  }

  const columns: TableColumnsType<TemperatureSeries> = [
    { title: '传感器 / 区段', fixed: 'left', width: 185, render: (_, item) => <div className="entity-cell"><strong>{item.sensor_code}</strong><span>{item.section_code}</span></div> },
    { title: '区段状态', width: 115, render: (_, item) => { const section = sectionById.get(item.pour_section_id); return section ? <CuringStateBadge state={section.curing_state} /> : '—' } },
    { title: '采样间隔', dataIndex: 'sample_interval_min', width: 100, render: (value: number) => `${value} 分钟` },
    { title: '时间范围', width: 220, render: (_, item) => `${formatDateTime(item.started_at)} → ${formatDateTime(item.ended_at)}` },
    { title: '点数', width: 70, render: (_, item) => item.points.length },
    { title: '缺失率', dataIndex: 'missing_ratio', width: 90, render: (value: number) => <span className={value > 0.08 ? 'value-danger' : ''}>{(value * 100).toFixed(1)}%</span> },
    { title: '校验和', dataIndex: 'source_checksum', width: 150, render: (value: string) => <span className="mono">{shortHash(value)}</span> },
    { title: '状态', dataIndex: 'series_state', width: 110, render: (state: TemperatureSeries['series_state']) => <SeriesStateTag state={state} /> },
    { title: '质量说明', dataIndex: 'quality_note', width: 240, ellipsis: true },
    { title: '操作', fixed: 'right', width: 160, render: (_, item) => <div className="row-actions">
      <Button type="text" icon={<LineChart size={16} />} onClick={() => setChartSeries(item)} aria-label="查看成熟度曲线" />
      {can(user?.role, 'temperature:write') && item.series_state === 'imported' && <Popconfirm title="确认序列可用于预测？" onConfirm={() => void changeState(item, 'confirm')}><Button type="link" size="small">确认</Button></Popconfirm>}
      {can(user?.role, 'temperature:write') && item.series_state !== 'invalid' && <Popconfirm title="确认作废该序列？" onConfirm={() => void changeState(item, 'invalidate')}><Button type="link" danger size="small">作废</Button></Popconfirm>}
    </div> },
  ]

  return (
    <div className="page-stack">
      <PageHeader eyebrow="CURING TELEMETRY" title="温度序列" description="导入真实观测点，检查时间顺序、覆盖率与来源校验和。" actions={can(user?.role, 'temperature:write') ? <Button type="primary" icon={<FileUp size={16} />} onClick={() => setImportOpen(true)}>导入序列</Button> : undefined} />
      <div className="metric-strip"><div><span>全部序列</span><strong>{total}</strong></div><div><span>可用于计算</span><strong>{counts.usable}</strong></div><div><span>待确认</span><strong>{counts.pending}</strong></div><div><span>传感器</span><strong>{counts.sensors}</strong></div></div>
      <section className="work-section">
        <div className="table-toolbar"><span className="table-context">按开始时间倒序 · 缺失率上限 10%</span></div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={items} pagination={{ total, pageSize: 20, showSizeChanger: false }} scroll={{ x: 1450 }} />
      </section>
      <Modal title="导入温度序列" open={importOpen} onCancel={() => setImportOpen(false)} onOk={() => void importSeries()} confirmLoading={saving} okText="导入并校验" width={720} destroyOnClose>
        <Form form={form} layout="vertical" className="two-column-form" initialValues={{ sample_interval_min: 120, points_json: samplePoints, quality_note: '现场导出文件，经试验室人工复核' }}>
          <Form.Item label="浇筑区段" name="pour_section_id" rules={[{ required: true }]}><Select options={sections.filter((section) => section.curing_state !== 'closed').map((section) => ({ value: section.id, label: `${section.section_code} · ${section.name}` }))} /></Form.Item>
          <Form.Item label="传感器编号" name="sensor_code" rules={[{ required: true }, { min: 2 }]}><Input placeholder="例如 TC-B2-W08-A" /></Form.Item>
          <Form.Item label="采样间隔（分钟）" name="sample_interval_min" rules={[{ required: true }]}><InputNumber min={1} max={1440} /></Form.Item>
          <Form.Item label="质量说明" name="quality_note"><Input /></Form.Item>
          <Form.Item className="form-list-span" label="观测点 JSON" name="points_json" rules={[{ required: true }]} extra="时间戳必须严格递增，系统不会补造传感器点。"><Input.TextArea rows={13} className="code-input" /></Form.Item>
        </Form>
      </Modal>
      <Modal title={chartSeries ? `${chartSeries.sensor_code} · 成熟度累计` : '成熟度累计'} open={Boolean(chartSeries)} onCancel={() => setChartSeries(null)} footer={null} width={780} destroyOnClose>
        {chartSeries && <MaturityChart points={chartSeries.points} height={340} />}
      </Modal>
    </div>
  )
}
