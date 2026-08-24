import type { MaturityStep } from '../types/strength-forecast'
import type { TemperaturePoint } from '../types/temperature-series'

export function cumulativeMaturity(points: TemperaturePoint[], datum = 0): Array<{ timestamp: string; value: number }> {
  if (points.length === 0) return []
  let total = 0
  const output = [{ timestamp: points[0].timestamp, value: 0 }]
  for (let index = 1; index < points.length; index += 1) {
    const previous = points[index - 1]
    const current = points[index]
    const hours = (new Date(current.timestamp).getTime() - new Date(previous.timestamp).getTime()) / 3_600_000
    const average = (previous.temperature_c + current.temperature_c) / 2
    total += Math.max(0, average - datum) * hours
    output.push({ timestamp: current.timestamp, value: Math.round(total * 100) / 100 })
  }
  return output
}

export function stepsToMaturity(steps: MaturityStep[]): Array<{ timestamp: string; value: number }> {
  if (!steps.length) return []
  return [{ timestamp: steps[0].from, value: 0 }, ...steps.map((step) => ({ timestamp: step.to, value: step.running_maturity_degree_hours }))]
}
