import { zodResolver } from '@hookform/resolvers/zod'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'
import { z } from 'zod'
import { Button } from './Button'
import { Field } from './Field'
import { Input } from './Input'
import { Select } from './Select'

const schema = z.object({
  name: z.string().trim().min(1, 'Name is required'),
  status: z.enum(['PENDING_VERIFICATION', 'ARCHIVED'], { message: 'Choose a status' }),
})
type Values = z.infer<typeof schema>

function SampleForm({ onSubmit }: { onSubmit: (v: Values) => void }) {
  const { register, handleSubmit, formState } = useForm<Values>({ resolver: zodResolver(schema) })
  return (
    <form onSubmit={(e) => void handleSubmit(onSubmit)(e)} noValidate>
      <Field label="Name" required error={formState.errors.name?.message}>
        {(p) => <Input {...p} {...register('name')} />}
      </Field>
      <Field
        label="Status"
        hint="New records start pending."
        error={formState.errors.status?.message}
      >
        {(p) => (
          <Select
            {...p}
            {...register('status')}
            placeholder="Select…"
            options={[
              { value: 'PENDING_VERIFICATION', label: 'Pending verification' },
              { value: 'ARCHIVED', label: 'Archived' },
            ]}
          />
        )}
      </Field>
      <Button type="submit">Save</Button>
    </form>
  )
}

describe('Field + Input + Select with React Hook Form and Zod', () => {
  it('associates labels, hints and validation errors', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()
    render(<SampleForm onSubmit={onSubmit} />)

    await user.click(screen.getByRole('button', { name: 'Save' }))

    const name = screen.getByLabelText(/Name/)
    expect(name).toHaveAttribute('aria-invalid', 'true')
    expect(name).toHaveAccessibleDescription('Name is required')
    expect(screen.getByLabelText('Status')).toHaveAccessibleDescription(
      'New records start pending. Choose a status',
    )
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('submits valid values', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()
    render(<SampleForm onSubmit={onSubmit} />)

    await user.type(screen.getByLabelText(/Name/), 'Example')
    await user.selectOptions(screen.getByLabelText('Status'), 'ARCHIVED')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(onSubmit).toHaveBeenCalledWith(
      { name: 'Example', status: 'ARCHIVED' },
      expect.anything(),
    )
  })
})
