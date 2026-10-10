import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SetupGuide from './SetupGuide'
import type { SetupStep } from '../../application/setup'

const steps: SetupStep[] = [
  {
    id: 'event',
    title: 'Event',
    state: 'done',
    required: true,
    detail: 'Ridgeline is active.',
    target: 'setup-event',
  },
  {
    id: 'races',
    title: 'Races',
    state: 'todo',
    required: true,
    detail: 'Add a race.',
    target: 'setup-races',
  },
  {
    id: 'share',
    title: 'Share',
    state: 'optional',
    required: false,
    detail: 'Export.',
    target: 'setup-event',
  },
]

describe('SetupGuide', () => {
  it('shows progress and each step with its state', () => {
    render(<SetupGuide steps={steps} onJump={() => {}} />)
    expect(screen.getByText('1 of 2 done')).toBeInTheDocument()
    expect(screen.getByText('Done')).toBeInTheDocument()
    expect(screen.getByText('To do')).toBeInTheDocument()
    expect(screen.getByText('Optional')).toBeInTheDocument()
    expect(screen.getByText('Add a race.')).toBeInTheDocument()
  })

  it('jumps to the step target', async () => {
    const onJump = vi.fn()
    render(<SetupGuide steps={steps} onJump={onJump} />)
    await userEvent.click(screen.getByRole('button', { name: 'Go to Races' }))
    expect(onJump).toHaveBeenCalledWith('setup-races')
  })
})
