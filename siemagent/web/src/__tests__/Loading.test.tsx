import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BootSplash, LoadingState } from '../components/Loading'

describe('LoadingState', () => {
  it('announces what is loading', () => {
    render(<LoadingState label="Loading rules…" />)
    expect(screen.getByRole('status')).toHaveTextContent('Loading rules…')
  })

  it('defaults to a generic label', () => {
    render(<LoadingState compact />)
    expect(screen.getByRole('status')).toHaveTextContent('Loading…')
  })
})

describe('BootSplash', () => {
  it('announces the session check', () => {
    render(<BootSplash />)
    expect(screen.getByRole('status')).toHaveTextContent('Checking your session…')
  })
})
