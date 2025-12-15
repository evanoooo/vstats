import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ThemeProvider, useTheme, THEMES } from './ThemeContext'

// Test component that uses the theme context
const TestConsumer = () => {
  const { themeId, theme, isDark, setTheme, background, setBackground, themes } = useTheme()
  return (
    <div>
      <span data-testid="theme-id">{themeId}</span>
      <span data-testid="theme-name">{theme.name}</span>
      <span data-testid="is-dark">{isDark ? 'dark' : 'light'}</span>
      <span data-testid="themes-count">{themes.length}</span>
      <span data-testid="bg-type">{background.type}</span>
      <button onClick={() => setTheme('daylight')} data-testid="set-daylight">Set Daylight</button>
      <button onClick={() => setTheme('midnight')} data-testid="set-midnight">Set Midnight</button>
      <button onClick={() => setTheme('cyberpunk')} data-testid="set-cyberpunk">Set Cyberpunk</button>
      <button onClick={() => setBackground({ type: 'solid', solidColor: '#000' })} data-testid="set-bg">Set BG</button>
    </div>
  )
}

describe('ThemeContext', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    // Mock fetch to prevent network requests
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: false,
      status: 404,
      json: () => Promise.resolve({}),
    } as Response)
  })

  it('provides theme context', () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    )
    
    // Should render without errors
    expect(screen.getByTestId('theme-id')).toBeInTheDocument()
  })

  it('provides all available themes', () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    )
    
    const themesCount = parseInt(screen.getByTestId('themes-count').textContent || '0')
    expect(themesCount).toBe(THEMES.length)
    expect(themesCount).toBeGreaterThan(10) // We have many themes
  })

  it('sets theme to daylight', async () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    )
    
    fireEvent.click(screen.getByTestId('set-daylight'))
    
    await waitFor(() => {
      expect(screen.getByTestId('theme-id').textContent).toBe('daylight')
      expect(screen.getByTestId('is-dark').textContent).toBe('light')
    })
  })

  it('sets theme to midnight', async () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    )
    
    fireEvent.click(screen.getByTestId('set-midnight'))
    
    await waitFor(() => {
      expect(screen.getByTestId('theme-id').textContent).toBe('midnight')
      expect(screen.getByTestId('is-dark').textContent).toBe('dark')
    })
  })

  it('sets theme to cyberpunk', async () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    )
    
    fireEvent.click(screen.getByTestId('set-cyberpunk'))
    
    await waitFor(() => {
      expect(screen.getByTestId('theme-id').textContent).toBe('cyberpunk')
      expect(screen.getByTestId('is-dark').textContent).toBe('dark')
    })
  })

  it('updates background config', async () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    )
    
    fireEvent.click(screen.getByTestId('set-bg'))
    
    await waitFor(() => {
      expect(screen.getByTestId('bg-type').textContent).toBe('solid')
    })
  })

  it('changes theme and updates HTML class', async () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    )
    
    fireEvent.click(screen.getByTestId('set-cyberpunk'))
    
    await waitFor(() => {
      // Verify theme changed in context
      expect(screen.getByTestId('theme-id').textContent).toBe('cyberpunk')
      // Verify HTML class was updated
      expect(document.documentElement.classList.contains('theme-cyberpunk')).toBe(true)
    })
  })

  it('throws error when useTheme is used outside provider', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    
    expect(() => {
      render(<TestConsumer />)
    }).toThrow('useTheme must be used within ThemeProvider')
    
    consoleSpy.mockRestore()
  })
})

describe('THEMES constant', () => {
  it('contains required themes', () => {
    const themeIds = THEMES.map(t => t.id)
    expect(themeIds).toContain('midnight')
    expect(themeIds).toContain('daylight')
    expect(themeIds).toContain('cyberpunk')
    expect(themeIds).toContain('terminal')
  })

  it('all themes have required properties', () => {
    THEMES.forEach(theme => {
      expect(theme.id).toBeDefined()
      expect(theme.name).toBeDefined()
      expect(theme.nameZh).toBeDefined()
      expect(typeof theme.isDark).toBe('boolean')
      expect(theme.fonts).toBeDefined()
      expect(theme.fonts.heading).toBeDefined()
      expect(theme.fonts.body).toBeDefined()
      expect(theme.fonts.mono).toBeDefined()
      expect(theme.preview).toBeDefined()
    })
  })
})
