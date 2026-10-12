import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { Provider } from 'react-redux'
import { useApplyTheme } from '@/hooks/useApplyTheme'
import type { AppStore } from '@/store'

function ThemeSync() {
  useApplyTheme()
  return null
}

export interface AppProvidersProps {
  store: AppStore
  queryClient: QueryClient
  children: ReactNode
}

export function AppProviders({ store, queryClient, children }: AppProvidersProps) {
  return (
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <ThemeSync />
        {children}
      </QueryClientProvider>
    </Provider>
  )
}
