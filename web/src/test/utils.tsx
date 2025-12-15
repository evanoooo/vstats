import React, { type ReactElement } from 'react'
import { render, type RenderOptions, type RenderResult } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { ThemeProvider } from '../context/ThemeContext'
import { AuthProvider } from '../context/AuthContext'
import { WebSocketProvider } from '../context/WebSocketContext'
import { I18nextProvider } from 'react-i18next'
import i18n from '../i18n'
import { vi } from 'vitest'

// Create a mock i18n for testing
const mockI18n = i18n

// All providers wrapper
interface AllTheProvidersProps {
  children: React.ReactNode
}

const AllTheProviders = ({ children }: AllTheProvidersProps) => {
  return (
    <I18nextProvider i18n={mockI18n}>
      <ThemeProvider>
        <AuthProvider>
          <WebSocketProvider>
            <BrowserRouter>
              {children}
            </BrowserRouter>
          </WebSocketProvider>
        </AuthProvider>
      </ThemeProvider>
    </I18nextProvider>
  )
}

// Minimal providers wrapper (without WebSocket)
const MinimalProviders = ({ children }: AllTheProvidersProps) => {
  return (
    <I18nextProvider i18n={mockI18n}>
      <ThemeProvider>
        <BrowserRouter>
          {children}
        </BrowserRouter>
      </ThemeProvider>
    </I18nextProvider>
  )
}

// Custom render function with all providers
const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
): RenderResult => render(ui, { wrapper: AllTheProviders, ...options })

// Minimal render without WebSocket and Auth
const minimalRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
): RenderResult => render(ui, { wrapper: MinimalProviders, ...options })

// Re-export everything
export * from '@testing-library/react'
export { customRender as render, minimalRender }

// Test data factories
export const createMockServer = (overrides = {}) => ({
  server_id: 'test-server-1',
  server_name: 'Test Server',
  location: 'US',
  provider: 'AWS',
  tag: 'production',
  online: true,
  metrics: {
    timestamp: new Date().toISOString(),
    cpu: { usage: 45.5 },
    memory: { usage_percent: 60.0, total: 16000000000, used: 9600000000 },
    disks: [{ usage_percent: 70.0, total: 500000000000, used: 350000000000 }],
    network: { rx_speed: 1000000, tx_speed: 500000 },
    uptime: 86400,
    load_average: { one: 1.5, five: 1.2, fifteen: 1.0 },
    os: { name: 'Ubuntu', version: '22.04', arch: 'x86_64' },
  },
  ...overrides,
})

export const createMockHistoryData = (points = 100) => {
  const data = []
  const now = Date.now()
  for (let i = 0; i < points; i++) {
    data.push({
      timestamp: new Date(now - (points - i) * 60000).toISOString(),
      cpu: Math.random() * 100,
      memory: Math.random() * 100,
      disk: Math.random() * 100,
      net_rx: Math.floor(Math.random() * 10000000),
      net_tx: Math.floor(Math.random() * 5000000),
    })
  }
  return data
}

// Wait utilities
export const waitForLoadingToFinish = () => 
  new Promise(resolve => setTimeout(resolve, 0))

// Mock fetch helper
export const mockFetch = (data: unknown, status = 200) => {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
  })
}

