import { Descriptions, Drawer, Table, Typography } from 'antd'
import type { CalibrationPoint } from '../../types/mix-design'
import type { StrengthForecast } from '../../types/strength-forecast'
import { MaturityChart } from './MaturityChart'
import { formatNumber, shortHash } from '../../utils/format'
import { SafetyNotice } from './SafetyNotice'

export function CalculationDrawer({ open, onClose, forecast, calibration, rawEvidence }: { open: boolean; onClose: () => void; forecast?: StrengthForecast | null; calibration?: CalibrationPoint[]; rawEvidence?: unknown }) {
  const points = calibration ?? ((forecast?.explanation?.interpolation ? [] : []) as CalibrationPoint[])
  return (
    <Drawer title="计算与标定证据" open={open} onClose={onClose} width={640} className="calculation-drawer">
      {forecast ? (
        <>
          <SafetyNotice compact />
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="公式版本">{forecast.formula_version}</Descriptions.Item>
            <Descriptions.Item label="输入哈希">{shortHash(forecast.input_hash)}</Descriptions.Item>
            <Descriptions.Item label="累计成熟度">{formatNumber(forecast.maturity_degree_hours, 2)} °C·h</Descriptions.Item>
            <Descriptions.Item label="预测强度">{formatNumber(forecast.predicted_strength_mpa, 2)} MPa</Descriptions.Item>
            <Descriptions.Item label="数据覆盖">{formatNumber(forecast.explanation?.data_coverage_percent, 1)}%</Descriptions.Item>
            <Descriptions.Item label="最近成熟速率">{formatNumber(forecast.explanation?.recent_maturity_rate_per_hour, 2)} °C·h/h</Descriptions.Item>
          </Descriptions>
          <div className="drawer-section">
            <Typography.Title level={5}>逐段累计</Typography.Title>
            <MaturityChart steps={forecast.explanation?.maturity?.steps ?? []} height={260} />
          </div>
          <pre className="evidence-json">{JSON.stringify(forecast.explanation?.interpolation ?? {}, null, 2)}</pre>
        </>
      ) : rawEvidence !== undefined ? (
        <pre className="evidence-json">{JSON.stringify(rawEvidence, null, 2)}</pre>
      ) : (
        <Table rowKey="maturity_degree_hours" size="small" pagination={false} dataSource={points} columns={[
          { title: '成熟度 (°C·h)', dataIndex: 'maturity_degree_hours' },
          { title: '强度 (MPa)', dataIndex: 'strength_mpa' },
        ]} />
      )}
    </Drawer>
  )
}
