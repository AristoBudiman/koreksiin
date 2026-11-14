import { createRequire } from 'module'
import path from 'path'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_CORE_PATH || 'playwright-core')

const token = process.argv[2]
const outDir = process.argv[3] || ''
const chromePath = process.env.CHROME_PATH || 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'

const widths = [375, 1024, 1152, 1280, 1310, 1366, 1440, 1536]
const modes = ['light', 'dark']

const browser = await chromium.launch({ executablePath: chromePath, headless: true })
const page = await browser.newPage()

await page.goto('http://localhost:8080/masuk')
await page.evaluate((t) => localStorage.setItem('koreksiin_token', t), token)

let anyFailure = false

for (const w of widths) {
  await page.setViewportSize({ width: w, height: 800 })
  for (const mode of modes) {
    await page.goto('http://localhost:8080/guru/langganan', { waitUntil: 'networkidle' })
    await page.evaluate((isDark) => {
      document.documentElement.classList.toggle('dark', isDark)
    }, mode === 'dark')
    await page.waitForTimeout(250)

    const report = await page.evaluate((vw) => {
      const headerRow = document.querySelector('header > div:first-child')
      const children = Array.from(headerRow.children)
      const rightGroup = children[2]

      const items = [
        { name: 'logo', el: children[0] },
        { name: 'nav', el: children[1] },
        ...Array.from(rightGroup.children).map((el, idx) => ({
          name: ['quota', 'theme', 'profile', 'logout'][idx] || `right_${idx}`,
          el
        }))
      ]

      const visible = []
      for (const item of items) {
        if (!item.el) continue
        const r = item.el.getBoundingClientRect()
        if (r.width === 0 || r.height === 0) continue
        // For nav, check actual content right edge of its last visible link
        let effectiveRight = r.right
        if (item.name === 'nav') {
          const links = Array.from(item.el.querySelectorAll('a'))
          const lastLink = links[links.length - 1]
          if (lastLink) {
            effectiveRight = lastLink.getBoundingClientRect().right
          }
        }
        visible.push({
          name: item.name,
          left: Math.round(r.left),
          right: Math.round(effectiveRight),
          width: Math.round(effectiveRight - r.left)
        })
      }

      const overlaps = []
      for (let i = 0; i < visible.length - 1; i++) {
        const a = visible[i]
        const b = visible[i + 1]
        if (a.right > b.left) {
          overlaps.push(`${a.name}(${a.right}) > ${b.name}(${b.left})`)
        }
      }
      const overflow = visible.filter((v) => v.right > vw).map((v) => `${v.name}(${v.right}>${vw})`)
      const quotaText = rightGroup.children[0]?.textContent?.trim() || ''

      return { visible, overlaps, overflow, quotaText }
    }, w)

    const status = report.overlaps.length === 0 && report.overflow.length === 0 ? 'OK' : 'FAIL'
    if (status === 'FAIL') anyFailure = true
    const gaps = report.visible
      .slice(0, -1)
      .map((v, i) => `${v.name}->${report.visible[i + 1].name}:${report.visible[i + 1].left - v.right}px`)
      .join(', ')
    console.log(`[${status}] w=${w} ${mode} | quota="${report.quotaText}" | gaps=[${gaps}]${report.overlaps.length ? ' | OVERLAPS=' + report.overlaps.join(';') : ''}${report.overflow.length ? ' | OVERFLOW=' + report.overflow.join(';') : ''}`)

    if (outDir && [375, 1024, 1310, 1536].includes(w)) {
      await page.screenshot({ path: path.join(outDir, `worstcase_${w}_${mode}.png`) })
    }
  }
}

await browser.close()
if (anyFailure) process.exit(1)
