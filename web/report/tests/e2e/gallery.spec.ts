import { test, expect } from '@playwright/test'
import { mockData } from '../../src/mockData'

test.describe('Gallery', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('renders a card for every base image and variant from mock data', async ({ page }) => {
    const expectedCards = mockData.images.flatMap(img => {
      const entries = [img.name]
      img.variants?.forEach(v => entries.push(`${img.name}${v.tagSuffix}`))
      return entries
    })

    for (const displayName of expectedCards) {
      await expect(
        page.locator('.image-card').filter({ has: page.locator('.image-name', { hasText: new RegExp(`^${displayName}$`) }) })
      ).toHaveCount(1)
    }

    await expect(page.locator('.image-card')).toHaveCount(expectedCards.length)
  })

  test('base and variant cards have correct kind badges', async ({ page }) => {
    const baseCards = page.locator('.image-card .card-kind-badge.base')
    await expect(baseCards.first()).toHaveText('Base')

    const variant = mockData.images.flatMap(img =>
      (img.variants ?? []).map(v => `${img.name}${v.tagSuffix}`)
    )[0]
    if (variant) {
      await expect(
        page.locator('.image-card').filter({ hasText: variant }).locator('.card-kind-badge.variant')
      ).toHaveText('Variant')
    }
  })

  test('search filters the gallery by image name', async ({ page }) => {
    const query = 'nginx'
    const searchBox = page.locator('input.search-box')
    await searchBox.fill(query)

    await expect(page.locator('.image-card')).toHaveCount(1)
    await expect(page.locator('.image-card .image-name')).toContainText('nginx')
  })

  test('search filters by description and shows empty state when nothing matches', async ({ page }) => {
    const searchBox = page.locator('input.search-box')
    await searchBox.fill('definitely-no-such-image-xyz')

    await expect(page.locator('.image-card')).toHaveCount(0)
    await expect(page.locator('.no-data')).toContainText('No images found')
  })

  test('platform badges render on cards that declare platforms', async ({ page }) => {
    const imageWithPlatforms = mockData.images.find(img => (img.platforms ?? []).length > 0)!
    const card = page.locator('.image-card').filter({
      has: page.locator('.image-name', { hasText: new RegExp(`^${imageWithPlatforms.name}$`) }),
    })
    for (const platform of imageWithPlatforms.platforms!) {
      await expect(card.locator('.platform-badge', { hasText: platform })).toBeVisible()
    }
  })

  test('clicking a base card navigates to its detail page', async ({ page }) => {
    const target = mockData.images[0]
    await page
      .locator('.image-card')
      .filter({ has: page.locator('.image-name', { hasText: new RegExp(`^${target.name}$`) }) })
      .click()

    await expect(page).toHaveURL(new RegExp(`#/image/${target.name}/base$`))
    await expect(page.locator('.detail-title')).toHaveText(target.name)
  })
})
