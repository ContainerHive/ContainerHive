import { Link } from 'react-router-dom'
import { highlightText } from '../utils/highlight'

type Kind = 'base' | 'variant'

interface ImageGalleryCardProps {
  imageName: string
  displayName: string
  kind: Kind
  icon?: string
  description?: string
  tagCount: number
  platforms: string[]
  searchTerm?: string
}

function ImageGalleryCard({
  imageName,
  displayName,
  kind,
  icon,
  description,
  tagCount,
  platforms,
  searchTerm,
}: Readonly<ImageGalleryCardProps>) {
  const highlightedDisplayName = highlightText(displayName, searchTerm || '')
  const highlightedDescription = description ? highlightText(description, searchTerm || '') : null

  return (
    <Link
      to={`/image/${encodeURIComponent(imageName)}/${kind === 'base' ? 'base' : encodeURIComponent(displayName)}`}
      className="image-card"
    >
      <div className={`card-kind-badge ${kind}`}>
        {kind === 'base' ? 'Base' : displayName}
      </div>
      <div className="card-header">
        <div className="card-icon">
          {icon ? (
            <i className={icon}></i>
          ) : (
            <span>📦</span>
          )}
        </div>
        <div className="image-name">{highlightedDisplayName}</div>
      </div>
      {highlightedDescription && (
        <div className="image-description">{highlightedDescription}</div>
      )}
      <div className="image-meta">
        <span><span className="tag-icon"></span> {tagCount} tag{tagCount !== 1 ? 's' : ''}</span>
      </div>
      <div className="platforms-list">
        {platforms.map(platform => (
          <span key={platform} className="platform-badge">{platform}</span>
        ))}
      </div>
    </Link>
  )
}

export default ImageGalleryCard
