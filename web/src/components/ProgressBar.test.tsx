import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ProgressBar } from './ProgressBar'

describe('ProgressBar', () => {
  it('renders with basic value', () => {
    const { container } = render(<ProgressBar value={50} />)
    expect(container.querySelector('.progress-bar')).toBeInTheDocument()
  })

  it('displays value percentage', () => {
    render(<ProgressBar value={50} showValue={true} />)
    expect(screen.getByText('50.0%')).toBeInTheDocument()
  })

  it('renders with label', () => {
    render(<ProgressBar value={75} label="CPU Usage" />)
    expect(screen.getByText('CPU Usage')).toBeInTheDocument()
  })

  it('renders with sublabel', () => {
    render(<ProgressBar value={60} label="Memory" sublabel="8GB / 16GB" />)
    expect(screen.getByText('Memory')).toBeInTheDocument()
    expect(screen.getByText('8GB / 16GB')).toBeInTheDocument()
  })

  it('renders with 0% value', () => {
    render(<ProgressBar value={0} />)
    expect(screen.getByText('0.0%')).toBeInTheDocument()
  })

  it('renders with 100% value', () => {
    render(<ProgressBar value={100} />)
    expect(screen.getByText('100.0%')).toBeInTheDocument()
  })

  it('caps width at 100%', () => {
    const { container } = render(<ProgressBar value={150} />)
    const fill = container.querySelector('.progress-bar-fill')
    expect(fill).toHaveStyle({ width: '100%' })
  })

  it('hides value when showValue is false', () => {
    render(<ProgressBar value={50} showValue={false} />)
    expect(screen.queryByText('50.0%')).not.toBeInTheDocument()
  })

  it('uses green color for low values', () => {
    const { container } = render(<ProgressBar value={30} />)
    const fill = container.querySelector('.progress-bar-fill')
    expect(fill).toBeInTheDocument()
    // Check that green gradient is applied
    const style = fill?.getAttribute('style')
    expect(style).toContain('#10b981')
  })

  it('uses yellow color for medium values', () => {
    const { container } = render(<ProgressBar value={60} />)
    const fill = container.querySelector('.progress-bar-fill')
    const style = fill?.getAttribute('style')
    expect(style).toContain('#f59e0b')
  })

  it('uses red color for high values', () => {
    const { container } = render(<ProgressBar value={85} />)
    const fill = container.querySelector('.progress-bar-fill')
    const style = fill?.getAttribute('style')
    expect(style).toContain('#ef4444')
  })
})
