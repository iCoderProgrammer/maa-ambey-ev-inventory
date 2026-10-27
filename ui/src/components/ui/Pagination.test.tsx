import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Pagination } from './Pagination'

describe('Pagination', () => {
  it('shows the record range of a page', () => {
    render(
      <Pagination
        page={2}
        limit={25}
        total={43}
        onPage={vi.fn()}
        noun={['supplier', 'suppliers']}
      />,
    )
    expect(screen.getByText('26–43 of 43 suppliers')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled()
  })

  it('explains a page past the end and leads back to the last page', async () => {
    const onPage = vi.fn()
    const user = userEvent.setup()
    render(
      <Pagination
        page={99}
        limit={25}
        total={43}
        onPage={onPage}
        noun={['supplier', 'suppliers']}
      />,
    )
    expect(screen.getByText('No suppliers on this page (43 in total)')).toBeInTheDocument()
    expect(screen.queryByText(/2451/)).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Previous page' }))
    expect(onPage).toHaveBeenCalledWith(2)
  })
})
