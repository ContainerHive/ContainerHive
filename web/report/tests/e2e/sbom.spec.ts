import { test, expect } from '@playwright/test'
import { mockData } from '../../src/mockData'

const image = mockData.images.find(img => img.name === 'alpine')!
const firstTag = image.tags[0]
const platformWithSbom = firstTag.platforms!.find(p => (p.sbom ?? []).length > 0)!
const firstPackage = platformWithSbom.sbom![0]

test.describe('SBOM viewer', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`/#/image/${image.name}/base`)
  })

  test('renders SBOM table with package rows for a platform with SBOM', async ({ page }) => {
    const platformCard = page.locator('.platform-card', { hasText: platformWithSbom.platform })
    await expect(platformCard.locator('.sbom-badge')).toHaveText('SBOM')
    await expect(platformCard.locator('.sbom-count')).toContainText(
      `${platformWithSbom.sbom!.length} packages`
    )
    await expect(platformCard.locator('table.sbom-table tbody tr')).toHaveCount(
      platformWithSbom.sbom!.length
    )
  })

  test('search filters SBOM rows by package name', async ({ page }) => {
    await page.locator('.sbom-search input').fill(firstPackage.name)

    const platformCard = page.locator('.platform-card', { hasText: platformWithSbom.platform })
    const rows = platformCard.locator('table.sbom-table tbody tr')
    await expect(rows).toHaveCount(
      platformWithSbom.sbom!.filter(pkg =>
        pkg.name.toLowerCase().includes(firstPackage.name.toLowerCase()) ||
        (pkg.version ?? '').toLowerCase().includes(firstPackage.name.toLowerCase())
      ).length
    )
    await expect(rows.first()).toContainText(firstPackage.name)
  })

  test('search showing no matches renders a "No matches" state', async ({ page }) => {
    await page.locator('.sbom-search input').fill('pkg-that-does-not-exist-xyz')

    const platformCard = page.locator('.platform-card', { hasText: platformWithSbom.platform })
    await expect(platformCard.locator('table.sbom-table')).toHaveCount(0)
    await expect(platformCard.locator('.no-data')).toContainText('No matches')
  })

  test('switching tags resets the SBOM search input', async ({ page }) => {
    await page.locator('.sbom-search input').fill(firstPackage.name)
    await expect(page.locator('.sbom-search input')).toHaveValue(firstPackage.name)

    const secondTab = page.locator('.tabs .tab', { hasText: image.tags[1].name })
    await secondTab.click()

    await expect(page.locator('.sbom-search input')).toHaveValue('')
  })
})
