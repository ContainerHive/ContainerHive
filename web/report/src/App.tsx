import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import ThemeToggle from './components/ThemeToggle'
import type { ProjectReport } from './types'
import logo from './logo.png'

type Kind = 'base' | 'variant'

interface FlattenedItem {
  imageName: string
  displayName: string
  kind: Kind
  icon?: string
  platforms: string[]
  tagCount: number
}

function App({ data }: Readonly<{ data: ProjectReport }>) {
  const [search, setSearch] = useState('')

  const flattenedItems = useMemo(() => {
    const items: FlattenedItem[] = []

    data.images.forEach(img => {
      items.push({
        imageName: img.name,
        displayName: img.name,
        kind: 'base',
        icon: img.report?.icon,
        platforms: img.platforms || [],
        tagCount: img.tags.length,
      })

      if (img.variants) {
        img.variants.forEach(variant => {
          items.push({
            imageName: img.name,
            displayName: `${img.name}${variant.tagSuffix}`,
            kind: variant.name as Kind,
            icon: variant.report?.icon,
            platforms: variant.platforms || img.platforms || [],
            tagCount: variant.tags.length,
          })
        })
      }
    })

    return items.filter(item =>
      item.displayName.toLowerCase().includes(search.toLowerCase()) ||
      item.imageName.toLowerCase().includes(search.toLowerCase())
    )
  }, [data.images, search])

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
                      <i className={item.icon}></i>
                    ) : (
                      <span>📦</span>
                    )}
                  </div>
                  <div className="image-name">{item.displayName}</div>
                </div>
                <div className="image-meta">
                  <span><span className="tag-icon"></span> {item.tagCount} tag{item.tagCount !== 1 ? 's' : ''}</span>
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
