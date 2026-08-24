import { useEffect, useMemo, useState } from 'react'
import { App, Button, Form, Modal, Popconfirm, Select, Table, Tooltip, type TableColumnsType } from 'antd'
import { Calculator, Eye, RotateCcw } from 'lucide-react'
import { PageHeader } from '../components/common/PageHeader'
import { CalculationDrawer } from '../components/common/CalculationDrawer'
import { CuringStateBadge } from '../components/common/CuringStateBadge'
import { ConfidenceTag, ForecastStateTag } from '../components/common/StatusTag'
import { useStrengthForecastStore } from '../stores/strength-forecast-store'
import { usePourSectionStore } from '../stores/pour-section-store'
import { useTemperatureSeriesStore } from '../stores/temperature-series-store'
import { useForecastRun } from '../hooks/useForecastRun'
import { strengthForecastApi } from '../api/strength-forecast-api'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { can } from '../utils/permissions'
import type { StrengthForecast } from '../types/strength-forecast'
import { formatDateTime, formatNumber, shortHash } from '../utils/format'

interface RunFormValues { pour_section_id: number; temperature_series_id: number }

export function ForecastsPage() {
  const { message } = App.useApp()
  const { user } = useAuth()
  const { items, total, loading, fetch } = useStrengthForecastStore()
  const { items: sections, fetch: fetchSections } = usePourSectionStore()
  const { items: series, fetch: fetchSeries } = useTemperatureSeriesStore()
  const { running, run } = useForecastRun()
  const [runOpen, setRunOpen] = useState(false)
  const [selected, setSelected] = useState<StrengthForecast | null>(null)
  const [form] = Form.useForm<RunFormValues>()
  const selectedSection = Form.useWatch('pour_section_id', form)
  const load = async () => {
    try { await fetch() } catch (reason) { message.error(errorMessage(reason)) }
  }
  useEffect(() => {
    void load(); void fetchSections().catch((reason) => message.error(errorMessage(reason))); void fetchSeries({ series_state: 'usable' }).catch((reason) => message.error(errorMessage(reason)))
  }, [])

  const sectionById = useMemo(() => new Map(sections.map((section) => [section.id, section])), [sections])
  const counts = useMemo(() => ({
    confirmed: items.filter((item) => item.forecast_state === 'confirmed').length,
    review: items.filter((item) => item.forecast_state === 'completed').length,
    high: items.filter((item) => item.confidence_level === 'high').length,
  }), [items])

  const runForecast = async () => {
    const values = await form.validateFields()
    try {
      const forecast = await run(values.pour_section_id, values.temperature_series_id)
      message.success('成熟度预测已完成'); setRunOpen(false); form.resetFields(); setSelected(forecast); await load()
    } catch (reason) { message.error(errorMessage(reason)) }
  }

  const action = async (item: StrengthForecast, actionName: 'review' | 'confirm' | 'void') => {
    try {
      await strengthForecastApi.action(item.id, actionName, `工程工作台执行预测${actionName}，已核对计算证据`)
      message.success('预测状态已更新'); await load()
    } catch (reason) { message.error(errorMessage(reason)) }
  }

  const replay = async (item: StrengthForecast) => {
    try {
      await strengthForecastApi.replay(item.id); message.success('确定性重放通过，结果与冻结输入一致'); await load()
    } catch (reason) { message.error(errorMessage(reason)) }
  }

  const columns: TableColumnsType<StrengthForecast> = [
    { title: '预测 / 区段', fixed: 'left', width: 160, render: (_, item) => <div className="entity-cell"><strong>FC-{String(item.id).padStart(4, '0')}</strong><span>{item.section_code}</span></div> },
    { title: '区段状态', width: 115, render: (_, item) => { const section = sectionById.get(item.pour_section_id); return section ? <CuringStateBadge state={section.curing_state} /> : '—' } },
    { title: '输入序列', dataIndex: 'sensor_code', width: 150 },
    { title: '成熟度', dataIndex: 'maturity_degree_hours', width: 125, render: (value: number) => <strong>{formatNumber(value, 1)} °C·h</strong> },
    { title: '预测强度', dataIndex: 'predicted_strength_mpa', width: 120, render: (value: number) => <strong className="strength-value">{formatNumber(value, 2)} MPa</strong> },
    { title: '阈值 ETA', dataIndex: 'threshold_eta', width: 130, render: formatDateTime },
    { title: '置信度', dataIndex: 'confidence_level', width: 100, render: (level: StrengthForecast['confidence_level']) => <ConfidenceTag level={level} /> },
    { title: '流程状态', dataIndex: 'forecast_state', width: 100, render: (state: string) => <ForecastStateTag state={state} /> },
    { title: '输入哈希', dataIndex: 'input_hash', width: 145, render: (value: string) => <span className="mono">{shortHash(value)}</span> },
    { title: '计算人 / 时间', width: 170, render: (_, item) => <div className="entity-cell"><strong>{item.calculated_by_name}</strong><span>{formatDateTime(item.calculated_at)}</span></div> },
    { title: '操作', fixed: 'right', width: 230, render: (_, item) => <div className="row-actions">
      <Tooltip title="查看计算证据"><Button type="text" icon={<Eye size={16} />} onClick={() => setSelected(item)} aria-label="查看计算证据" /></Tooltip>
      {can(user?.role, 'forecast:run') && <Tooltip title="确定性重放"><Button type="text" icon={<RotateCcw size={16} />} onClick={() => void replay(item)} aria-label="确定性重放" /></Tooltip>}
      {item.forecast_state === 'completed' && can(user?.role, 'forecast:review') && <Popconfirm title="确认已核对计算证据？" onConfirm={() => void action(item, 'review')}><Button type="link" size="small">评审</Button></Popconfirm>}
      {item.forecast_state === 'reviewed' && can(user?.role, 'forecast:confirm') && <Tooltip title={item.calculated_by === user?.id ? '发起人不能确认自己的结果' : ''}><span><Popconfirm title="确认该预测结果？" disabled={item.calculated_by === user?.id} onConfirm={() => void action(item, 'confirm')}><Button type="link" size="small" disabled={item.calculated_by === user?.id}>确认</Button></Popconfirm></span></Tooltip>}
      {['completed', 'reviewed', 'failed'].includes(item.forecast_state) && can(user?.role, 'forecast:review') && <Popconfirm title="确认作废该历史结果？" onConfirm={() => void action(item, 'void')}><Button type="link" danger size="small">作废</Button></Popconfirm>}
    </div> },
  ]

  return (
    <div className="page-stack">
      <PageHeader eyebrow="MATURITY FORECAST" title="成熟度预测" description="冻结输入与公式版本，呈现插值证据、阈值时间和签认链。" actions={can(user?.role, 'forecast:run') ? <Button type="primary" icon={<Calculator size={16} />} onClick={() => setRunOpen(true)}>运行预测</Button> : undefined} />
      <div className="metric-strip"><div><span>历史结果</span><strong>{total}</strong></div><div><span>待评审</span><strong>{counts.review}</strong></div><div><span>已确认</span><strong>{counts.confirmed}</strong></div><div><span>高置信结果</span><strong>{counts.high}</strong></div></div>
      <section className="work-section">
        <div className="table-toolbar"><span className="table-context">Nurse-Saul · 单调分段线性插值 · 不做无依据外推</span></div>
        <Table rowKey="id" loading={loading} columns={columns} dataSource={items} pagination={{ total, pageSize: 20, showSizeChanger: false }} scroll={{ x: 1600 }} />
      </section>
      <Modal title="运行成熟度预测" open={runOpen} onCancel={() => setRunOpen(false)} onOk={() => void runForecast()} confirmLoading={running} okText="冻结输入并计算" width={560} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item label="浇筑区段" name="pour_section_id" rules={[{ required: true }]}><Select onChange={() => form.setFieldValue('temperature_series_id', undefined)} options={sections.filter((section) => section.curing_state !== 'closed').map((section) => ({ value: section.id, label: `${section.section_code} · 目标 ${section.target_strength_mpa} MPa` }))} /></Form.Item>
          <Form.Item label="已确认温度序列" name="temperature_series_id" rules={[{ required: true }]}><Select disabled={!selectedSection} options={series.filter((item) => item.pour_section_id === selectedSection).map((item) => ({ value: item.id, label: `${item.sensor_code} · ${(item.missing_ratio * 100).toFixed(1)}% 缺失` }))} /></Form.Item>
        </Form>
      </Modal>
      <CalculationDrawer open={Boolean(selected)} onClose={() => setSelected(null)} forecast={selected} />
    </div>
  )
}
