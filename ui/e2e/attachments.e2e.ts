import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

// 1×1 PNG and a tiny PDF, generated here: no real documents are used.
const png = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64',
)
const pdf = Buffer.from('%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n%%EOF\n')

async function csrf(request: APIRequestContext): Promise<string> {
  const res = await request.get('/api/v1/auth/session')
  const body = (await res.json()) as { data: { csrfToken: string } }
  return body.data.csrfToken
}

async function apiUpload(page: Page, name: string, mimeType: string, buffer: Buffer) {
  return page.request.post('/api/v1/attachments', {
    headers: { 'X-CSRF-Token': await csrf(page.request) },
    multipart: { file: { name, mimeType, buffer } },
  })
}

async function errorCode(res: { json: () => Promise<unknown> }): Promise<string> {
  return ((await res.json()) as { error: { code: string } }).error.code
}

test.describe('private attachments (B-09)', () => {
  test('a user without attachment permissions sees no Documents and is denied', async ({
    page,
    isMobile,
  }) => {
    const problems = watchConsole(page)
    await signIn(page)
    if (!isMobile) {
      const nav = page.getByRole('navigation', { name: 'Main' })
      await expect(nav.getByRole('link', { name: /Users & Roles/ })).toHaveCount(0)
      await expect(nav.getByRole('link', { name: /Documents/ })).toHaveCount(0)
    }
    await page.goto('/documents')
    await expect(page.getByText('Access denied')).toBeVisible()
    for (const path of ['/api/v1/attachments', '/api/v1/attachments/limits']) {
      const res = await page.request.get(path)
      expect(res.status()).toBe(403)
    }
    const up = await apiUpload(page, 'a.pdf', 'application/pdf', pdf)
    expect(up.status()).toBe(403)
    expect(problems).toEqual([])
  })

  test.describe('with permissions', () => {
    test.skip(
      !hasAccount('attachments') || !hasAccount('attachmentsViewer'),
      'set E2E_ATTACH_* and E2E_ATTACH_VIEWER_* for users holding the attachment permissions',
    )

    test('upload, list, preview, download and refresh', async ({ page, isMobile }) => {
      const problems = watchConsole(page)
      await signIn(page, '/', 'attachments')
      if (!isMobile) {
        await page
          .getByRole('navigation', { name: 'Main' })
          .getByRole('link', { name: /Documents/ })
          .click()
      } else {
        await page.goto('/documents')
      }
      await expect(page.getByRole('heading', { level: 1, name: 'Documents' })).toBeVisible()
      await expect(page.getByText(/JPG, PNG, WEBP, PDF · up to 10 MB each/)).toBeVisible()

      const stamp = `${Date.now()}`
      const pdfName = `bill-${stamp}.pdf`
      const pngName = `pdi-${stamp}.png`
      await page.getByLabel('Choose files to upload').setInputFiles([
        { name: pdfName, mimeType: 'application/pdf', buffer: pdf },
        { name: pngName, mimeType: 'image/png', buffer: png },
      ])
      await expect(page.getByText('Uploaded and stored privately.')).toHaveCount(2)
      const list = page.getByRole('list', { name: 'My uploads' })
      await expect(list.getByText(pdfName)).toBeVisible()
      await expect(list.getByText(pngName)).toBeVisible()

      // Client-side checks: wrong type and empty file never reach the server.
      await page
        .getByLabel('Choose files to upload')
        .setInputFiles([
          { name: `notes-${stamp}.txt`, mimeType: 'text/plain', buffer: Buffer.from('hello') },
        ])
      await expect(page.getByText(/This file type is not accepted/)).toBeVisible()

      // Preview the image (from memory, through a signed link).
      await list.getByRole('button', { name: `Preview ${pngName}` }).click()
      const dialog = page.getByRole('dialog', { name: pngName })
      const img = dialog.getByRole('img', { name: `Preview of ${pngName}` })
      await expect(img).toBeVisible()
      expect(await img.getAttribute('src')).toMatch(/^blob:/)
      expect(await img.evaluate((el: HTMLImageElement) => el.naturalWidth)).toBe(1)
      await dialog.getByRole('button', { name: 'Close dialog' }).click()

      // Download the PDF and compare bytes.
      const downloadPromise = page.waitForEvent('download')
      await list.getByRole('button', { name: `Download ${pdfName}` }).click()
      const download = await downloadPromise
      expect(download.suggestedFilename()).toBe(pdfName)
      const path = await download.path()
      const { readFileSync } = await import('node:fs')
      expect(readFileSync(path).equals(pdf)).toBe(true)

      // Persisted: still listed after a reload.
      await page.reload()
      await expect(page.getByRole('list', { name: 'My uploads' }).getByText(pdfName)).toBeVisible()
      // No storage paths or signed links in the page.
      expect(await page.content()).not.toMatch(/signature=|var\/attachments|object_?key/i)

      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow).toBeLessThanOrEqual(0)
      expect(problems).toEqual([])
    })

    test('the server rejects bad files even when the client is bypassed', async ({ page }) => {
      await signIn(page, '/', 'attachments')
      const cases: [string, string, Buffer, number, string][] = [
        ['notes.txt', 'text/plain', Buffer.from('plain text'), 415, 'UNSUPPORTED_MEDIA_TYPE'],
        [
          'bill.pdf',
          'application/pdf',
          Buffer.from('<html><script>x</script>'),
          415,
          'UNSUPPORTED_MEDIA_TYPE',
        ],
        ['photo.png', 'image/png', pdf, 415, 'UNSUPPORTED_MEDIA_TYPE'],
        ['empty.pdf', 'application/pdf', Buffer.alloc(0), 422, 'VALIDATION_FAILED'],
        [
          'big.pdf',
          'application/pdf',
          Buffer.concat([pdf, Buffer.alloc(10 * 1024 * 1024)]),
          413,
          'FILE_TOO_LARGE',
        ],
      ]
      for (const [name, type, buffer, status, code] of cases) {
        const res = await apiUpload(page, name, type, buffer)
        expect(res.status(), name).toBe(status)
        expect(await errorCode(res), name).toBe(code)
      }
      const noCsrf = await page.request.post('/api/v1/attachments', {
        multipart: { file: { name: 'a.pdf', mimeType: 'application/pdf', buffer: pdf } },
      })
      expect(noCsrf.status()).toBe(403)
      expect(await errorCode(noCsrf)).toBe('CSRF_FAILED')
    })

    test('links are private, signed, short-lived and bound to one user', async ({
      page,
      browser,
      isMobile,
    }) => {
      test.skip(isMobile, 'API-level checks run once (desktop)')
      test.setTimeout(90_000)
      await signIn(page, '/', 'attachments')
      const up = await apiUpload(page, 'private.pdf', 'application/pdf', pdf)
      expect(up.status()).toBe(201)
      const att = ((await up.json()) as { data: { id: string } }).data
      const linkRes = await page.request.get(`/api/v1/attachments/${att.id}/link`)
      const link = ((await linkRes.json()) as { data: { url: string; expiresAt: string } }).data
      expect(link.url).toMatch(/^\/api\/v1\/attachments\/.+\/content\?/)

      const ok = await page.request.get(link.url)
      expect(ok.status()).toBe(200)
      expect(Buffer.from(await ok.body()).equals(pdf)).toBe(true)
      expect(ok.headers()['x-content-type-options']).toBe('nosniff')
      expect(ok.headers()['cache-control']).toBe('private, no-store')
      expect(ok.headers()['content-security-policy']).toContain('sandbox')
      expect(ok.headers()['content-disposition']).toMatch(/^attachment;/)

      const tampered = await page.request.get(link.url.replace(/signature=./, 'signature=A'))
      expect(tampered.status()).toBe(403)
      expect(await errorCode(tampered)).toBe('DOWNLOAD_LINK_INVALID')
      const inline = await page.request.get(
        link.url.replace('disposition=attachment', 'disposition=inline'),
      )
      expect(await errorCode(inline)).toBe('DOWNLOAD_LINK_INVALID')

      // Another user holding attachments.view: cannot see the file or use the link.
      const other = await browser.newContext()
      const otherPage = await other.newPage()
      await signIn(otherPage, '/', 'attachmentsViewer')
      expect((await otherPage.request.get(`/api/v1/attachments/${att.id}`)).status()).toBe(404)
      expect((await otherPage.request.get(`/api/v1/attachments/${att.id}/link`)).status()).toBe(404)
      const stolen = await otherPage.request.get(link.url)
      expect(stolen.status()).toBe(403)
      expect(await errorCode(stolen)).toBe('DOWNLOAD_LINK_INVALID')
      expect((await otherPage.request.post('/api/v1/attachments')).status()).toBe(403)
      await other.close()

      // Without a session the link is useless.
      const anon = await browser.newContext()
      expect((await anon.request.get(new URL(link.url, page.url()).href)).status()).toBe(401)
      await anon.close()

      // No mutation route exists for stored attachments.
      for (const method of ['PUT', 'PATCH', 'DELETE'] as const) {
        const res = await page.request.fetch(`/api/v1/attachments/${att.id}`, {
          method,
          headers: { 'X-CSRF-Token': await csrf(page.request) },
        })
        expect(res.status(), method).toBe(405)
      }

      // Expired: the isolated stack runs with ATTACHMENT_LINK_TTL=30s.
      const wait = new Date(link.expiresAt).getTime() - Date.now() + 1500
      expect(wait).toBeLessThan(35_000)
      await page.waitForTimeout(Math.max(wait, 0))
      const expired = await page.request.get(link.url)
      expect(expired.status()).toBe(403)
      expect(await errorCode(expired)).toBe('DOWNLOAD_LINK_EXPIRED')
    })
  })
})
