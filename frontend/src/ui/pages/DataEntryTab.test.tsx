import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor, within, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/server'
import { noSession, mockRunner, mockRunner2 } from '../../test/handlers'
import DataEntryTab from './DataEntryTab'
import { useStream } from '../../adapters/sse/useStream'

vi.mock('../../adapters/sse/useStream', () => ({ useStream: vi.fn() }))

afterEach(() => {
  vi.mocked(useStream).mockReset()
})

describe('DataEntryTab', () => {
  it('shows no-event alert when session has no event', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json(noSession)))
    render(<DataEntryTab />)
    await waitFor(() => expect(screen.getByText(/no active event/i)).toBeInTheDocument())
  })

  it('shows race stats cards when races are loaded', async () => {
    render(<DataEntryTab />)
    await waitFor(() => expect(screen.getByText('GDR')).toBeInTheDocument())
    expect(screen.getByText(/runners:/i)).toBeInTheDocument()
  })

  it('shows the Log Bib form', async () => {
    render(<DataEntryTab />)
    await waitFor(() => expect(screen.getByText(/log bib/i)).toBeInTheDocument())
    // Multiple Bib # inputs exist (Log, DNS/DNF, Transfer) — grab the first (Log Bib)
    expect(screen.getAllByLabelText(/bib #/i)[0]).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^log$/i })).toBeInTheDocument()
  })

  it('Log button is disabled when no active checkpoint', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json({ EventID: 1, Checkpoints: [] })))
    render(<DataEntryTab />)
    await waitFor(() => screen.getByRole('button', { name: /^log$/i }))
    expect(screen.getByRole('button', { name: /^log$/i })).toBeDisabled()
  })

  it('logs a bib and shows it in the recent log table', async () => {
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getAllByLabelText(/bib #/i)[0])
    const bibInput = screen.getAllByLabelText(/bib #/i)[0]
    await user.type(bibInput, '100')
    await waitFor(() => expect(screen.getByRole('button', { name: /^log$/i })).not.toBeDisabled())
    await user.click(screen.getByRole('button', { name: /^log$/i }))

    await waitFor(() => expect(screen.getByText(/alice/i)).toBeInTheDocument())
  })

  it('shows validation error for invalid bib', async () => {
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getAllByLabelText(/bib #/i)[0])
    await user.type(screen.getAllByLabelText(/bib #/i)[0], 'abc')
    await waitFor(() => expect(screen.getByRole('button', { name: /^log$/i })).not.toBeDisabled())
    await user.click(screen.getByRole('button', { name: /^log$/i }))

    await waitFor(() => expect(screen.getByText(/valid bib/i)).toBeInTheDocument())
  })

  it('shows duplicate alert when bib is already logged', async () => {
    server.use(
      http.post('/api/log/bib', () =>
        HttpResponse.json({ runner: mockRunner, log: null, is_duplicate: true }),
      ),
    )
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getAllByLabelText(/bib #/i)[0])
    await user.type(screen.getAllByLabelText(/bib #/i)[0], '100')
    await waitFor(() => expect(screen.getByRole('button', { name: /^log$/i })).not.toBeDisabled())
    await user.click(screen.getByRole('button', { name: /^log$/i }))

    await waitFor(() => expect(screen.getAllByText(/duplicate/i).length).toBeGreaterThan(0))
  })

  it('shows DNS/DNF form and submits status', async () => {
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/dns \/ dnf/i))
    const statusSection = screen.getByText(/dns \/ dnf/i).closest('div')!
    const bibInputs = within(statusSection).getAllByLabelText(/bib #/i)
    const statusBibInput = bibInputs[0]

    await user.type(statusBibInput, '100')
    await user.click(within(statusSection).getByRole('button', { name: /submit/i }))

    await waitFor(() => expect(screen.getByText(/bib 100 marked dns/i)).toBeInTheDocument())
  })

  it('shows Transfer Runner form', async () => {
    render(<DataEntryTab />)
    await waitFor(() => expect(screen.getByText(/transfer runner/i)).toBeInTheDocument())
    expect(screen.getByRole('button', { name: /transfer/i })).toBeInTheDocument()
  })

  it('shows "No entries yet" when recent log is empty', async () => {
    render(<DataEntryTab />)
    await waitFor(() => expect(screen.getByText(/no entries yet/i)).toBeInTheDocument())
  })

  it('shows DNS/DNF count without checkpoint when no active checkpoint set', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json({ EventID: 1, Checkpoints: [] })))
    render(<DataEntryTab />)
    // Race stats cards appear even without active CP — DNS/DNF uses fallback count
    await waitFor(() => expect(screen.getByText(/runners:/i)).toBeInTheDocument())
    expect(screen.getByText(/dns\/dnf:/i)).toBeInTheDocument()
  })

  it('changes DNS/DNF status select to DNF', async () => {
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/dns \/ dnf/i))
    const statusSection = screen.getByText(/dns \/ dnf/i).closest('div')!

    const statusSelect = within(statusSection).getByRole('combobox')
    await user.click(statusSelect)
    await waitFor(() => screen.getByRole('option', { name: /^dnf$/i }))
    await user.click(screen.getByRole('option', { name: /^dnf$/i }))

    // submit with DNF
    await user.type(within(statusSection).getAllByLabelText(/bib #/i)[0], '100')
    await user.click(within(statusSection).getByRole('button', { name: /submit/i }))

    await waitFor(() => expect(screen.getByText(/bib 100 marked dnf/i)).toBeInTheDocument())
  })

  it('shows error alert when logBib API fails', async () => {
    server.use(
      http.post('/api/log/bib', () =>
        HttpResponse.json({ error: 'bib not found' }, { status: 404 }),
      ),
    )
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getAllByLabelText(/bib #/i)[0])
    await user.type(screen.getAllByLabelText(/bib #/i)[0], '999')
    await waitFor(() => expect(screen.getByRole('button', { name: /^log$/i })).not.toBeDisabled())
    await user.click(screen.getByRole('button', { name: /^log$/i }))

    await waitFor(() => expect(screen.getByText(/bib not found/i)).toBeInTheDocument())
  })

  it('transfers a runner to another race', async () => {
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/^transfer runner$/i))
    const section = screen.getByText(/^transfer runner$/i).closest('div')!

    await user.type(within(section).getByLabelText(/bib #/i), '100')

    await user.click(within(section).getByRole('combobox'))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(within(section).getByRole('button', { name: /transfer/i }))

    await waitFor(() => expect(screen.getByText(/bib 100 transferred/i)).toBeInTheDocument())
  })

  it('closes duplicate alert when X is clicked', async () => {
    server.use(
      http.post('/api/log/bib', () =>
        HttpResponse.json({ runner: mockRunner, log: null, is_duplicate: true }),
      ),
    )
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getAllByLabelText(/bib #/i)[0])
    await user.type(screen.getAllByLabelText(/bib #/i)[0], '100')
    await waitFor(() => expect(screen.getByRole('button', { name: /^log$/i })).not.toBeDisabled())
    await user.click(screen.getByRole('button', { name: /^log$/i }))

    // Alert text is "DUPLICATE: Bib 100 (...)" — distinct from the chip label "DUPLICATE"
    await waitFor(() => expect(screen.getByText(/DUPLICATE: Bib/)).toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: /close/i }))

    await waitFor(() => expect(screen.queryByText(/DUPLICATE: Bib/)).not.toBeInTheDocument())
  })

  it('counts DNS runner in DNS/DNF fallback when no active checkpoint', async () => {
    server.use(
      http.get('/api/session', () => HttpResponse.json({ EventID: 1, Checkpoints: [] })),
      http.get('/api/races/:raceID/runners', () =>
        HttpResponse.json([mockRunner, { ...mockRunner2, Status: 'DNS' }]),
      ),
    )
    render(<DataEntryTab />)
    await waitFor(() => expect(screen.getByText(/runners:/i)).toBeInTheDocument())
    expect(screen.getByText(/dns\/dnf:/i)).toBeInTheDocument()
  })

  it('shows projected next arrival when runner pace can be computed', async () => {
    server.use(
      http.get('/api/session', () =>
        HttpResponse.json({ EventID: 1, Checkpoints: [{ RaceID: 1, CheckpointID: 3 }] }),
      ),
      http.get('/api/races/:raceID/checkpoints', () =>
        HttpResponse.json([
          {
            ID: 1,
            RaceID: 1,
            Code: 'CP1',
            DisplayName: 'CP 1',
            DisplayOrder: 1,
            DistanceFromStart: 5.0,
            CreatedAt: '',
          },
          {
            ID: 2,
            RaceID: 1,
            Code: 'CP2',
            DisplayName: 'CP 2',
            DisplayOrder: 2,
            DistanceFromStart: 15.0,
            CreatedAt: '',
          },
          {
            ID: 3,
            RaceID: 1,
            Code: 'CP3',
            DisplayName: 'CP 3',
            DisplayOrder: 3,
            DistanceFromStart: 25.0,
            CreatedAt: '',
          },
        ]),
      ),
      http.get('/api/races/:raceID/logs', () =>
        HttpResponse.json([
          {
            ID: 1,
            RunnerID: 1,
            CheckpointID: 1,
            RecordedAt: '2026-06-14T10:00:00Z',
            Source: 'MANUAL',
            RawMessage: '10:00',
            CreatedAt: '',
          },
          {
            ID: 2,
            RunnerID: 1,
            CheckpointID: 2,
            RecordedAt: '2026-06-14T11:00:00Z',
            Source: 'MANUAL',
            RawMessage: '11:00',
            CreatedAt: '',
          },
        ]),
      ),
    )
    render(<DataEntryTab />)
    await waitFor(() => expect(screen.getByText(/next expected:/i)).toBeInTheDocument())
  })

  it('shows error when submitStatus API fails', async () => {
    server.use(
      http.post('/api/log/status', () =>
        HttpResponse.json({ error: 'status update failed' }, { status: 500 }),
      ),
    )
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/dns \/ dnf/i))
    const statusSection = screen.getByText(/dns \/ dnf/i).closest('div')!

    await user.type(within(statusSection).getAllByLabelText(/bib #/i)[0], '100')
    await user.click(within(statusSection).getByRole('button', { name: /submit/i }))

    await waitFor(() => expect(screen.getByText(/status update failed/i)).toBeInTheDocument())
  })

  it('shows error when submitTransfer API fails', async () => {
    server.use(
      http.post('/api/runners/transfer', () =>
        HttpResponse.json({ error: 'transfer failed' }, { status: 500 }),
      ),
    )
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/^transfer runner$/i))
    const section = screen.getByText(/^transfer runner$/i).closest('div')!

    await user.type(within(section).getByLabelText(/bib #/i), '100')
    await user.click(within(section).getByRole('combobox'))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))
    await user.click(within(section).getByRole('button', { name: /transfer/i }))

    await waitFor(() => expect(screen.getByText(/transfer failed/i)).toBeInTheDocument())
  })

  it('handles SSE bib logged callback', async () => {
    let capturedCbs: Parameters<typeof useStream>[0] | null = null
    vi.mocked(useStream).mockImplementation((cbs) => {
      capturedCbs = cbs
    })
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/log bib/i))

    act(() => {
      capturedCbs?.onBibLogged?.({ runner: mockRunner, log: null, is_duplicate: false })
    })

    await waitFor(() => expect(screen.getByText(/alice smith/i)).toBeInTheDocument())
  })

  it('handles SSE session changed callback', async () => {
    let capturedCbs: Parameters<typeof useStream>[0] | null = null
    vi.mocked(useStream).mockImplementation((cbs) => {
      capturedCbs = cbs
    })
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/log bib/i))

    act(() => {
      capturedCbs?.onSessionChanged?.({ EventID: 1, Checkpoints: [] })
    })

    await waitFor(() => expect(screen.getByRole('button', { name: /^log$/i })).toBeInTheDocument())
  })

  it('shows bib not found when transfer runner does not exist in roster', async () => {
    const user = userEvent.setup()
    render(<DataEntryTab />)

    await waitFor(() => screen.getByText(/^transfer runner$/i))
    const section = screen.getByText(/^transfer runner$/i).closest('div')!

    await user.type(within(section).getByLabelText(/bib #/i), '999')

    await user.click(within(section).getByRole('combobox'))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.click(within(section).getByRole('button', { name: /transfer/i }))

    await waitFor(() => expect(screen.getByText(/bib 999 not found/i)).toBeInTheDocument())
  })
})
