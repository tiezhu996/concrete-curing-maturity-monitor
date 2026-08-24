import { describe, expect, it } from 'vitest'
import { CONFIDENCE_LABELS, CONFIDENCE_LEVELS } from './confidence-level'
import { CURING_STATE_LABELS, CURING_STATES, CURING_TRANSITIONS } from './curing-state'

describe('shared domain enums', () => {
  it('covers every curing state with a label and legal transition table', () => {
    expect(CURING_STATES).toHaveLength(6)
    CURING_STATES.forEach((state) => {
      expect(CURING_STATE_LABELS[state]).toBeTruthy()
      expect(CURING_TRANSITIONS[state]).toBeDefined()
    })
  })

  it('covers every confidence level with a label', () => {
    expect(CONFIDENCE_LEVELS.map((level) => CONFIDENCE_LABELS[level])).toEqual(['低置信', '中置信', '高置信'])
  })
})
