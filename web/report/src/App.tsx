import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import ThemeToggle from './components/ThemeToggle'
import type { ProjectReport } from './types'
import logo from './logo.png'

interface FlattenedItem {
  imageName: string
  displayName: string
  kind: 'base' | 'variant'
  icon?: string
  platforms: string[]
  tagCount: number
  totalSize: number
  hasSbom: boolean
}

function App({ data }: { data: ProjectReport }) {
  const [search, setSearch] = useState('')

  const flattenedItems = useMemo(() => {
    const items: FlattenedItem[] = []
    
    data.images.forEach(img => {
      const allBasePlatforms = img.tags.flatMap(t => t.platforms).filter(p => p !== null)
      const basePlatforms = [...new Set(allBasePlatforms.map(p => p.platform))]
      const baseSize = allBasePlatforms.reduce((acc, p) => acc + p.size, 0)
      const baseHasSbom = allBasePlatforms.some(p => p.hasSbom)
      
      items.push({
        imageName: img.name,
        displayName: img.name,
        kind: 'base',
        icon: img.icon,
        platforms: basePlatforms,
        tagCount: img.tags.length,
        totalSize: baseSize,
        hasSbom: baseHasSbom
      })
      
      if (img.variants) {
        img.variants.forEach((variant, vIdx) => {
          const variantPlatformsList = variant.tags.flatMap(t => t.platforms).filter(p => p !== null)
          const variantPlatforms = [...new Set(variantPlatformsList.map(p => p.platform))]
          const variantSize = variantPlatformsList.reduce((acc, p) => acc + p.size, 0)
          const variantHasSbom = variantPlatformsList.some(p => p.hasSbom)
          
          items.push({
            imageName: img.name,
            displayName: `${img.name}${variant.tagSuffix}`,
            kind: variant.name,
            icon: variant.icon,
            platforms: variantPlatforms,
            tagCount: variant.tags.length,
            totalSize: variantSize,
            hasSbom: variantHasSbom
          })
        })
      }
    })
    
    return items.filter(item => 
      item.displayName.toLowerCase().includes(search.toLowerCase()) ||
      item.imageName.toLowerCase().includes(search.toLowerCase())
    )
  }, [data.images, search])

  const totalImages = data.images.length

  return (
    <>
      <header className="page-header">
        <div className="header-content">
          <div className="header-title">
            <img src={logo} alt="Logo" className="logo-icon" />
            <h1>Image Overview</h1>
          </div>
          <div className="header-right">
            <ThemeToggle />
          </div>
        </div>
      </header>
      <div className="container">
        <input
          type="text"
          className="search-box"
          placeholder="Search images..."
          value={search}
          onChange={e => setSearch(e.target.value)}
        />

        <div className="gallery">
          {flattenedItems.length === 0 ? (
            <div className="no-data">No images found</div>
          ) : (
            flattenedItems.map((item, idx) => (
              <Link 
                to={`/image/${encodeURIComponent(item.imageName)}/${item.kind}`} 
                key={`${item.imageName}-${item.kind}-${idx}`} 
                className="image-card"
              >
                <div className={`card-kind-badge ${item.kind}`}>
                  {item.kind === 'base' ? 'Base' : 'Variant'}
                </div>
                <div className="card-header">
                  <div className="card-icon">
                    {item.icon ? (
                      <i className={`devicon-${item.icon}-plain`}></i>
                    ) : (
                      <span>📦</span>
                    )}
                  </div>
                  <div className="image-name">{item.displayName}</div>
                </div>
                <div className="image-meta">
                  <span><span className="tag-icon"></span> {item.tagCount} tag{item.tagCount !== 1 ? 's' : ''}</span>
                  <span> | </span>
                  <span>{(item.totalSize / 1024 / 1024).toFixed(2)} MB</span>
                  {item.hasSbom && <span className="sbom-badge">✓ SBOM</span>}
                </div>
                <div className="platforms-list">
                  {item.platforms.map(platform => (
                    <span key={platform} className="platform-badge">{platform}</span>
                  ))}
                </div>
              </Link>
            ))
          )}
        </div>
      </div>
    </>
  )
}

export default App