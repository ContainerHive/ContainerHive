import { useState } from 'react'
import type { ImageReport } from '../types'

interface ImageCardProps {
  image: ImageReport
}

function ImageCard({ image }: ImageCardProps) {
  const [selectedTag, setSelectedTag] = useState<string | null>(null)

  const selectedTagData = image.tags.find(t => t.name === selectedTag)

  const handleTagClick = (tagName: string) => {
    setSelectedTag(selectedTag === tagName ? null : tagName)
  }

  return (
    <div className="image-card">
      <div className="image-name">{image.name}</div>
      <div className="tags">
        {image.tags.map(tag => (
          <span
            key={tag.name}
            className={`tag ${selectedTag === tag.name ? 'selected' : ''}`}
            onClick={() => handleTagClick(tag.name)}
          >
            {tag.name}
            {tag.hasSbom && <span className="sbom-badge" style={{ marginLeft: 6 }}>SBOM</span>}
          </span>
        ))}
      </div>
      <div className="platforms">
        {image.tags[0]?.platforms.map(p => (
          <span key={p.platform} style={{ marginRight: 15 }}>
            {p.platform}: {(p.size / 1024 / 1024).toFixed(2)} MB
          </span>
        ))}
      </div>

      {selectedTagData && (
        <div className="tag-details">
          <h3>Tag: {selectedTagData.name}</h3>
          <p><strong>Platforms:</strong> {selectedTagData.platforms.map(p => p.platform).join(', ')}</p>
          <p><strong>Digest:</strong> {selectedTagData.platforms[0]?.digest || 'N/A'}</p>
          {selectedTagData.hasSbom && selectedTagData.sbom && (
            <div className="sbom-section">
              <h4>SBOM ({selectedTagData.sbom.length} packages)</h4>
              <table className="sbom-table">
                <thead>
                  <tr>
                    <th>Package</th>
                    <th>Version</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedTagData.sbom.filter(pkg => pkg.version).map((pkg, i) => (
                    <tr key={i}>
                      <td>{pkg.name}</td>
                      <td>{pkg.version}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default ImageCard