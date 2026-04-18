import { test, expect } from '@playwright/test'

test.describe('Navigation', () => {
  test('header title changes based on route', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('.page-header h1')).toHaveText('Image Overview')

    await page.locator('.header-right a.header-link', { hasText: 'About' }).click()
    await expect(page).toHaveURL(/#\/about$/)
    await expect(page.locator('.page-header h1')).toHaveText('About')

    await page.locator('.header-right a.header-link', { hasText: 'Licenses' }).click()
    await expect(page).toHaveURL(/#\/license$/)
    await expect(page.locator('.page-header h1')).toHaveText('Licenses')

    await page.locator('a.logo-icon').click()
    await expect(page).toHaveURL(/#?\/?$/)
    await expect(page.locator('.page-header h1')).toHaveText('Image Overview')
  })

  test('about page renders expected content', async ({ page }) => {
    await page.goto('/#/about')
    await expect(page.getByRole('heading', { name: 'Why this report?' })).toBeVisible()
    await expect(page.locator('.container')).toContainText('ContainerHive')
  })

  test('license page renders notice content in a pre block', async ({ page }) => {
    await page.goto('/#/license')
    const notice = page.locator('pre.notice-block')
    await expect(notice).toBeVisible()
    const text = await notice.textContent()
    expect(text?.length ?? 0).toBeGreaterThan(0)
  })
})

test.describe('Theme toggle', () => {
  test('toggles data-theme attribute and persists to localStorage across reloads', async ({ page }) => {
    await page.goto('/')

    const initial = await page.evaluate(() => document.documentElement.getAttribute('data-theme'))
    expect(['light', 'dark']).toContain(initial)
    const other = initial === 'light' ? 'dark' : 'light'

    await page.locator('button.theme-toggle').click()
    await expect(page.locator('html')).toHaveAttribute('data-theme', other)
    expect(await page.evaluate(() => localStorage.getItem('theme'))).toBe(other)

    await page.reload()
    await expect(page.locator('html')).toHaveAttribute('data-theme', other)
    expect(await page.evaluate(() => localStorage.getItem('theme'))).toBe(other)
  })
})
