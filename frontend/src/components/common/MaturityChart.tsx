import { useEffect, useRef } from 'react'
import * as echarts from 'echarts'
import type { TemperaturePoint } from '../../types/temperature-series'
import type { MaturityStep } from '../../types/strength-forecast'
import { cumulativeMaturity, stepsToMaturity } from '../../utils/chart-data'

export function MaturityChart({ points, steps, datum = 0, height = 300 }: { points?: TemperaturePoint[]; steps?: MaturityStep[]; datum?: number; height?: number }) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!ref.current) return
    const chart = echarts.init(ref.current, undefined, { renderer: 'canvas' })
    const maturity = steps ? stepsToMaturity(steps) : cumulativeMaturity(points ?? [], datum)
    chart.setOption({
      animationDuration: 300,
      color: ['#2f6b4f', '#c27a24'],
      grid: { left: 52, right: 22, top: 36, bottom: 48 },
      tooltip: { trigger: 'axis', valueFormatter: (value: unknown) => `${Number(value).toFixed(1)} °C·h` },
      xAxis: { type: 'category', data: maturity.map((item) => new Date(item.timestamp).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit' })), axisLabel: { hideOverlap: true, color: '#59615d' } },
      yAxis: { type: 'value', name: '成熟度 (°C·h)', nameTextStyle: { color: '#59615d' }, splitLine: { lineStyle: { color: '#e1e4df' } } },
      series: [{ name: '累计成熟度', type: 'line', smooth: false, symbolSize: 6, areaStyle: { color: 'rgba(47,107,79,.10)' }, data: maturity.map((item) => item.value) }],
    })
    const observer = new ResizeObserver(() => chart.resize())
    observer.observe(ref.current)
    return () => { observer.disconnect(); chart.dispose() }
  }, [datum, points, steps])
  return <div ref={ref} style={{ width: '100%', height }} role="img" aria-label="累计成熟度曲线" />
}
