import { Link } from 'react-router-dom'

type Kind = 'base' | 'variant'

interface ImageGalleryCardProps {
  imageName: string
  displayName: string
  kind: Kind
  icon?: string
  description?: string
  tagCount: number
  platforms: string[]
}

function ImageGalleryCard({
  imageName,
  displayName,
  kind,
  icon,
  description,
  tagCount,
  platforms,
}: Readonly<ImageGalleryCardProps>) {
  return (
    <Link
      to={`/image/${encodeURIComponent(imageName)}/${kind}`}
      className="image-card"
    >
      <div className={`card-kind-badge ${kind}`}>
        {kind === 'base' ? 'Base' : 'Variant'}
      </div>
      <div className="card-header">
        <div className="card-icon">
          {icon ? (
            <i className={icon}></i>
          ) : (
            <span>📦</span>
          )}
        </div>
        <div className="image-name">{displayName}</div>
      </div>
      {description && (
        <div className="image-description">{description}</div>
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
