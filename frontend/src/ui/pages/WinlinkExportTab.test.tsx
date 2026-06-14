import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import { useStream } from '../../adapters/sse/useStream'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/server'
import { noSession } from '../../test/handlers'
import WinlinkExportTab from './WinlinkExportTab'

vi.mock('../../adapters/sse/useStream', () => ({ useStream: vi.fn() }))

const mockWriteText = vi.fn().mockResolvedValue(undefined)
beforeEach(() => {
  // Inject our spy into the persistent clipboard stub defined in setup.ts
  Object.assign(navigator.clipboard, { writeText: mockWriteText })
  mockWriteText.mockClear()
})
afterEach(() => {
  vi.mocked(useStream).mockReset()
})

describe('WinlinkExportTab', () => {
  it('shows no-event alert when session has no event', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json(noSession)))
    render(<WinlinkExportTab />)
    await waitFor(() => expect(screen.getByText(/no active event/i)).toBeInTheDocument())
  })

  it('renders race selector and Generate button when session is active', async () => {
    render(<WinlinkExportTab />)
    await waitFor(() => expect(screen.getByRole('combobox', { name: /race/i })).toBeInTheDocument())
    expect(screen.getByRole('button', { name: /generate/i })).toBeInTheDocument()
  })

  it('Generate button is disabled when no race selected', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json({ EventID: 1, Checkpoints: [] })))
    render(<WinlinkExportTab />)
    await waitFor(() => screen.getByRole('button', { name: /generate/i }))
    expect(screen.getByRole('button', { name: /generate/i })).toBeDisabled()
  })

  it('shows column text and subject after clicking Generate', async () => {
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(screen.getByRole('button', { name: /generate/i }))

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /copy column data/i })).toBeInTheDocument(),
    )
    expect(screen.getByRole('textbox', { name: /email subject/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /copy subject/i })).toBeInTheDocument()
  })

  it('subject field contains CP name, race name, time, and "update"', async () => {
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(screen.getByRole('button', { name: /generate/i }))

    await waitFor(() => screen.getByRole('textbox', { name: /email subject/i }))
    const subjectField = screen.getByRole('textbox', { name: /email subject/i }) as HTMLInputElement
    // mockCheckpoint.DisplayName = "Aid Station 1", mockRace.Name = "GDR"
    expect(subjectField.value).toMatch(/Aid Station 1/)
    expect(subjectField.value).toMatch(/GDR/)
    expect(subjectField.value).toMatch(/\d{2}:\d{2}/)
    expect(subjectField.value).toMatch(/update/)
  })

  it('copies subject to clipboard on Copy Subject click', async () => {
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(screen.getByRole('button', { name: /generate/i }))
    await waitFor(() => screen.getByRole('button', { name: /copy subject/i }))

    await user.click(screen.getByRole('button', { name: /copy subject/i }))

    await waitFor(() =>
      expect(mockWriteText).toHaveBeenCalledWith(expect.stringMatching(/update/i)),
    )
  })

  it('shows Copied! on Copy Subject button after click', async () => {
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(screen.getByRole('button', { name: /generate/i }))
    await waitFor(() => screen.getByRole('button', { name: /copy subject/i }))

    await user.click(screen.getByRole('button', { name: /copy subject/i }))

    await waitFor(() => expect(screen.getByText(/copied!/i)).toBeInTheDocument())
  })

  it('shows error alert when export API fails', async () => {
    server.use(
      http.get('/api/winlink/export/:raceID', () =>
        HttpResponse.json({ error: 'not found' }, { status: 404 }),
      ),
    )
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(screen.getByRole('button', { name: /generate/i }))

    await waitFor(() => expect(screen.getByText(/not found/i)).toBeInTheDocument())
  })

  it('copies column to clipboard on Copy button click', async () => {
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(screen.getByRole('button', { name: /generate/i }))
    await waitFor(() => screen.getByRole('button', { name: /copy column data/i }))

    await user.click(screen.getByRole('button', { name: /copy column data/i }))

    await waitFor(() => expect(mockWriteText).toHaveBeenCalledWith(expect.stringContaining('AS1')))
  })

  it('manually selecting race when no checkpoint fires onChange', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json({ EventID: 1, Checkpoints: [] })))
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    // No auto-selected race — raceID starts as ''
    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    expect(screen.getByRole('button', { name: /generate/i })).toBeDisabled()

    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    // onChange fired — Generate is now enabled
    await waitFor(() => expect(screen.getByRole('button', { name: /generate/i })).toBeEnabled())
  })

  it('handles SSE session changed event', async () => {
    let capturedCbs: Parameters<typeof useStream>[0] | null = null
    vi.mocked(useStream).mockImplementation((cbs) => {
      capturedCbs = cbs
    })
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))

    act(() => {
      capturedCbs?.onSessionChanged?.({ EventID: 1, Checkpoints: [] })
    })

    await waitFor(() => expect(screen.getByRole('combobox', { name: /race/i })).toBeInTheDocument())
  })

  it('shows Copied! feedback text after clicking copy', async () => {
    const user = userEvent.setup()
    render(<WinlinkExportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(screen.getByRole('button', { name: /generate/i }))
    await waitFor(() => screen.getByRole('button', { name: /copy column data/i }))

    await user.click(screen.getByRole('button', { name: /copy column data/i }))

    await waitFor(() => expect(screen.getByText(/copied!/i)).toBeInTheDocument())
  })
})
