import { test, expect } from '@playwright/test'
import { existsSync, readFileSync } from 'node:fs'
import { createServer, type Server } from 'node:http'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { AddressInfo } from 'node:net'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const DIST_DIR = resolve(__dirname, '..', '..', 'dist')
const INDEX_HTML = join(DIST_DIR, 'index.html')

const hasBuild = existsSync(INDEX_HTML)

test.describe('Built single-file artifact', () => {
  test.skip(!hasBuild, `web/report/dist/index.html not found — run \`yarn build\` to enable this suite`)

  let server: Server
  let baseURL: string

  test.beforeAll(async () => {
    const html = readFileSync(INDEX_HTML, 'utf-8')
    server = createServer((req, res) => {
      res.setHeader('content-type', 'text/html; charset=utf-8')
      res.end(html)
    })
    await new Promise<void>(ok => server.listen(0, '127.0.0.1', ok))
    const port = (server.address() as AddressInfo).port
    baseURL = `http://127.0.0.1:${port}`
  })

  test.afterAll(async () => {
    await new Promise<void>(ok => server.close(() => ok()))
  })

  test('renders gallery with at least one image card from mock fallback', async ({ page }) => {
    await page.goto(baseURL)
    await expect(page.locator('.image-card').first()).toBeVisible()
    await expect(page.locator('.page-header h1')).toHaveText('Image Overview')
  })

  test('navigates to a detail page in the bundled artifact', async ({ page }) => {
    await page.goto(baseURL)
    await page.locator('.image-card').first().click()
    await expect(page).toHaveURL(/#\/image\//)
    await expect(page.locator('.detail-title')).toBeVisible()
  })
})
