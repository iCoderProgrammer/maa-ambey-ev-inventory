import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { resolveApiPath } from '@/lib/api/client'
import { renderApp, testMe } from '@/test/render'
import type { Me } from '@/features/access/api/access'
import { checkFile, formatBytes, type Attachment, type AttachmentLimits } from './api/attachments'

const limits: AttachmentLimits = {
  maxBytes: 1024,
  allowedTypes: [
    { contentType: 'image/jpeg', extensions: ['.jpg', '.jpeg'], previewable: true },
    { contentType: 'image/png', extensions: ['.png'], previewable: true },
    { contentType: 'application/pdf', extensions: ['.pdf'], previewable: false },
  ],
}

const pdf: Attachment = {
  id: 'a1',
  fileName: 'supplier-bill.pdf',
  contentType: 'application/pdf',
  sizeBytes: 2048,
  sha256: 'ab'.repeat(32),
  locationId: null,
  linked: false,
  uploadedBy: 'user-1',
  createdAt: '2026-10-13T09:30:00Z',
  previewable: false,
}
const photo: Attachment = {
  ...pdf,
  id: 'a2',
  fileName: 'pdi.png',
  contentType: 'image/png',
  previewable: true,
}

const both: Me = { ...testMe, permissions: ['attachments.upload', 'attachments.view'] }

function urlOf(input: RequestInfo | URL): string {
  return typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
}

function errorResponse(status: number, code: string, message: string) {
  return Response.json({ error: { code, message, requestId: 'req-42' } }, { status })
}

type Routes = Record<string, () => Response>

function mockFetch(routes: Routes) {
  return vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
    const url = urlOf(input)
    for (const [suffix, handler] of Object.entries(routes)) {
      if (url.includes(suffix)) return Promise.resolve(handler())
    }
    if (url.endsWith('/health')) {
      return Promise.resolve(
        Response.json({ data: { status: 'ok', service: 'api', version: 'v', time: 'x' } }),
      )
    }
    return Promise.resolve(
      Response.json({ data: { status: 'ready', checks: { database: 'up', migrations: 'up' } } }),
    )
  })
}

/** Minimal XMLHttpRequest stand-in: answers each upload with the next queued response. */
class FakeXHR {
  static responses: { status: number; body: unknown }[] = []
  static sent: FormData[] = []
  static headers: Record<string, string>[] = []
  status = 0
  responseText = ''
  withCredentials = false
  upload: { onprogress: ((e: ProgressEvent) => void) | null } = { onprogress: null }
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  onabort: (() => void) | null = null
  private reqHeaders: Record<string, string> = {}
  open() {}
  setRequestHeader(k: string, v: string) {
    this.reqHeaders[k] = v
  }
  getResponseHeader(name: string) {
    return name === 'X-Request-ID' ? 'req-42' : null
  }
  abort() {
    this.onabort?.()
  }
  send(form: FormData) {
    FakeXHR.sent.push(form)
    FakeXHR.headers.push(this.reqHeaders)
    const next = FakeXHR.responses.shift() ?? { status: 500, body: {} }
    setTimeout(() => {
      this.upload.onprogress?.({ lengthComputable: true, loaded: 5, total: 10 } as ProgressEvent)
      this.status = next.status
      this.responseText = JSON.stringify(next.body)
      this.onload?.()
    }, 0)
  }
}

beforeEach(() => {
  FakeXHR.responses = []
  FakeXHR.sent = []
  FakeXHR.headers = []
  vi.stubGlobal('XMLHttpRequest', FakeXHR)
})
afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function file(name: string, size: number, type: string) {
  return new File([new Uint8Array(size)], name, { type })
}

describe('client-side file checks', () => {
  it('mirror the server limits', () => {
    expect(checkFile(file('a.pdf', 10, 'application/pdf'), limits)).toBeNull()
    expect(checkFile(file('a.JPG', 10, 'image/jpeg'), limits)).toBeNull()
    expect(checkFile(file('a.pdf', 0, 'application/pdf'), limits)).toMatch(/empty/)
    expect(checkFile(file('a.pdf', 2048, 'application/pdf'), limits)).toMatch(/limit is 1.0 KB/)
    expect(checkFile(file('a.exe', 10, 'application/x-msdownload'), limits)).toMatch(/not accepted/)
    expect(checkFile(file('a.svg', 10, 'image/svg+xml'), limits)).toMatch(/not accepted/)
    expect(checkFile(file('a.png', 10, 'application/pdf'), limits)).toMatch(/does not match/)
    expect(formatBytes(10 * 1024 * 1024)).toBe('10 MB')
  })

  it('resolves download links against the API origin only', () => {
    expect(resolveApiPath('/api/v1/attachments/a1/content?x=1', '/api/v1')).toBe(
      '/api/v1/attachments/a1/content?x=1',
    )
    expect(resolveApiPath('/api/v1/x', 'https://api.example.com/api/v1')).toBe(
      'https://api.example.com/api/v1/x',
    )
  })
})

describe('documents navigation and access', () => {
  it('hides the link and denies the page without either permission, without calling the API', () => {
    const fetchMock = mockFetch({})
    renderApp('/documents')
    expect(screen.getByText('Access denied')).toBeInTheDocument()
    const nav = screen.getByRole('navigation', { name: 'Main' })
    expect(within(nav).queryByRole('link', { name: /Documents/ })).not.toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u]) => urlOf(u).includes('/attachments'))).toBe(false)
  })

  it('shows the link with either permission', () => {
    mockFetch({ '/attachments/limits': () => Response.json({ data: limits }) })
    renderApp('/', { me: { ...testMe, permissions: ['attachments.upload'] } })
    const nav = screen.getByRole('navigation', { name: 'Main' })
    expect(within(nav).getByRole('link', { name: /Documents/ })).toBeInTheDocument()
  })

  it('upload-only users get the uploader but no list', async () => {
    mockFetch({ '/attachments/limits': () => Response.json({ data: limits }) })
    renderApp('/documents', { me: { ...testMe, permissions: ['attachments.upload'] } })
    expect(await screen.findByText('Choose files to upload')).toBeInTheDocument()
    expect(screen.getByText(/You can upload files but not open them/)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'My uploads' })).not.toBeInTheDocument()
  })
})

describe('uploader', () => {
  async function openUploader() {
    const fetchMock = mockFetch({
      '/attachments/limits': () => Response.json({ data: limits }),
      '/attachments': () => Response.json({ data: [] }),
    })
    renderApp('/documents', { me: both })
    const input = await screen.findByLabelText('Choose files to upload')
    return { input, fetchMock }
  }

  it('rejects oversized and unsupported files before sending anything', async () => {
    const { input } = await openUploader()
    await userEvent.upload(input, [file('big.pdf', 4096, 'application/pdf')], {
      applyAccept: false,
    })
    await userEvent.upload(input, [file('script.html', 10, 'text/html')], { applyAccept: false })
    const alerts = await screen.findAllByRole('alert')
    expect(alerts.map((a) => a.textContent).join(' ')).toMatch(/limit is 1.0 KB/)
    expect(alerts.map((a) => a.textContent).join(' ')).toMatch(/not accepted/)
    expect(FakeXHR.sent).toHaveLength(0)
    expect(screen.queryByRole('button', { name: /Retry/ })).not.toBeInTheDocument()
  })

  it('shows success only after the server stores the file, and sends the CSRF token', async () => {
    const { input } = await openUploader()
    FakeXHR.responses.push({ status: 201, body: { data: pdf } })
    await userEvent.upload(input, file('supplier-bill.pdf', 10, 'application/pdf'))
    expect(await screen.findByText('Uploaded and stored privately.')).toBeInTheDocument()
    expect(FakeXHR.sent).toHaveLength(1)
    expect(FakeXHR.sent[0]?.get('file')).toBeInstanceOf(File)
    expect(FakeXHR.headers[0]?.['X-CSRF-Token']).toBe('test-csrf-token')
  })

  it('displays server rejections; final errors cannot be retried', async () => {
    const { input } = await openUploader()
    FakeXHR.responses.push({
      status: 415,
      body: {
        error: {
          code: 'UNSUPPORTED_MEDIA_TYPE',
          message: "The file's content does not match its type or file-name extension.",
          requestId: 'req-42',
        },
      },
    })
    await userEvent.upload(input, file('fake.pdf', 10, 'application/pdf'))
    const alert = await screen.findByText(/does not match its type/)
    expect(alert).toHaveTextContent('Reference: req-42')
    expect(screen.queryByRole('button', { name: /Retry upload/ })).not.toBeInTheDocument()
    expect(screen.queryByText('Uploaded and stored privately.')).not.toBeInTheDocument()
  })

  it('offers retry after a transient failure and succeeds on retry', async () => {
    const { input } = await openUploader()
    FakeXHR.responses.push(
      {
        status: 503,
        body: { error: { code: 'SERVICE_UNAVAILABLE', message: 'Try again shortly.' } },
      },
      { status: 201, body: { data: pdf } },
    )
    await userEvent.upload(input, file('supplier-bill.pdf', 10, 'application/pdf'))
    await userEvent.click(await screen.findByRole('button', { name: /Retry upload of/ }))
    expect(await screen.findByText('Uploaded and stored privately.')).toBeInTheDocument()
    expect(FakeXHR.sent).toHaveLength(2)
  })

  it('explains a permission refusal from the server', async () => {
    const { input } = await openUploader()
    FakeXHR.responses.push({
      status: 403,
      body: { error: { code: 'FORBIDDEN', message: 'You do not have permission to do this.' } },
    })
    await userEvent.upload(input, file('a.pdf', 10, 'application/pdf'))
    expect(await screen.findByText(/You do not have permission to do this/)).toBeInTheDocument()
  })
})

describe('my uploads', () => {
  it('shows an empty state', async () => {
    mockFetch({
      '/attachments/limits': () => Response.json({ data: limits }),
      '/attachments': () => Response.json({ data: [] }),
    })
    renderApp('/documents', { me: both })
    expect(await screen.findByText('No uploads yet')).toBeInTheDocument()
  })

  it('lists documents and reports a refused download', async () => {
    mockFetch({
      '/attachments/a1/link': () => errorResponse(403, 'FORBIDDEN', 'No.'),
      '/attachments/limits': () => Response.json({ data: limits }),
      '/attachments': () => Response.json({ data: [pdf, photo] }),
    })
    renderApp('/documents', { me: both })
    const list = await screen.findByRole('list', { name: 'My uploads' })
    expect(within(list).getByText('supplier-bill.pdf')).toBeInTheDocument()
    expect(within(list).getByText(/PDF document · 2.0 KB/)).toBeInTheDocument()
    // Only images offer a preview.
    expect(within(list).queryByRole('button', { name: 'Preview supplier-bill.pdf' })).toBeNull()
    expect(within(list).getByRole('button', { name: 'Preview pdi.png' })).toBeInTheDocument()

    await userEvent.click(within(list).getByRole('button', { name: 'Download supplier-bill.pdf' }))
    expect(
      await within(list).findByText('You no longer have permission to open this file.'),
    ).toBeInTheDocument()
  })

  it('downloads through a fresh link and never renders a storage URL', async () => {
    const createURL = vi.fn(() => 'blob:local-1')
    vi.stubGlobal(
      'URL',
      Object.assign(URL, { createObjectURL: createURL, revokeObjectURL: vi.fn() }),
    )
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const fetchMock = mockFetch({
      '/attachments/a1/content': () => new Response(new Blob(['%PDF-1.4'])),
      '/attachments/a1/link': () =>
        Response.json({
          data: {
            url: '/api/v1/attachments/a1/content?disposition=attachment&expires=1&signature=s',
            expiresAt: '2026-10-13T09:35:00Z',
            disposition: 'attachment',
          },
        }),
      '/attachments/limits': () => Response.json({ data: limits }),
      '/attachments': () => Response.json({ data: [pdf] }),
    })
    renderApp('/documents', { me: both })
    await userEvent.click(await screen.findByRole('button', { name: 'Download supplier-bill.pdf' }))
    await waitFor(() => {
      expect(click).toHaveBeenCalled()
    })
    expect(createURL).toHaveBeenCalled()
    const contentCall = fetchMock.mock.calls.find(([u]) => urlOf(u).includes('/content?'))
    expect(contentCall?.[1]).toMatchObject({ credentials: 'include' })
    expect(document.body.innerHTML).not.toContain('signature=')
  })

  it('previews an image from memory and handles an expired link', async () => {
    vi.stubGlobal(
      'URL',
      Object.assign(URL, { createObjectURL: () => 'blob:preview-1', revokeObjectURL: vi.fn() }),
    )
    let expired = true
    mockFetch({
      '/attachments/a2/content': () =>
        expired
          ? errorResponse(
              403,
              'DOWNLOAD_LINK_EXPIRED',
              'This download link has expired. Open the file again.',
            )
          : new Response(new Blob(['png'])),
      '/attachments/a2/link': () =>
        Response.json({
          data: {
            url: '/api/v1/attachments/a2/content?disposition=inline&expires=1&signature=s',
            expiresAt: '2026-10-13T09:35:00Z',
            disposition: 'inline',
          },
        }),
      '/attachments/limits': () => Response.json({ data: limits }),
      '/attachments': () => Response.json({ data: [photo] }),
    })
    renderApp('/documents', { me: both })
    await userEvent.click(await screen.findByRole('button', { name: 'Preview pdi.png' }))
    const dialog = await screen.findByRole('dialog', { name: 'pdi.png' })
    expect(await within(dialog).findByText(/download link has expired/)).toBeInTheDocument()
    expired = false
    await userEvent.click(within(dialog).getByRole('button', { name: 'Try again' }))
    const img = await within(dialog).findByRole('img', { name: 'Preview of pdi.png' })
    expect(img).toHaveAttribute('src', 'blob:preview-1')
  })

  it('shows a load error with retry', async () => {
    mockFetch({
      '/attachments/limits': () => Response.json({ data: limits }),
      '/attachments': () => errorResponse(500, 'INTERNAL_ERROR', 'Something went wrong.'),
    })
    renderApp('/documents', { me: both })
    expect(await screen.findByText('Could not load your uploads')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
  })
})
