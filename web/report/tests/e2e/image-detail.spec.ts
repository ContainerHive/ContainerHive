import { test, expect } from '@playwright/test'
import { mockData } from '../../src/mockData'

const base = mockData.images.find(img => img.name === 'alpine')!
const baseSecondTag = base.tags[1]
const variant = base.variants![0]
const variantDisplayName = `${base.name}${variant.tagSuffix}`

test.describe('Image detail — base image', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`/#/image/${base.name}/base`)
  })

  test('renders title, base badge, description, and back link', async ({ page }) => {
    await expect(page.locator('.detail-title')).toHaveText(base.name)
    await expect(page.locator('.variant-badge')).toHaveText('base')
    await expect(page.locator('.description-panel')).toContainText(base.description!)
    await expect(page.locator('a.back-link')).toBeVisible()
  })

  test('renders one tab per tag, first tab active by default', async ({ page }) => {
    const tabs = page.locator('.tabs .tab')
    await expect(tabs).toHaveCount(base.tags.length)
    await expect(tabs.first()).toHaveClass(/active/)
    await expect(tabs.first()).toHaveText(base.tags[0].name)
  })

  test('switching a tab updates build args and versions', async ({ page }) => {
    const secondTab = page.locator('.tabs .tab', { hasText: baseSecondTag.name })
    await secondTab.click()
    await expect(secondTab).toHaveClass(/active/)

    if (baseSecondTag.versions) {
      for (const [key, value] of Object.entries(baseSecondTag.versions)) {
        await expect(page.locator('.versions-section')).toContainText(key)
        await expect(page.locator('.versions-section')).toContainText(value)
      }
    }

    if (baseSecondTag.buildArgs) {
      for (const [key, value] of Object.entries(baseSecondTag.buildArgs)) {
        await expect(page.locator('.build-args-section')).toContainText(key)
        await expect(page.locator('.build-args-section')).toContainText(value)
      }
    }
  })

  test('platform cards render for each tag platform', async ({ page }) => {
    const platforms = base.tags[0].platforms ?? []
    for (const p of platforms) {
      await expect(page.locator('.platform-card', { hasText: p.platform })).toBeVisible()
    }
  })
})

test.describe('Image detail — variant', () => {
  test('variant URL renders the variant name in the heading and badge', async ({ page }) => {
    await page.goto(`/#/image/${base.name}/${variantDisplayName}`)

    await expect(page.locator('.detail-title')).toHaveText(variantDisplayName)
    await expect(page.locator('.variant-badge')).toHaveText('variant')
  })

  test('variant shows only its own tags', async ({ page }) => {
    await page.goto(`/#/image/${base.name}/${variantDisplayName}`)

    const tabs = page.locator('.tabs .tab')
    await expect(tabs).toHaveCount(variant.tags.length)
    await expect(tabs.first()).toHaveText(variant.tags[0].name)
  })
})

test.describe('Image detail — not found', () => {
  test('renders a not-found message for an unknown image', async ({ page }) => {
    await page.goto('/#/image/does-not-exist/base')
    await expect(page.locator('.no-data')).toContainText('Image not found')
  })
})
