import { useState } from 'react'
import { strengthForecastApi } from '../api/strength-forecast-api'
import type { StrengthForecast } from '../types/strength-forecast'

function idempotencyKey(sectionId: number, seriesId: number): string {
  const random = crypto.getRandomValues(new Uint32Array(2)).join('-')
  return `ui-${sectionId}-${seriesId}-${Date.now()}-${random}`
}

export function useForecastRun() {
  const [running, setRunning] = useState(false)
  const [latest, setLatest] = useState<StrengthForecast | null>(null)

  const run = async (sectionId: number, seriesId: number) => {
    setRunning(true)
    try {
      const forecast = await strengthForecastApi.run(sectionId, seriesId, idempotencyKey(sectionId, seriesId))
      setLatest(forecast)
      return forecast
    } finally {
      setRunning(false)
    }
  }

  return { running, latest, run }
}
