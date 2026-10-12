import type { ButtonHTMLAttributes } from 'react'
import { buttonVariants, type ButtonSize, type ButtonVariant } from './buttonVariants'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
}

export function Button({
  variant = 'primary',
  size = 'md',
  type = 'button',
  className,
  ...props
}: ButtonProps) {
  return <button type={type} className={buttonVariants({ variant, size, className })} {...props} />
}
