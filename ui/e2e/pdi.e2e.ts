import { expect, test, type Page } from '@playwright/test'
import { hasAccount, signIn, watchConsole } from './support'

/**
 * PDI checklist and disposition (D-06) against the real API and PostgreSQL:
 * configure a checklist item, pass one vehicle (becomes saleable), hold another
 * with a defect note and a private photo (not saleable), reject the held one,
 * refuse a stale submission, reload, read-only and no-permission users.
 *
 * Needs vehicle units awaiting PDI on a THROWAWAY database: receipt posting is
 * D-07, so they are inserted there directly (see DEVELOPMENT_SETUP "End-to-end
 * browser tests"). PROPOSED grants: E2E_CATALOGUE needs vehicle_units.view,
 * vehicle_units.pdi.record, vehicle_units.pdi.checklist.manage, attachments.upload,
 * attachments.view and the units' location in scope; E2E_CATALOGUE_VIEWER
 * vehicle_units.view (same location); E2E_* no grants.
 */

// A few bytes the server's content sniffing reads as JPEG (placeholder photo).
const jpeg = Buffer.from([
  0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00,
])

const visible = (page: Page, text: string | RegExp) =>
  page.getByText(text).filter({ visible: true }).first()

async function pendingUnits(page: Page): Promise<{ id: string; unitNumber: string }[]> {
  const res = await page.request.get('/api/v1/vehicle-units?status=RECEIVED_PENDING_PDI&limit=100')
  expect(res.ok(), await res.text()).toBe(true)
  const items = (
    (await res.json()) as {
      data: { items: { id: string; unitNumber: string; chassisNumber: string | null }[] }
    }
  ).data.items
  return items.filter((u) => u.chassisNumber)
}

/** Answers every checklist item on the page with Pass, except `fail` (with a note). */
async function answerAll(page: Page, fail?: string) {
  const groups = page
    .getByRole('radiogroup')
    .filter({ has: page.getByRole('radio', { name: 'Fail' }) })
  const n = await groups.count()
  expect(n).toBeGreaterThan(0)
  for (let i = 0; i < n; i++) {
    const g = groups.nth(i)
    const label = (await g.getAttribute('aria-label'))?.replace(/ outcome$/, '') ?? ''
    if (fail && label === fail) {
      await g.getByText('Fail').click()
      await page.getByLabel(`Defect note — ${label}`).fill('Rear brake lever loose')
    } else {
      await g.getByText('Pass').click()
    }
  }
}

test.describe('PDI', () => {
  test('configure, pass, hold with evidence, reject, refuse stale, stay read-only for others', async ({
    page,
    browser,
  }, info) => {
    test.skip(!hasAccount('catalogue'), 'Set E2E_CATALOGUE_* (see DEVELOPMENT_SETUP)')
    test.setTimeout(150_000)
    const problems = watchConsole(page)
    const tag = `${info.project.name.slice(0, 1)}${Date.now().toString(36)}`.toUpperCase()
    await signIn(page, '/procurement/pdi/checklist', 'catalogue')
    const units = await pendingUnits(page)
    test.skip(units.length < 2, 'Needs two vehicle units awaiting PDI on the throwaway database')
    const [passUnit, holdUnit] = units as [
      { id: string; unitNumber: string },
      { id: string; unitNumber: string },
    ]

    // Configure a checklist item through the UI.
    const code = `E2E_${tag}`
    await page.getByRole('button', { name: 'Add item' }).first().click()
    let dialog = page.getByRole('dialog', { name: 'Add checklist item' })
    await dialog.getByLabel(/^Code/).fill(code)
    await dialog.getByLabel(/^Label/).fill(`Brakes ${tag}`)
    await dialog.getByRole('button', { name: 'Add item' }).click()
    await expect(dialog).toHaveCount(0)
    await expect(visible(page, code)).toBeVisible()

    // PASS: confirmation states the effect; the vehicle becomes saleable only after the server confirms.
    await page.goto(`/procurement/pdi/${passUnit.id}`)
    await expect(page.getByRole('heading', { level: 1, name: passUnit.unitNumber })).toBeVisible()
    await expect(visible(page, /Not saleable\./)).toBeVisible()
    await answerAll(page)
    await page
      .getByRole('group', { name: 'Disposition' })
      .getByText('Pass', { exact: true })
      .click()
    await page.getByRole('button', { name: 'Review inspection' }).click()
    dialog = page.getByRole('dialog', { name: `Pass ${passUnit.unitNumber}?` })
    await expect(dialog.getByText(/becomes available \(saleable\) stock/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Record pass' }).dblclick()
    await expect(
      visible(page, /recorded\. The vehicle is now Available and saleable/),
    ).toBeVisible()
    await expect(visible(page, /Saleable\./)).toBeVisible()
    let res = await page.request.get(`/api/v1/vehicle-units/${passUnit.id}/pdi`)
    let o = (
      (await res.json()) as {
        data: { unit: { status: string; saleable: boolean }; inspections: unknown[] }
      }
    ).data
    expect(o.unit).toMatchObject({ status: 'AVAILABLE', saleable: true })
    expect(o.inspections).toHaveLength(1)

    // HOLD: failed item with note, notes, a private photo for the unit's location.
    await page.goto(`/procurement/pdi/${holdUnit.id}`)
    await expect(page.getByRole('heading', { level: 1, name: holdUnit.unitNumber })).toBeVisible()
    await answerAll(page, `Brakes ${tag}`)
    await page.getByLabel(/^Defect notes/).fill(`Scratched side panel ${tag}`)
    await page
      .getByLabel('Choose files to upload')
      .setInputFiles({ name: 'panel.jpg', mimeType: 'image/jpeg', buffer: jpeg })
    await expect(page.getByRole('list', { name: 'Evidence for this inspection' })).toContainText(
      'panel.jpg',
    )
    await page
      .getByRole('group', { name: 'Disposition' })
      .getByText('Hold', { exact: true })
      .click()
    await page.getByRole('button', { name: 'Review inspection' }).click()
    dialog = page.getByRole('dialog', { name: `Hold ${holdUnit.unitNumber}?` })
    await expect(dialog.getByText(/not saleable and cannot be reserved or sold/)).toBeVisible()
    await dialog.getByRole('button', { name: 'Put on hold' }).click()
    await expect(
      visible(page, /recorded\. The vehicle is now Quality hold and not saleable/),
    ).toBeVisible()
    await expect(visible(page, /on quality hold after a failed inspection/)).toBeVisible()
    // Held vehicles are excluded from saleable search.
    res = await page.request.get('/api/v1/vehicle-units?saleable=true&limit=100')
    const saleable = ((await res.json()) as { data: { items: { id: string }[] } }).data.items.map(
      (u) => u.id,
    )
    expect(saleable).toContain(passUnit.id)
    expect(saleable).not.toContain(holdUnit.id)
    // A request carrying the version from before the hold is refused; nothing is saved.
    const csrf = (
      (await (await page.request.get('/api/v1/auth/session')).json()) as {
        data: { csrfToken: string }
      }
    ).data.csrfToken
    const stale = await page.request.post(`/api/v1/vehicle-units/${holdUnit.id}/pdi`, {
      headers: { 'X-CSRF-Token': csrf, 'Idempotency-Key': `stale-${tag}` },
      data: {
        version: 1,
        disposition: 'HOLD',
        notes: 'stale',
        attachmentIds: ['00000000-0000-4000-8000-000000000000'],
      },
    })
    expect(stale.status()).toBe(409)
    expect(((await stale.json()) as { error: { code: string } }).error.code).toBe(
      'VERSION_CONFLICT',
    )

    // History shows the evidence (downloadable through a signed link only).
    await page.reload()
    const history = page.getByRole('list', { name: 'Inspections' })
    await expect(history.getByText(`Scratched side panel ${tag}`)).toBeVisible()
    await history.getByRole('button', { name: 'Show details' }).first().click()
    await expect(history.getByRole('button', { name: 'Download panel.jpg' })).toBeVisible()
    const download = page.waitForEvent('download')
    await history.getByRole('button', { name: 'Download panel.jpg' }).click()
    expect((await download).suggestedFilename()).toBe('panel.jpg')

    // REJECT is the only option for a held vehicle; a stale submission is refused with nothing saved.
    const group = page.getByRole('group', { name: 'Disposition' })
    await expect(group.getByRole('radio')).toHaveCount(1)
    await answerAll(page, `Brakes ${tag}`)
    await page.getByLabel(/^Defect notes/).fill('Panel cannot be repaired')
    await page
      .getByLabel('Choose files to upload')
      .setInputFiles({ name: 'panel-2.jpg', mimeType: 'image/jpeg', buffer: jpeg })
    await expect(page.getByRole('list', { name: 'Evidence for this inspection' })).toContainText(
      'panel-2.jpg',
    )
    await page.getByRole('button', { name: 'Review inspection' }).click()
    dialog = page.getByRole('dialog', { name: `Reject ${holdUnit.unitNumber}?` })
    await dialog.getByRole('button', { name: 'Reject vehicle' }).click()
    await expect(
      visible(page, /recorded\. The vehicle is now Rejected and not saleable/),
    ).toBeVisible()
    await expect(visible(page, /No inspection can be recorded/)).toBeVisible()
    res = await page.request.get(`/api/v1/vehicle-units/${holdUnit.id}/pdi`)
    o = (
      (await res.json()) as {
        data: { unit: { status: string; saleable: boolean }; inspections: unknown[] }
      }
    ).data
    expect(o.unit).toMatchObject({ status: 'REJECTED', saleable: false })
    expect(o.inspections).toHaveLength(2)
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow).toBeLessThanOrEqual(1)

    // A viewer reads the history but cannot record; a user without grants is denied.
    if (hasAccount('catalogueViewer')) {
      const vctx = await browser.newContext({ ...info.project.use })
      const viewer = await vctx.newPage()
      await signIn(viewer, `/procurement/pdi/${passUnit.id}`, 'catalogueViewer')
      await expect(visible(viewer, /Saleable\./)).toBeVisible()
      await expect(viewer.getByRole('button', { name: 'Review inspection' })).toHaveCount(0)
      await vctx.close()
    }
    const nctx = await browser.newContext()
    const nobody = await nctx.newPage()
    await signIn(nobody, '/procurement/pdi', 'active')
    await expect(
      visible(nobody, /You do not have permission to view vehicle inspections/),
    ).toBeVisible()
    await nctx.close()

    // Retire the item so later runs start from the same checklist.
    await page.goto('/procurement/pdi/checklist')
    await page
      .getByRole('button', { name: `Retire ${code}` })
      .filter({ visible: true })
      .first()
      .click()
    dialog = page.getByRole('dialog', { name: `Retire ${code}?` })
    await dialog.getByLabel(/^Reason/).fill('End of browser test')
    await dialog.getByRole('button', { name: 'Retire item' }).click()
    await expect(dialog).toHaveCount(0)
    expect(problems).toEqual([])
  })
})
