import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router-dom'
import Tour from './Tour'
import { TOUR_STEPS, openTour } from './tourState'

const KEY = 'ares-bib-logger:tour-seen'

function Where() {
  return <span data-testid="path">{useLocation().pathname}</span>
}
const setup = () =>
  render(
    <MemoryRouter initialEntries={['/data-entry']}>
      <Tour />
      <Where />
      <div id="setup-event">event</div>
      <div className="setup-guide">guide</div>
    </MemoryRouter>,
  )

describe('Tour', () => {
  it('stays closed once it has been seen', () => {
    setup()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('opens on first visit, remembers it, and can be declined', async () => {
    window.localStorage.removeItem(KEY)
    setup()
    expect(await screen.findByText('Welcome to ARES Bib Logger')).toBeInTheDocument()
    expect(window.localStorage.getItem(KEY)).toBe('1')
    await userEvent.click(screen.getByRole('button', { name: 'No thanks' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('walks the steps, goes to each page, and returns to where it started', async () => {
    setup()
    openTour()
    await userEvent.click(await screen.findByRole('button', { name: 'Take the tour' }))
    expect(await screen.findByText(`Step 1 of ${TOUR_STEPS.length}`)).toBeInTheDocument()
    await waitFor(() => expect(screen.getByTestId('path')).toHaveTextContent('/admin'))
    await waitFor(() => expect(screen.getByTestId('tour-spot')).toBeInTheDocument())

    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText(`Step 2 of ${TOUR_STEPS.length}`)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Back' }))
    expect(await screen.findByText(`Step 1 of ${TOUR_STEPS.length}`)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await waitFor(() => expect(screen.getByTestId('path')).toHaveTextContent('/data-entry'))
  })

  it('closes on Escape', async () => {
    setup()
    openTour()
    await screen.findByRole('dialog')
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('finishes on the last step', async () => {
    setup()
    openTour()
    await userEvent.click(await screen.findByRole('button', { name: 'Take the tour' }))
    for (let i = 0; i < TOUR_STEPS.length - 1; i++) {
      await userEvent.click(await screen.findByRole('button', { name: 'Next' }))
    }
    await userEvent.click(await screen.findByRole('button', { name: 'Done' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
