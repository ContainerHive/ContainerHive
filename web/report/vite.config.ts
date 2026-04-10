import { defineConfig, type Plugin } from 'vite'
import preact from '@preact/preset-vite'
import { viteSingleFile } from 'vite-plugin-singlefile'

function removeDeviconSvg(): Plugin {
  return {
    name: 'remove-devicon-svg',
    enforce: 'pre' as const,
    transform(code, id) {
      if (id.endsWith('.css') && code.includes('devicon')) {
        console.log('Transforming CSS:', id)
        return code.replace(/url\(['"]?[^'"]*devicon\.svg[^'"]*['"]?\)/g, "/* devicon-svg-removed */")
      }
      return null
    }
  }
}

export default defineConfig({
  plugins: [
    preact(),
    removeDeviconSvg(),
    viteSingleFile(),
  ],
  build: {
    target: 'esnext',
    minify: 'esbuild',
    cssCodeSplit: false,
    assetsInlineLimit: 100000000,
  },
  base: './',
})