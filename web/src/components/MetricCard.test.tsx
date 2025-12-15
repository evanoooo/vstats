import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MetricCard } from './MetricCard'

describe('MetricCard', () => {
  const TestIcon = () => <span data-testid="test-icon">📊</span>

  it('renders with title and icon', () => {
    render(
      <MetricCard title="CPU Usage" icon={<TestIcon />}>
        <div>Content</div>
      </MetricCard>
    )
    expect(screen.getByText('CPU Usage')).toBeInTheDocument()
    expect(screen.getByTestId('test-icon')).toBeInTheDocument()
  })

  it('renders children content', () => {
    render(
      <MetricCard title="Memory" icon={<TestIcon />}>
        <span>75% used</span>
      </MetricCard>
    )
    expect(screen.getByText('75% used')).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(
      <MetricCard title="Disk" icon={<TestIcon />} className="custom-class">
        <div>Content</div>
      </MetricCard>
    )
    expect(container.firstChild).toHaveClass('custom-class')
  })

  it('applies animation delay', () => {
    const { container } = render(
      <MetricCard title="Network" icon={<TestIcon />} delay={200}>
        <div>Content</div>
      </MetricCard>
    )
    expect(container.firstChild).toHaveStyle({ animationDelay: '200ms' })
  })

  it('has glass-card styling', () => {
    const { container } = render(
      <MetricCard title="Load" icon={<TestIcon />}>
        <div>Content</div>
      </MetricCard>
    )
    expect(container.firstChild).toHaveClass('glass-card')
  })

  it('renders complex children', () => {
    render(
      <MetricCard title="Stats" icon={<TestIcon />}>
        <div>
          <span>Line 1</span>
          <span>Line 2</span>
        </div>
      </MetricCard>
    )
    expect(screen.getByText('Line 1')).toBeInTheDocument()
    expect(screen.getByText('Line 2')).toBeInTheDocument()
  })
})
