import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor, within, act } from '@testing-library/react'
import { useStream } from '../../adapters/sse/useStream'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/server'
import { noSession, mockEvent } from '../../test/handlers'
import AdminTab from './AdminTab'

vi.mock('../../adapters/sse/useStream', () => ({ useStream: vi.fn() }))

describe('AdminTab — Active Event', () => {
  it('renders the Active Event section', async () => {
    render(<AdminTab />)
    await waitFor(() =>
      expect(screen.getByText(/active event/i)).toBeInTheDocument(),
    )
  })

  it('shows event chip when session has an active event', async () => {
    render(<AdminTab />)
    await waitFor(() =>
      expect(screen.getByText(/event #1 active/i)).toBeInTheDocument(),
    )
  })

  it('allows creating a new event', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByLabelText(/new event name/i))
    await user.type(screen.getByLabelText(/new event name/i), 'Test Event')
    await user.click(screen.getByRole('button', { name: /create event/i }))

    await waitFor(() =>
      expect(screen.queryByRole('alert')).not.toBeInTheDocument(),
    )
  })

  it('shows error when creating event fails', async () => {
    server.use(
      http.post('/api/events', () =>
        HttpResponse.json({ error: 'name required' }, { status: 400 }),
      ),
    )
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByLabelText(/new event name/i))
    await user.type(screen.getByLabelText(/new event name/i), 'Bad')
    await user.click(screen.getByRole('button', { name: /create event/i }))

    await waitFor(() =>
      expect(screen.getByText(/name required/i)).toBeInTheDocument(),
    )
  })
})

describe('AdminTab — Races', () => {
  it('shows no-event info when session lacks an event', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json(noSession)))
    render(<AdminTab />)
    await waitFor(() => {
      const alerts = screen.getAllByText(/select an active event first/i)
      expect(alerts.length).toBeGreaterThan(0)
    })
  })

  it('renders race cards with race names', async () => {
    render(<AdminTab />)
    await waitFor(() => expect(screen.getByText('GDR')).toBeInTheDocument())
  })

  it('shows checkpoint rows inside each race card', async () => {
    render(<AdminTab />)
    await waitFor(() => expect(screen.getByText('AS1')).toBeInTheDocument())
    expect(screen.getByText('Aid Station 1')).toBeInTheDocument()
  })

  it('allows creating a new race', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByLabelText(/new race name/i))
    await user.type(screen.getByLabelText(/new race name/i), 'New Race')
    await user.click(screen.getByRole('button', { name: /create race/i }))

    await waitFor(() => expect(screen.queryByText(/name required/i)).not.toBeInTheDocument())
  })

  it('shows delete confirmation dialog when delete race icon is clicked', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /delete race and all its data/i }))
    await user.click(screen.getByRole('button', { name: /delete race and all its data/i }))

    await waitFor(() =>
      expect(screen.getByRole('dialog')).toBeInTheDocument(),
    )
    expect(screen.getByText(/confirm delete/i)).toBeInTheDocument()
  })

  it('closes delete dialog on Cancel', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /delete race and all its data/i }))
    await user.click(screen.getByRole('button', { name: /delete race and all its data/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.click(screen.getByRole('button', { name: /cancel/i }))
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    )
  })
})

describe('AdminTab — Roster Import', () => {
  it('renders the Roster Import section', async () => {
    render(<AdminTab />)
    await waitFor(() =>
      expect(screen.getByText(/roster import/i)).toBeInTheDocument(),
    )
  })

  it('Import Roster button is disabled when no race or TSV', async () => {
    render(<AdminTab />)
    await waitFor(() => screen.getByRole('button', { name: /import roster/i }))
    expect(screen.getByRole('button', { name: /import roster/i })).toBeDisabled()
  })
})

describe('AdminTab — Change Runner Status', () => {
  it('renders the Change Runner Status section', async () => {
    render(<AdminTab />)
    await waitFor(() =>
      expect(screen.getByText(/change runner status/i)).toBeInTheDocument(),
    )
  })

  it('shows no-event info when session lacks event', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json(noSession)))
    render(<AdminTab />)
    await waitFor(() => {
      const alerts = screen.getAllByText(/select an active event first/i)
      expect(alerts.length).toBeGreaterThan(0)
    })
  })

  it('Search button is disabled until race and bib are filled', async () => {
    render(<AdminTab />)
    await waitFor(() => screen.getByRole('button', { name: /search/i }))
    expect(screen.getByRole('button', { name: /search/i })).toBeDisabled()
  })

  it('searches for a runner and shows their status', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    // Use the data-testid we added to the status form
    await waitFor(() => screen.getByTestId('runner-status-form'))
    const form = screen.getByTestId('runner-status-form')

    // Find and interact with the race select inside the status form
    const raceCombobox = within(form).getByRole('combobox', { name: /race/i })
    await user.click(raceCombobox)
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.type(within(form).getByLabelText(/bib number/i), '100')
    await user.click(within(form).getByRole('button', { name: /search/i }))

    await waitFor(() =>
      expect(screen.getByText(/alice smith/i)).toBeInTheDocument(),
    )
    expect(screen.getByRole('button', { name: /set/i })).toBeInTheDocument()
  })

  it('shows error when bib not found', async () => {
    const user = userEvent.setup()
    server.use(
      http.get('/api/races/:raceID/runners', () => HttpResponse.json([])),
    )
    render(<AdminTab />)

    await waitFor(() => screen.getByTestId('runner-status-form'))
    const form = screen.getByTestId('runner-status-form')

    const raceCombobox = within(form).getByRole('combobox', { name: /race/i })
    await user.click(raceCombobox)
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.type(within(form).getByLabelText(/bib number/i), '999')
    await user.click(within(form).getByRole('button', { name: /search/i }))

    await waitFor(() =>
      expect(screen.getByText(/not found/i)).toBeInTheDocument(),
    )
  })

  it('shows error when searchRunner API fails', async () => {
    server.use(
      http.get('/api/races/:raceID/runners', () =>
        HttpResponse.json({ error: 'internal error' }, { status: 500 }),
      ),
    )
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByTestId('runner-status-form'))
    const form = screen.getByTestId('runner-status-form')

    await user.click(within(form).getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))
    await user.type(within(form).getByLabelText(/bib number/i), '100')
    await user.click(within(form).getByRole('button', { name: /search/i }))

    await waitFor(() => expect(screen.getByText(/internal error/i)).toBeInTheDocument())
  })

  it('shows error when applyRunnerStatus API fails', async () => {
    server.use(
      http.post('/api/log/status', () =>
        HttpResponse.json({ error: 'runner locked' }, { status: 400 }),
      ),
    )
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByTestId('runner-status-form'))
    const form = screen.getByTestId('runner-status-form')

    await user.click(within(form).getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))
    await user.type(within(form).getByLabelText(/bib number/i), '100')
    await user.click(within(form).getByRole('button', { name: /search/i }))

    await waitFor(() => screen.getByText(/alice smith/i))
    await user.click(screen.getByRole('button', { name: /^set$/i }))

    await waitFor(() => expect(screen.getByText(/runner locked/i)).toBeInTheDocument())
  })

  it('applies new runner status after searching', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByTestId('runner-status-form'))
    const form = screen.getByTestId('runner-status-form')

    await user.click(within(form).getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.type(within(form).getByLabelText(/bib number/i), '100')
    await user.click(within(form).getByRole('button', { name: /search/i }))

    await waitFor(() => screen.getByRole('combobox', { name: /new status/i }))
    await user.click(screen.getByRole('combobox', { name: /new status/i }))
    await waitFor(() => screen.getByRole('option', { name: /^dnf$/i }))
    await user.click(screen.getByRole('option', { name: /^dnf$/i }))

    await user.click(screen.getByRole('button', { name: /^set$/i }))

    await waitFor(() =>
      expect(screen.getByText(/status updated to DNF/i)).toBeInTheDocument(),
    )
  })
})

describe('AdminTab — Delete Confirmations', () => {
  it('confirms race delete when Delete button clicked in dialog', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /delete race and all its data/i }))
    await user.click(screen.getByRole('button', { name: /delete race and all its data/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /^delete$/i }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('opens checkpoint delete dialog and confirms', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getAllByRole('button', { name: /delete checkpoint/i })[0])
    await user.click(screen.getAllByRole('button', { name: /delete checkpoint/i })[0])

    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    expect(screen.getByText(/delete checkpoint/i)).toBeInTheDocument()

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /^delete$/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })
})

describe('AdminTab — Archive Event', () => {
  it('opens archive dialog and confirms', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /archive this event/i }))
    await user.click(screen.getByRole('button', { name: /archive this event/i }))

    await waitFor(() => screen.getByRole('dialog'))
    expect(screen.getByText(/archive event/i)).toBeInTheDocument()

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /^archive$/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('closes archive dialog on Cancel', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /archive this event/i }))
    await user.click(screen.getByRole('button', { name: /archive this event/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /cancel/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })
})

describe('AdminTab — Lock Order', () => {
  it('opens lock order dialog and confirms', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /lock checkpoint order/i }))
    await user.click(screen.getByRole('button', { name: /lock checkpoint order/i }))

    await waitFor(() => screen.getByRole('dialog'))
    expect(screen.getByText(/lock checkpoint order/i)).toBeInTheDocument()

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /lock order/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('closes lock order dialog on Cancel', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /lock checkpoint order/i }))
    await user.click(screen.getByRole('button', { name: /lock checkpoint order/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /cancel/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })
})

describe('AdminTab — Roster Import (with confirmation)', () => {
  it('shows confirm dialog and completes roster import', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByTestId('roster-section'))
    const roster = screen.getByTestId('roster-section')

    await user.click(within(roster).getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.type(within(roster).getByRole('textbox'), '100\tAlice\tSmith')

    await user.click(screen.getByRole('button', { name: /import roster/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /import & lock/i }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    await waitFor(() => expect(screen.getByText(/imported 2 runners/i)).toBeInTheDocument())
  })

  it('cancels roster import from confirm dialog', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByTestId('roster-section'))
    const roster = screen.getByTestId('roster-section')

    await user.click(within(roster).getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await user.type(within(roster).getByRole('textbox'), '100\tAlice\tSmith')
    await user.click(screen.getByRole('button', { name: /import roster/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: /cancel/i }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })
})

describe('AdminTab — Checkpoint Management', () => {
  it('creates a new checkpoint with distance', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByLabelText(/^code$/i))
    await user.type(screen.getByLabelText(/^code$/i), 'FIN')
    await user.type(screen.getByLabelText(/display name/i), 'Finish Line')
    await user.type(screen.getByLabelText(/dist \(mi\)/i), '50')

    await user.click(screen.getByRole('button', { name: /add checkpoint/i }))

    await waitFor(() => expect(screen.getByLabelText(/^code$/i)).toHaveValue(''))
  })

  it('edits a checkpoint inline, types new values, and saves', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getAllByRole('button', { name: /edit checkpoint code and name/i })[0])
    await user.click(screen.getAllByRole('button', { name: /edit checkpoint code and name/i })[0])

    // Type in each edit field to cover their onChange handlers
    await waitFor(() => screen.getByDisplayValue('AS1'))
    await user.type(screen.getByDisplayValue('AS1'), 'X')
    await user.type(screen.getByDisplayValue('Aid Station 1'), 'X')

    // For the number input, clear first so the value changes (typing into 10.5 appends → sanitized back to 10.5)
    const distInput = screen.getByDisplayValue('10.5')
    await user.clear(distInput)
    await user.type(distInput, '15')

    await user.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => expect(screen.queryByDisplayValue('AS1X')).not.toBeInTheDocument())
  })

  it('cancels checkpoint edit', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getAllByRole('button', { name: /edit checkpoint code and name/i })[0])
    await user.click(screen.getAllByRole('button', { name: /edit checkpoint code and name/i })[0])

    await waitFor(() => screen.getByDisplayValue('AS1'))

    await user.click(screen.getByRole('button', { name: /^cancel$/i }))

    await waitFor(() => expect(screen.queryByDisplayValue('AS1')).not.toBeInTheDocument())
  })

  it('moves a checkpoint down', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getAllByRole('button', { name: /move checkpoint down/i })[0])
    await user.click(screen.getAllByRole('button', { name: /move checkpoint down/i })[0])

    await waitFor(() => expect(screen.queryByText(/^error/i)).not.toBeInTheDocument())
  })

  it('moves a checkpoint up', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getAllByRole('button', { name: /move checkpoint up/i })[1])
    await user.click(screen.getAllByRole('button', { name: /move checkpoint up/i })[1])

    await waitFor(() => expect(screen.queryByText(/^error/i)).not.toBeInTheDocument())
  })

  it('changes active checkpoint for a race', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /active checkpoint/i }))
    await user.click(screen.getByRole('combobox', { name: /active checkpoint/i }))
    await waitFor(() => screen.getByRole('option', { name: /AS2/i }))
    await user.click(screen.getByRole('option', { name: /AS2/i }))

    await waitFor(() => expect(screen.queryByRole('alert', { name: /error/i })).not.toBeInTheDocument())
  })
})

describe('AdminTab — Dialog onClose (Escape key)', () => {
  it('closes archive dialog via Escape key', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /archive this event/i }))
    await user.click(screen.getByRole('button', { name: /archive this event/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('closes lock order dialog via Escape key', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /lock checkpoint order/i }))
    await user.click(screen.getByRole('button', { name: /lock checkpoint order/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('closes delete dialog via Escape key', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('button', { name: /delete race and all its data/i }))
    await user.click(screen.getByRole('button', { name: /delete race and all its data/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('closes roster import dialog via Escape key', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByTestId('roster-section'))
    const roster = screen.getByTestId('roster-section')

    await user.click(within(roster).getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))
    await user.type(within(roster).getByRole('textbox'), 'data')
    await user.click(screen.getByRole('button', { name: /import roster/i }))
    await waitFor(() => screen.getByRole('dialog'))

    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })
})

describe('AdminTab — SSE Callbacks', () => {
  it('handles SSE session changed event', async () => {
    let capturedCbs: Parameters<typeof useStream>[0] | null = null
    vi.mocked(useStream).mockImplementationOnce((cbs) => { capturedCbs = cbs })
    render(<AdminTab />)

    await waitFor(() => screen.getByText(/active event/i))

    act(() => {
      capturedCbs?.onSessionChanged?.({ EventID: 1, Checkpoints: [] })
    })

    await waitFor(() => expect(screen.getByText(/active event/i)).toBeInTheDocument())
  })
})

describe('AdminTab — Event Selection', () => {
  it('changes active event from dropdown', async () => {
    server.use(
      http.get('/api/events', () =>
        HttpResponse.json([mockEvent, { ...mockEvent, ID: 2, Name: 'Other Event' }]),
      ),
    )
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /^event$/i }))
    await user.click(screen.getByRole('combobox', { name: /^event$/i }))
    await waitFor(() => screen.getByRole('option', { name: /other event/i }))
    await user.click(screen.getByRole('option', { name: /other event/i }))

    await waitFor(() => expect(screen.queryByText(/^error:/i)).not.toBeInTheDocument())
  })

  it('clears active checkpoint when none is selected', async () => {
    const user = userEvent.setup()
    render(<AdminTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /active checkpoint/i }))
    await user.click(screen.getByRole('combobox', { name: /active checkpoint/i }))
    await waitFor(() => screen.getByRole('option', { name: /— none —/i }))
    await user.click(screen.getByRole('option', { name: /— none —/i }))

    await waitFor(() => expect(screen.queryByRole('alert', { name: /error/i })).not.toBeInTheDocument())
  })
})
