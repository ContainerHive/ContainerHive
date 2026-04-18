import { useState, useMemo } from 'react'
import ImageGalleryCard from './components/ImageGalleryCard'
import NoData from './components/NoData'
import type { ProjectReport } from './types'

type Kind = 'base' | 'variant'

interface FlattenedItem {
  imageName: string
  displayName: string
  kind: Kind
  icon?: string
  description?: string
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
        description: img.description,
        platforms: img.platforms || [],
        tagCount: img.tags.length,
      })

      if (img.variants) {
        img.variants.forEach(variant => {
          items.push({
            imageName: img.name,
            description: img.description,
            displayName: `${img.name}${variant.tagSuffix}`,
            kind: 'variant',
            icon: variant.report?.icon,
            platforms: variant.platforms || img.platforms || [],
            tagCount: variant.tags.length,
          })
        })
      }
    })

    const lowerSearchTerm = search.toLowerCase()
    return items.filter(item =>
      item.displayName.toLowerCase().includes(lowerSearchTerm) ||
      item.imageName.toLowerCase().includes(lowerSearchTerm) ||
      (item.description?.toLowerCase().includes(lowerSearchTerm) ?? false)
    )
  }, [data.images, search])

  return (
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
            <NoData message="No images found" />
          ) : (
            flattenedItems.map((item, idx) => (
              <ImageGalleryCard
                key={`${item.imageName}-${item.kind}-${idx}`}
                imageName={item.imageName}
                displayName={item.displayName}
                kind={item.kind}
                icon={item.icon}
                description={item.description}
                tagCount={item.tagCount}
                platforms={item.platforms}
                searchTerm={search}
              />
            ))
          )}
        </div>
    </div>
  )
}

export default App
