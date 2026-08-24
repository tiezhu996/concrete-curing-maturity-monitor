import { describe, expect, it } from 'vitest'
import { cumulativeMaturity } from './chart-data'

describe('cumulativeMaturity', () => {
  it('uses trapezoidal Nurse-Saul accumulation', () => {
    expect(cumulativeMaturity([
      { timestamp: '2026-08-22T00:00:00Z', temperature_c: 10 },
      { timestamp: '2026-08-22T01:00:00Z', temperature_c: 20 },
    ], 0)).toEqual([
      { timestamp: '2026-08-22T00:00:00Z', value: 0 },
      { timestamp: '2026-08-22T01:00:00Z', value: 15 },
    ])
  })

  it('clamps contributions below datum', () => {
    expect(cumulativeMaturity([
      { timestamp: '2026-08-22T00:00:00Z', temperature_c: -4 },
      { timestamp: '2026-08-22T02:00:00Z', temperature_c: -2 },
    ], 0)[1].value).toBe(0)
  })
})
