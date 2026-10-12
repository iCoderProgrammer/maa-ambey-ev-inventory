/** Error envelope returned by every failing API call (docs/API_CONVENTIONS.md). */
export interface ApiFieldError {
  field: string
  message: string
}

export interface ApiErrorBody {
  code: string
  message: string
  requestId?: string
  details?: ApiFieldError[]
}
