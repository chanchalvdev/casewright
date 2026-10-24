import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrandMark, Wordmark } from '../components/BrandMark'

describe('BrandMark', () => {
  it('is decorative unless titled', () => {
    const { container } = render(<BrandMark />)
    expect(container.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
    render(<BrandMark title="Casewright logo" />)
    expect(screen.getByRole('img', { name: 'Casewright logo' })).toBeInTheDocument()
  })

  it('gives each instance its own gradient ids', () => {
    const { container } = render(<><BrandMark /><BrandMark /></>)
    const ids = [...container.querySelectorAll('linearGradient')].map((g) => g.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('shows the product name and tagline in the wordmark', () => {
    render(<Wordmark tagline="AI security analyst" />)
    expect(screen.getByText('Casewright')).toBeInTheDocument()
    expect(screen.getByText('AI security analyst')).toBeInTheDocument()
  })
})
