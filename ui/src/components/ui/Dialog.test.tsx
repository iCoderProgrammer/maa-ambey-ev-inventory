import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
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

  it('returns focus to the button that opened it after Escape or Cancel', async () => {
    const user = userEvent.setup()
    function Harness() {
      const [open, setOpen] = useState(false)
      const close = () => {
        setOpen(false)
      }
      return (
        <>
          <button
            type="button"
            onClick={() => {
              setOpen(true)
            }}
          >
            Add supplier
          </button>
          <Dialog
            open={open}
            onClose={close}
            title="Add supplier"
            footer={
              <button type="button" onClick={close}>
                Cancel
              </button>
            }
          />
        </>
      )
    }
    render(<Harness />)
    const opener = screen.getByRole('button', { name: 'Add supplier' })

    await user.click(opener)
    expect(screen.getByRole('dialog')).toContainElement(document.activeElement as HTMLElement)
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()

    await user.click(opener)
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })
})
