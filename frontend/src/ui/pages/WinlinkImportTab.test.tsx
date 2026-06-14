import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/server'
import { noSession } from '../../test/handlers'
import WinlinkImportTab from './WinlinkImportTab'

vi.mock('../../adapters/sse/useStream', () => ({ useStream: vi.fn() }))

describe('WinlinkImportTab', () => {
  it('shows no-event alert when session has no event', async () => {
    server.use(http.get('/api/session', () => HttpResponse.json(noSession)))
    render(<WinlinkImportTab />)
    await waitFor(() =>
      expect(screen.getByText(/no active event/i)).toBeInTheDocument(),
    )
  })

  it('renders race and checkpoint selects when session is active', async () => {
    render(<WinlinkImportTab />)
    await waitFor(() =>
      expect(screen.getByRole('combobox', { name: /race/i })).toBeInTheDocument(),
    )
    expect(screen.getByRole('combobox', { name: /checkpoint/i })).toBeInTheDocument()
  })

  it('Import button is disabled when form is incomplete', async () => {
    render(<WinlinkImportTab />)
    await waitFor(() => screen.getByRole('button', { name: /import/i }))
    expect(screen.getByRole('button', { name: /import/i })).toBeDisabled()
  })

  it('shows import summary after successful import', async () => {
    const user = userEvent.setup()
    render(<WinlinkImportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await waitFor(() => screen.getByRole('combobox', { name: /checkpoint/i }))
    await user.click(screen.getByRole('combobox', { name: /checkpoint/i }))
    await waitFor(() => screen.getByRole('option', { name: /Aid Station 1/i }))
    await user.click(screen.getByRole('option', { name: /Aid Station 1/i }))

    await user.click(screen.getByLabelText(/paste winlink column/i))
    await user.type(screen.getByLabelText(/paste winlink column/i), '10:00')

    await user.click(screen.getByRole('button', { name: /import/i }))

    await waitFor(() => expect(screen.getByText(/import summary/i)).toBeInTheDocument())
    expect(screen.getByText(/created:/i)).toBeInTheDocument()
  })

  it('shows error alert on API failure', async () => {
    const user = userEvent.setup()
    server.use(
      http.post('/api/winlink/import', () =>
        HttpResponse.json({ error: 'no active checkpoint' }, { status: 400 }),
      ),
    )
    render(<WinlinkImportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await waitFor(() => screen.getByRole('combobox', { name: /checkpoint/i }))
    await user.click(screen.getByRole('combobox', { name: /checkpoint/i }))
    await waitFor(() => screen.getByRole('option', { name: /Aid Station 1/i }))
    await user.click(screen.getByRole('option', { name: /Aid Station 1/i }))

    await user.click(screen.getByLabelText(/paste winlink column/i))
    await user.type(screen.getByLabelText(/paste winlink column/i), 'bad')

    await user.click(screen.getByRole('button', { name: /import/i }))

    await waitFor(() =>
      expect(screen.getByText(/no active checkpoint/i)).toBeInTheDocument(),
    )
  })

  it('shows errors table when import returns errors', async () => {
    const user = userEvent.setup()
    server.use(
      http.post('/api/winlink/import', () =>
        HttpResponse.json({
          Created: 0,
          Updated: 0,
          Skipped: 0,
          SkippedDetails: [],
          Errors: ['bib 999 not in roster'],
        }),
      ),
    )
    render(<WinlinkImportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await waitFor(() => screen.getByRole('combobox', { name: /checkpoint/i }))
    await user.click(screen.getByRole('combobox', { name: /checkpoint/i }))
    await waitFor(() => screen.getByRole('option', { name: /Aid Station 1/i }))
    await user.click(screen.getByRole('option', { name: /Aid Station 1/i }))

    await user.type(screen.getByLabelText(/paste winlink column/i), '10:00')
    await user.click(screen.getByRole('button', { name: /import/i }))

    await waitFor(() => expect(screen.getByText(/errors:/i)).toBeInTheDocument())
    expect(screen.getByText(/bib 999 not in roster/i)).toBeInTheDocument()
  })

  it('shows raw reason when skip reason code is unknown', async () => {
    const user = userEvent.setup()
    server.use(
      http.post('/api/winlink/import', () =>
        HttpResponse.json({
          Created: 0,
          Updated: 0,
          Skipped: 1,
          SkippedDetails: [{ Position: 1, BibNumber: 0, Reason: 'custom_unknown_reason' }],
          Errors: [],
        }),
      ),
    )
    render(<WinlinkImportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await waitFor(() => screen.getByRole('combobox', { name: /checkpoint/i }))
    await user.click(screen.getByRole('combobox', { name: /checkpoint/i }))
    await waitFor(() => screen.getByRole('option', { name: /Aid Station 1/i }))
    await user.click(screen.getByRole('option', { name: /Aid Station 1/i }))

    await user.type(screen.getByLabelText(/paste winlink column/i), '10:00')
    await user.click(screen.getByRole('button', { name: /import/i }))

    await waitFor(() => expect(screen.getByText(/custom_unknown_reason/i)).toBeInTheDocument())
  })

  it('shows skipped details when import has skips', async () => {
    const user = userEvent.setup()
    server.use(
      http.post('/api/winlink/import', () =>
        HttpResponse.json({
          Created: 0,
          Updated: 0,
          Skipped: 1,
          SkippedDetails: [{ Position: 1, BibNumber: 0, Reason: 'blank' }],
          Errors: [],
        }),
      ),
    )
    render(<WinlinkImportTab />)

    await waitFor(() => screen.getByRole('combobox', { name: /race/i }))
    await user.click(screen.getByRole('combobox', { name: /race/i }))
    await waitFor(() => screen.getByRole('option', { name: /GDR/i }))
    await user.click(screen.getByRole('option', { name: /GDR/i }))

    await waitFor(() => screen.getByRole('combobox', { name: /checkpoint/i }))
    await user.click(screen.getByRole('combobox', { name: /checkpoint/i }))
    await waitFor(() => screen.getByRole('option', { name: /Aid Station 1/i }))
    await user.click(screen.getByRole('option', { name: /Aid Station 1/i }))

    await user.click(screen.getByLabelText(/paste winlink column/i))
    await user.type(screen.getByLabelText(/paste winlink column/i), '10:00')
    await user.click(screen.getByRole('button', { name: /import/i }))

    await waitFor(() => expect(screen.getByText(/skipped details/i)).toBeInTheDocument())
    expect(screen.getByText(/blank line/i)).toBeInTheDocument()
  })
})
