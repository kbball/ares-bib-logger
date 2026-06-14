import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/server'
import { noSession } from '../../test/handlers'
import RunnersTab from './RunnersTab'
import { useStream } from '../../adapters/sse/useStream'

vi.mock('../../adapters/sse/useStream', () => ({ useStream: vi.fn() }))

describe('RunnersTab', () => {
  it('shows no-event alert when session has no event', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json(noSession)))
    render(<RunnersTab />)
    await waitFor(() =>
      expect(screen.getByText(/no active event/i)).toBeInTheDocument(),
    )
  })

  it('renders runner rows when data loads', async () => {
    render(<RunnersTab />)
    // Runners render as "FirstName LastName" — match with regex
    await waitFor(() => expect(screen.getByText(/alice smith/i)).toBeInTheDocument())
    expect(screen.getByText(/bob jones/i)).toBeInTheDocument()
  })

  it('shows bib numbers in the table', async () => {
    render(<RunnersTab />)
    await waitFor(() => expect(screen.getByText('100')).toBeInTheDocument())
    expect(screen.getByText('101')).toBeInTheDocument()
  })

  it('shows All tab and race-specific tab', async () => {
    render(<RunnersTab />)
    // Both tabs load after races fetch — await them together
    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /all/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /GDR/i })).toBeInTheDocument()
    })
  })

  it('filters runners by name search', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.type(screen.getByLabelText(/search bib \/ name/i), 'Alice')

    await waitFor(() => expect(screen.getByText(/alice smith/i)).toBeInTheDocument())
    expect(screen.queryByText(/bob jones/i)).not.toBeInTheDocument()
  })

  it('filters runners by bib number search', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.type(screen.getByLabelText(/search bib \/ name/i), '101')

    await waitFor(() => expect(screen.queryByText(/alice smith/i)).not.toBeInTheDocument())
    expect(screen.getByText(/bob jones/i)).toBeInTheDocument()
  })

  it('clicking a row opens the detail modal', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.click(screen.getByText(/alice smith/i).closest('tr')!)

    await waitFor(() =>
      expect(screen.getByRole('dialog')).toBeInTheDocument(),
    )
    expect(screen.getAllByText(/alice smith/i).length).toBeGreaterThan(0)
  })

  it('modal shows bib, race, and status', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.click(screen.getByText(/alice smith/i).closest('tr')!)

    await waitFor(() => screen.getByRole('dialog'))
    expect(screen.getByText(/bib:/i)).toBeInTheDocument()
    expect(screen.getByText(/race:/i)).toBeInTheDocument()
    expect(screen.getByText(/status:/i)).toBeInTheDocument()
  })

  it('modal shows checkpoint log table', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.click(screen.getByText(/alice smith/i).closest('tr')!)

    await waitFor(() => screen.getByRole('dialog'))
    await waitFor(() =>
      expect(screen.getByText(/checkpoint log/i)).toBeInTheDocument(),
    )
    // "AS1 – Aid Station 1" appears in the log table cell
    expect(screen.getAllByText(/aid station 1/i).length).toBeGreaterThan(0)
  })

  it('modal shows projected arrival at active checkpoint', async () => {
    const user = userEvent.setup()

    server.use(
      http.get('/api/races/:raceID/logs', () =>
        HttpResponse.json([
          {
            ID: 1, RunnerID: 1, CheckpointID: 1,
            RecordedAt: '2026-06-14T10:00:00Z', Source: 'MANUAL', RawMessage: '10:00', CreatedAt: '',
          },
          {
            ID: 2, RunnerID: 1, CheckpointID: 2,
            RecordedAt: '2026-06-14T11:00:00Z', Source: 'MANUAL', RawMessage: '11:00', CreatedAt: '',
          },
        ]),
      ),
      http.get('/api/session', () =>
        HttpResponse.json({ EventID: 1, Checkpoints: [{ RaceID: 1, CheckpointID: 1 }] }),
      ),
    )

    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.click(screen.getByText(/alice smith/i).closest('tr')!)

    await waitFor(() => screen.getByRole('dialog'))
    await waitFor(() =>
      expect(screen.getByText(/proj\. arrival at/i)).toBeInTheDocument(),
    )
  })

  it('closing modal by Escape dismisses it', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.click(screen.getByText(/alice smith/i).closest('tr')!)

    await waitFor(() => screen.getByRole('dialog'))

    await user.keyboard('{Escape}')
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    )
  })

  it('sorts runners by name when Name column header is clicked', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))

    // Initial sort is by BibNumber asc — Alice (100) before Bob (101)
    let rows = screen.getAllByRole('row')
    const aliceIdxBefore = rows.findIndex((r) => r.textContent?.includes('Alice Smith'))
    const bobIdxBefore = rows.findIndex((r) => r.textContent?.includes('Bob Jones'))
    expect(aliceIdxBefore).toBeLessThan(bobIdxBefore)

    // Click Name to sort by last+first name asc — Jones before Smith
    await user.click(screen.getByText('Name'))

    await waitFor(() => {
      rows = screen.getAllByRole('row')
      const aliceIdx = rows.findIndex((r) => r.textContent?.includes('Alice Smith'))
      const bobIdx = rows.findIndex((r) => r.textContent?.includes('Bob Jones'))
      expect(bobIdx).toBeLessThan(aliceIdx)
    })
  })

  it('sorts runners by status when Status column header is clicked twice', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByText(/alice smith/i))

    // Click Status once (asc: ACTIVE < UNKNOWN — Alice first)
    await user.click(screen.getByText('Status'))
    await waitFor(() => {
      const rows = screen.getAllByRole('row')
      const aliceIdx = rows.findIndex((r) => r.textContent?.includes('Alice Smith'))
      const bobIdx = rows.findIndex((r) => r.textContent?.includes('Bob Jones'))
      expect(aliceIdx).toBeLessThan(bobIdx)
    })

    // Click Status again (desc: UNKNOWN > ACTIVE — Bob first)
    await user.click(screen.getByText('Status'))
    await waitFor(() => {
      const rows = screen.getAllByRole('row')
      const aliceIdx = rows.findIndex((r) => r.textContent?.includes('Alice Smith'))
      const bobIdx = rows.findIndex((r) => r.textContent?.includes('Bob Jones'))
      expect(bobIdx).toBeLessThan(aliceIdx)
    })
  })

  it('filters runners by race tab and hides Race column', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => {
      expect(screen.getByRole('tab', { name: /all/i })).toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /GDR/i })).toBeInTheDocument()
    })

    // Click GDR tab — filterRaceID is set, Race column disappears
    await user.click(screen.getByRole('tab', { name: /GDR/i }))

    await waitFor(() =>
      expect(screen.queryByRole('columnheader', { name: /^race$/i })).not.toBeInTheDocument(),
    )
    expect(screen.getByText(/alice smith/i)).toBeInTheDocument()
  })

  it('switches back to All tab when search query has no matches in selected race tab', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByRole('tab', { name: /GDR/i }))
    await user.click(screen.getByRole('tab', { name: /GDR/i }))

    // Search for something that matches no runners → GDR tab disappears → filterRaceID resets to ''
    await user.type(screen.getByLabelText(/search bib \/ name/i), 'zzznomatch')

    // Race column reappears because filterRaceID is reset to ''
    await waitFor(() =>
      expect(screen.queryByRole('tab', { name: /GDR/i })).not.toBeInTheDocument(),
    )
  })

  it('handles SSE onSessionChanged callback', async () => {
    let capturedCbs: Parameters<typeof useStream>[0] | null = null
    vi.mocked(useStream).mockImplementation((cbs) => { capturedCbs = cbs })

    render(<RunnersTab />)
    await waitFor(() => screen.getByText(/alice smith/i))

    act(() => {
      capturedCbs?.onSessionChanged?.({ EventID: 1, Checkpoints: [{ RaceID: 1, CheckpointID: 1 }] })
    })

    await waitFor(() => expect(screen.getByText(/alice smith/i)).toBeInTheDocument())
  })

  it('handles SSE onBibLogged callback after races loaded (if-true branch)', async () => {
    let capturedCbs: Parameters<typeof useStream>[0] | null = null
    vi.mocked(useStream).mockImplementation((cbs) => { capturedCbs = cbs })

    render(<RunnersTab />)

    // Call onBibLogged immediately — races is still [] at first render (if-false branch)
    act(() => { capturedCbs?.onBibLogged?.({}) })

    // Wait for races to load so capturedCbs now has races.length > 0 in closure
    await waitFor(() => screen.getByText(/alice smith/i))

    // Call onBibLogged again — races is loaded now (if-true branch)
    act(() => { capturedCbs?.onBibLogged?.({}) })

    await waitFor(() => expect(screen.getByText(/alice smith/i)).toBeInTheDocument())
  })

  afterEach(() => { vi.mocked(useStream).mockReset() })

  it('shows Pace and Proj. Next columns when filtering by race with distances', async () => {
    const user = userEvent.setup()
    render(<RunnersTab />)

    await waitFor(() => screen.getByRole('tab', { name: /GDR/i }))
    await user.click(screen.getByRole('tab', { name: /GDR/i }))

    await waitFor(() =>
      expect(screen.getByRole('columnheader', { name: /^pace$/i })).toBeInTheDocument(),
    )
    // "Proj. Next" is inside a Tooltip span — check by text content
    expect(screen.getByText('Proj. Next')).toBeInTheDocument()
  })
})
