import { defineConfig, type Plugin } from 'vite'
import preact from '@preact/preset-vite'
import { viteSingleFile } from 'vite-plugin-singlefile'
import webfontDl from 'vite-plugin-webfont-dl'
import { readFileSync, writeFileSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))

function inlineFontsPostBuild(): Plugin {
  return {
    name: 'inline-fonts-postbuild',
    apply: 'build',
    closeBundle() {
      const htmlPath = join(__dirname, 'dist', 'index.html')
      let html = readFileSync(htmlPath, 'utf-8')
      const fontFiles = [
        'UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa2JL7SUc-BOeWTOD4.woff2',
        'UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa0ZL7SUc-DqGufNeO.woff2',
        'UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa2ZL7SUc-DlzME5K_.woff2',
        'UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa1pL7SUc-CkhJZR-_.woff2',
        'UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa2pL7SUc-CBcvBZtf.woff2',
        'UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa25L7SUc-DO1Apj_S.woff2',
        'UcC73FwrK3iLTeHuS_nVMrMxCp50SjIa1ZL7-Dx4kXJAl.woff2',
        'V8mDoQDjQSkFtoMM3T6r8E7mPb54C-s0-D0rl6rjA.woff2',
        'V8mDoQDjQSkFtoMM3T6r8E7mPb94C-s0-D9tNdqV9.woff2',
        'V8mDoQDjQSkFtoMM3T6r8E7mPbF4Cw-BhU9QXUp.woff2',
      ]
      for (const fontFile of fontFiles) {
        const fontPath = join(__dirname, 'dist', fontFile)
        try {
          const fontData = readFileSync(fontPath)
          const base64 = fontData.toString('base64')
          const dataUri = `data:font/woff2;base64,${base64}`
          html = html.replace(new RegExp(`url\\([^)]*${fontFile}[^)]*\\)`, 'g'), `url(${dataUri})`)
        } catch {
          // Font file not found, skip
        }
      }
      writeFileSync(htmlPath, html)
    }
  }
}

export default defineConfig({
  plugins: [
    preact(),
    webfontDl({
      inject: {
        htmlLink: false,
      },
    }),
    viteSingleFile(),
    inlineFontsPostBuild(),
  ],
  build: {
    target: 'esnext',
    minify: 'esbuild',
    cssCodeSplit: false,
    assetsInlineLimit: 100000000,
  },
  base: './',
})