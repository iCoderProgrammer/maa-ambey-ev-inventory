import { zodResolver } from '@hookform/resolvers/zod'
import { AlertTriangle, Eye, EyeOff, Info, LogIn, WifiOff } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { Navigate, useLocation, useNavigate } from 'react-router'
import { z } from 'zod'
import { Button } from '@/components/ui/Button'
import { Field } from '@/components/ui/Field'
import { Input } from '@/components/ui/Input'
import { LoadingState } from '@/components/ui/StateViews'
import { ApiError, ClientErrorCode } from '@/lib/api/client'
import { AuthErrorCode } from './api/auth'
import { safeDestination } from './redirect'
import { useLogin, useSession } from './hooks/useSession'

// Client checks only catch empty input; the server decides everything else.
const loginSchema = z.object({
  username: z.string().trim().min(1, 'Enter your username.').max(100, 'Username is too long.'),
  password: z.string().min(1, 'Enter your password.').max(256, 'Password is too long.'),
})
type LoginValues = z.infer<typeof loginSchema>

function retryAfterText(seconds?: number): string {
  if (!seconds) return 'Wait a few minutes before trying again.'
  const minutes = Math.ceil(seconds / 60)
  return minutes <= 1
    ? 'Try again in about a minute.'
    : `Try again in about ${String(minutes)} minutes.`
}

function loginErrorMessage(error: unknown): { title: string; message: string; requestId?: string } {
  if (!(error instanceof ApiError)) {
    return { title: 'Sign-in failed', message: 'An unexpected error occurred. Please try again.' }
  }
  switch (error.code) {
    case AuthErrorCode.InvalidCredentials:
      return {
        title: 'Incorrect username or password',
        message: 'Check both and try again. Usernames are not case-sensitive; passwords are.',
      }
    case AuthErrorCode.AccountDisabled:
      return { title: 'Account disabled', message: error.message }
    case AuthErrorCode.RateLimited:
      return {
        title: 'Too many sign-in attempts',
        message: retryAfterText(error.retryAfterSeconds),
      }
    case ClientErrorCode.Network:
      return { title: 'Cannot reach the server', message: error.message }
    default:
      return { title: 'Sign-in failed', message: error.message, requestId: error.requestId }
  }
}

export function LoginPage() {
  const session = useSession()
  const loginMutation = useLogin()
  const navigate = useNavigate()
  const location = useLocation()
  const destination = safeDestination((location.state as { from?: unknown } | null)?.from)
  const [showPassword, setShowPassword] = useState(false)

  const {
    register,
    handleSubmit,
    setError,
    resetField,
    setFocus,
    formState: { errors },
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: '', password: '' },
  })

  useEffect(() => {
    document.title = 'Sign in · Maa Ambey Inventory'
  }, [])

  // Announced through role="alert"; focus stays in the form so the user can correct it.
  const failure = loginMutation.isError ? loginErrorMessage(loginMutation.error) : null

  if (session.data?.status === 'authenticated' && !loginMutation.isPending) {
    return <Navigate to={destination} replace />
  }

  const onSubmit = handleSubmit((values) => {
    if (loginMutation.isPending) return
    loginMutation.mutate(values, {
      onSuccess: () => {
        void navigate(destination, { replace: true })
      },
      onError: (error) => {
        if (error instanceof ApiError) {
          for (const d of error.details) {
            if (d.field === 'username' || d.field === 'password') {
              setError(d.field, { message: d.message })
            }
          }
          if (error.code === AuthErrorCode.InvalidCredentials) {
            resetField('password')
            setFocus('password')
          }
        }
      },
    })
  })

  const expired = session.data?.status === 'anonymous' && session.data.expired

  return (
    <div className="grid min-h-dvh place-items-center bg-canvas px-4 py-10">
      <main className="w-full max-w-sm" aria-labelledby="login-title">
        <div className="mb-6 flex flex-col items-center gap-3 text-center">
          <span
            className="grid size-11 place-items-center rounded-card bg-gradient-to-br from-primary to-accent text-sm font-bold tracking-tight text-white shadow-sm"
            aria-hidden="true"
          >
            MA
          </span>
          <div>
            <h1 id="login-title" className="text-2xl font-semibold tracking-tight text-ink">
              Sign in
            </h1>
            <p className="mt-1 text-sm text-muted">Maa Ambey Enterprises · EV Inventory</p>
          </div>
        </div>

        <div className="rounded-card border border-border bg-surface p-6 shadow-sm">
          {session.isPending && session.fetchStatus === 'fetching' ? (
            <LoadingState label="Checking your session…" />
          ) : (
            <form noValidate onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-4">
              {session.fetchStatus === 'paused' && !failure && (
                <p
                  role="status"
                  className="flex items-start gap-2.5 rounded-control border border-warning/30 bg-warning-subtle p-3 text-sm text-ink"
                >
                  <WifiOff className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
                  No network connection. You can sign in once you are back online.
                </p>
              )}

              {expired && !failure && (
                <p
                  role="status"
                  className="flex items-start gap-2.5 rounded-control border border-info/25 bg-info-subtle p-3 text-sm text-ink"
                >
                  <Info className="mt-0.5 size-4 shrink-0 text-info" aria-hidden="true" />
                  Your session has ended. Sign in again to continue.
                </p>
              )}

              {failure && (
                <div
                  role="alert"
                  className="flex items-start gap-2.5 rounded-control border border-danger/30 bg-danger-subtle p-3 text-sm"
                >
                  <AlertTriangle
                    className="mt-0.5 size-4 shrink-0 text-danger"
                    aria-hidden="true"
                  />
                  <div className="min-w-0">
                    <p className="font-semibold text-danger">{failure.title}</p>
                    <p className="mt-0.5 text-ink">{failure.message}</p>
                    {failure.requestId && (
                      <p className="mt-1 font-mono text-xs text-muted">
                        Reference: {failure.requestId}
                      </p>
                    )}
                  </div>
                </div>
              )}

              <Field label="Username" required error={errors.username?.message}>
                {(control) => (
                  <Input
                    {...control}
                    {...register('username')}
                    autoComplete="username"
                    autoCapitalize="none"
                    autoCorrect="off"
                    spellCheck={false}
                    autoFocus
                  />
                )}
              </Field>

              <Field label="Password" required error={errors.password?.message}>
                {(control) => (
                  <div className="relative">
                    <Input
                      {...control}
                      {...register('password')}
                      type={showPassword ? 'text' : 'password'}
                      autoComplete="current-password"
                      className="pr-11"
                    />
                    <button
                      type="button"
                      onClick={() => {
                        setShowPassword((v) => !v)
                      }}
                      aria-label={showPassword ? 'Hide password' : 'Show password'}
                      aria-pressed={showPassword}
                      className="absolute top-1/2 right-1 grid size-8 -translate-y-1/2 cursor-pointer place-items-center rounded-control text-muted transition-colors hover:bg-surface-muted hover:text-ink"
                    >
                      {showPassword ? (
                        <EyeOff className="size-4" aria-hidden="true" />
                      ) : (
                        <Eye className="size-4" aria-hidden="true" />
                      )}
                    </button>
                  </div>
                )}
              </Field>

              <Button type="submit" loading={loginMutation.isPending} className="mt-1 w-full">
                {!loginMutation.isPending && <LogIn className="size-4" aria-hidden="true" />}
                {loginMutation.isPending ? 'Signing in…' : 'Sign in'}
              </Button>
            </form>
          )}
        </div>

        <p className="mt-4 text-center text-xs text-muted">
          No account or forgotten password? Ask your system administrator.
        </p>
      </main>
    </div>
  )
}
