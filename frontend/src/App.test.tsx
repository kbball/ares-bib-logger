import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'

vi.mock('./adapters/sse/useStream', () => ({ useStream: vi.fn() }))

describe('App', () => {
  it('renders the app bar with title', () => {
    render(<App />)
    expect(screen.getByText('ARES Bib Logger')).toBeInTheDocument()
  })

  it('renders all five tab labels', () => {
    render(<App />)
    expect(screen.getByRole('tab', { name: /data entry/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /winlink import/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /winlink export/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /runners/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /admin/i })).toBeInTheDocument()
  })

  it('shows Data Entry tab content by default', async () => {
    render(<App />)
    await waitFor(() =>
      expect(screen.getByText(/data entry/i, { selector: 'h5' })).toBeInTheDocument(),
    )
  })

  it('switches to Winlink Import tab on click', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('tab', { name: /winlink import/i }))
    await waitFor(() =>
      expect(screen.getByText(/winlink import/i, { selector: 'h5' })).toBeInTheDocument(),
    )
  })

  it('switches to Winlink Export tab on click', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('tab', { name: /winlink export/i }))
    await waitFor(() =>
      expect(screen.getByText(/winlink export/i, { selector: 'h5' })).toBeInTheDocument(),
    )
  })

  it('switches to Runners tab on click', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('tab', { name: /^runners$/i }))
    await waitFor(() =>
      expect(screen.getByText(/^runners$/i, { selector: 'h5' })).toBeInTheDocument(),
    )
  })

  it('switches to Admin tab on click', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.click(screen.getByRole('tab', { name: /admin/i }))
    await waitFor(() =>
      expect(screen.getByText(/^admin$/i, { selector: 'h5' })).toBeInTheDocument(),
    )
  })

  it('toggles between light and dark mode', async () => {
    const user = userEvent.setup()
    render(<App />)

    // Default is light; toggle button shows "Switch to dark mode"
    const toggleBtn = screen.getByRole('button', { name: /switch to dark mode/i })
    expect(toggleBtn).toBeInTheDocument()

    await user.click(toggleBtn)

    // After toggle, button should offer switching back to light
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /switch to light mode/i })).toBeInTheDocument(),
    )
  })
})
