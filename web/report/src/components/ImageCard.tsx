import type { ImageReport } from '../types'

interface ImageCardProps {
  image: ImageReport
}

function ImageCard({ image }: ImageCardProps) {
  return (
    <div className="image-card">
      <div className="image-name">{image.name}</div>
      <div className="tags">
        {image.tags.map(tag => (
          <span key={tag.name} className="tag">
            {tag.name}
          </span>
        ))}
      </div>
      <div className="platforms">
        {(image.platforms || []).map(platform => (
          <span key={platform} style={{ marginRight: 15 }}>
            {platform}
          </span>
        ))}
      </div>
    </div>
  )
}

export default ImageCard
