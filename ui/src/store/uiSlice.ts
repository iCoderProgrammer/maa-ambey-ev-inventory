import { createSlice, type PayloadAction } from '@reduxjs/toolkit'

export type ThemePreference = 'light' | 'dark' | 'system'

export const THEME_STORAGE_KEY = 'maa-ambey.theme'

export interface UiState {
  theme: ThemePreference
  sidebarCollapsed: boolean
  mobileNavOpen: boolean
}

function readStoredTheme(): ThemePreference {
  try {
    const value = localStorage.getItem(THEME_STORAGE_KEY)
    if (value === 'light' || value === 'dark' || value === 'system') return value
  } catch {
    // Storage can be unavailable (private mode, blocked site data).
  }
  return 'system'
}

export function createInitialUiState(): UiState {
  return { theme: readStoredTheme(), sidebarCollapsed: false, mobileNavOpen: false }
}

/** Shared shell UI state. Server data belongs in TanStack Query, not here. */
const uiSlice = createSlice({
  name: 'ui',
  initialState: createInitialUiState,
  reducers: {
    setTheme(state, action: PayloadAction<ThemePreference>) {
      state.theme = action.payload
    },
    toggleSidebarCollapsed(state) {
      state.sidebarCollapsed = !state.sidebarCollapsed
    },
    setMobileNavOpen(state, action: PayloadAction<boolean>) {
      state.mobileNavOpen = action.payload
    },
  },
})

export const { setTheme, toggleSidebarCollapsed, setMobileNavOpen } = uiSlice.actions
export const uiReducer = uiSlice.reducer
