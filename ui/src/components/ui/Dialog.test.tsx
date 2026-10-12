import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Dialog } from './Dialog'

describe('Dialog', () => {
  it('is labelled by its title and closes via the close button', async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    render(
      <Dialog open onClose={onClose} title="Confirm action" description="This cannot be undone.">
        Body
      </Dialog>,
    )

    const dialog = screen.getByRole('dialog', { name: 'Confirm action' })
    expect(dialog).toHaveAccessibleDescription('This cannot be undone.')
    await user.click(screen.getByRole('button', { name: 'Close dialog' }))
    expect(onClose).toHaveBeenCalled()
  })

  it('is not rendered as open when closed', () => {
    render(<Dialog open={false} onClose={() => undefined} title="Hidden" />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})
