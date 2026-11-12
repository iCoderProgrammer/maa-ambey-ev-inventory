// Route x account x viewport crawl on the isolated stack (scratchpad tool, not part of the repo).
const { chromium } = require('playwright')
const fs = require('fs')

const base = process.env.E2E_BASE_URL
const pw = process.env.E2E_CATALOGUE_PASSWORD
const accounts = {
  owner: 'e2e.catalogue',
  viewer: 'e2e.catviewer',
  approver: 'e2e.priceapprover',
  nogrants: 'e2e.active',
  roles: 'e2e.roles',
}
const viewports = { desktop: { width: 1366, height: 900 }, tablet: { width: 768, height: 1024 }, mobile: { width: 390, height: 844 } }

async function login(page, username) {
  await page.goto(base + '/login')
  await page.getByLabel('Username').fill(username)
  await page.getByLabel('Password', { exact: false }).first().fill(pw)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await page.getByRole('button', { name: /^Account: / }).waitFor()
}

async function ids(page) {
  const j = async (p) => (await (await page.request.get(base + '/api/v1' + p)).json()).data
  const first = async (p) => (await j(p)).items?.[0]?.id
  return {
    brand: await first('/brands?limit=1'),
    supplier: await first('/suppliers?limit=1'),
    po: await first('/purchase-orders?limit=1'),
    grn: await first('/goods-receipts?limit=1'),
    part: await first('/stock-items?limit=1'),
    count: await first('/stock-counts?limit=1'),
    unit: await first('/vehicle-units?limit=1'),
  }
}

;(async () => {
  const browser = await chromium.launch({ channel: 'chrome' })
  const ctx0 = await browser.newContext()
  const p0 = await ctx0.newPage()
  await login(p0, accounts.owner)
  const id = await ids(p0)
  await ctx0.close()
  const routes = [
    '/', '/documents', '/admin/users', '/admin/catalogue', '/admin/catalogue/models', '/admin/catalogue/variants',
    `/admin/catalogue/brands/${id.brand}`, '/procurement/suppliers', `/procurement/suppliers/${id.supplier}`,
    '/procurement/purchase-orders', `/procurement/purchase-orders/${id.po}`, '/procurement/goods-receipts',
    `/procurement/goods-receipts/${id.grn}`, '/procurement/pdi', '/procurement/pdi/holds', '/procurement/pdi/checklist',
    `/procurement/pdi/${id.unit}`, '/inventory/parts', '/inventory/parts/categories', `/inventory/parts/${id.part}`,
    '/inventory/counts', '/inventory/counts/approvals', `/inventory/counts/${id.count}`, '/inventory/vehicles',
    '/inventory/movements', '/sales/customers', '/sales/quotations', '/sales/reservations', '/sales/orders', '/reports',
    '/admin/settings', '/admin/audit', '/no-such-page',
  ]
  const rows = []
  let loads = 0
  for (const [who, username] of Object.entries(accounts)) {
    for (const [vpName, vp] of Object.entries(viewports)) {
      const ctx = await browser.newContext({ viewport: vp })
      const page = await ctx.newPage()
      const problems = []
      page.on('pageerror', (e) => problems.push('pageerror: ' + e.message))
      page.on('console', (m) => {
        if (m.type() === 'error' && !m.text().includes('Failed to load resource')) problems.push('console: ' + m.text())
      })
      page.on('response', (r) => {
        if (r.status() >= 500) problems.push(`5xx: ${r.status()} ${r.url()}`)
      })
      await login(page, username)
      for (const route of routes) {
        problems.length = 0
        await page.goto(base + route)
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(150)
        const r = await page.evaluate(() => {
          const vis = (e) => !!(e.offsetWidth || e.offsetHeight || e.getClientRects().length)
          const h1 = [...document.querySelectorAll('h1')].filter(vis)
          const unnamed = [...document.querySelectorAll('button, a[href]')].filter(
            (e) => vis(e) && !(e.getAttribute('aria-label') || e.textContent.trim() || e.getAttribute('title')),
          ).length
          const unlabelled = [...document.querySelectorAll('input:not([type=hidden]), select, textarea')].filter(
            (e) => vis(e) && !(e.labels?.length || e.getAttribute('aria-label') || e.getAttribute('aria-labelledby')),
          ).length
          const idList = [...document.querySelectorAll('[id]')].map((e) => e.id)
          const dupIds = idList.length - new Set(idList).size
          return {
            h1: h1.map((e) => e.textContent.trim()),
            overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
            unnamed, unlabelled, dupIds, url: location.pathname,
            denied: /You do not have permission/.test(document.body.innerText),
          }
        })
        loads++
        const flags = []
        if (r.h1.length !== 1) flags.push(`h1=${r.h1.length}`)
        if (r.overflow > 1) flags.push(`overflow=${r.overflow}`)
        if (r.unnamed) flags.push(`unnamed=${r.unnamed}`)
        if (r.unlabelled) flags.push(`unlabelled=${r.unlabelled}`)
        if (r.dupIds) flags.push(`dupIds=${r.dupIds}`)
        if (r.url === '/login') flags.push('redirected-to-login')
        flags.push(...problems)
        rows.push({ who, vp: vpName, route: route.replace(/[0-9a-f-]{36}/, ':id'), h1: r.h1[0] ?? '', denied: r.denied, flags })
      }
      await ctx.close()
    }
  }
  await browser.close()
  const flagged = rows.filter((r) => r.flags.length)
  const out = [`page loads: ${loads}`, `flagged: ${flagged.length}`, ...flagged.map((r) => `${r.who} ${r.vp} ${r.route}: ${r.flags.join('; ')}`), '',
    'all rows (who viewport route | h1 | denied):', ...rows.map((r) => `${r.who} ${r.vp} ${r.route} | ${r.h1} | ${r.denied ? 'denied' : ''}`)]
  fs.writeFileSync(process.argv[2], out.join('\n') + '\n')
  console.log(out.slice(0, 2 + flagged.length).join('\n'))
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
