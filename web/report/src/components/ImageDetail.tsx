import { useState } from 'react'
import { Link } from 'react-router-dom'
import type { ImageReport } from '../types'
import NoData from './NoData'

interface ImageDetailProps {
  data: {
    images: ImageReport[]
    source: string
  }
  imageName?: string
  kind?: string
}

function ImageDetail({ data, imageName, kind }: Readonly<ImageDetailProps>) {
  const image = data.images.find(img => img.name === imageName)
  const isVariant = !!kind && kind !== 'base'
  const selectedVariant = isVariant && kind ? image?.variants?.find(v => v.name === kind) : null
  const tags = selectedVariant ? selectedVariant.tags : image?.tags || []
  const displayName = isVariant && selectedVariant ? `${image?.name}${selectedVariant.tagSuffix}` : (image?.name || '')

  const [activeTag, setActiveTag] = useState<string>('')
  const [sbomSearch, setSbomSearch] = useState<string>('')

  const firstTag = tags[0]?.name || ''
  const currentTag = activeTag || firstTag
  const currentTagData = tags.find(t => t.name === currentTag)

  const buildArgs = currentTagData?.buildArgs || {}
  const versions = currentTagData?.versions || {}

  if (!image) {
    return (
      <div className="container">
        <Link to="/" className="back-link">← Back to Gallery</Link>
        <div className="no-data">Image not found</div>
      </div>
    )
  }

  const currentPlatforms = currentTagData?.platforms || []
  const allSbom = currentPlatforms.flatMap(p => p.sbom || [])

  const tagBuildArgs = currentTagData?.buildArgs || {}
  const mergedBuildArgs = { ...buildArgs, ...tagBuildArgs }

  return (
    <div className="container">
      <Link to="/" className="back-link">← Back to Gallery</Link>

        {image.description && (
          <div className="description-panel">
            {image.description}
          </div>
        )}

        <div className="section">
          <h2>Tags</h2>
          <div className="tabs">
            {tags.map(tag => (
              <button
                key={tag.name}
                className={`tab ${currentTag === tag.name ? 'active' : ''}`}
                onClick={() => {
                  setActiveTag(tag.name)
                  setSbomSearch('')
                }}
              >
                {tag.name}
              </button>
            ))}
          </div>
        </div>

        <div className="section">
          {Object.keys(versions).length > 0 && (
            <div className="versions-section">
              <h2>Versions</h2>
              <ul>
                {Object.entries(versions).map(([key, value]) => (
                  <li key={key}><span className="arg-key">{key}</span>=<span className="arg-value">{value}</span></li>
                ))}
              </ul>
            </div>
          )}

          {Object.keys(mergedBuildArgs).length > 0 && (
            <div className="build-args-section">
              <h2>Build Args</h2>
              <ul>
                {Object.entries(mergedBuildArgs).map(([key, value]) => (
                  <li key={key}><span className="arg-key">{key}</span>=<span className="arg-value">{value}</span></li>
                ))}
              </ul>
            </div>
          )}

          <h2>Platforms</h2>
          {allSbom.length > 0 && (
            <div className="sbom-search">
              <input
                type="text"
                placeholder="Search packages..."
                value={sbomSearch}
                onChange={(e) => setSbomSearch(e.target.value)}
              />
            </div>
          )}
          <div className="platforms-grid">
            {currentPlatforms.map(plat => {
              const platSbom = plat.sbom || []
              const displaySbom = sbomSearch
                ? platSbom.filter(pkg =>
                    pkg.name.toLowerCase().includes(sbomSearch.toLowerCase()) ||
                    (pkg.version && pkg.version.toLowerCase().includes(sbomSearch.toLowerCase()))
                  )
                : platSbom
              return (
                <div key={plat.platform} className="platform-card">
                  <div className="platform-card-header">
                    <h3>
                      <svg className="processor-icon" viewBox="0 0 1024 1024" width="14" height="14">
                        <path fill="currentColor" d="M356.571429 804.571429v128h-18.285715v-128a109.714286 109.714286 0 0 1-109.348571-100.571429H109.714286v-18.285714h118.857143v-91.428572H109.714286v-18.285714h118.857143v-91.428571H109.714286v-18.285715h118.857143v-91.428571H109.714286v-18.285714h119.222857A109.714286 109.714286 0 0 1 338.285714 256V128h18.285715v128h91.428571V128h18.285714v128h91.428572V128h18.285714v128h91.428571V128h18.285715v129.517714a109.769143 109.769143 0 0 1 91.062857 99.053715H914.285714v18.285714h-137.142857v91.428571H914.285714v18.285715h-137.142857v91.428571H914.285714v18.285714h-137.142857v91.428572H914.285714v18.285714h-137.508571a109.769143 109.769143 0 0 1-91.062857 99.053714V932.571429h-18.285715v-128h-91.428571v128h-18.285714v-128h-91.428572v128h-18.285714v-128h-91.428571zM256 365.714286v329.142857a82.285714 82.285714 0 0 0 82.285714 82.285714h329.142857A82.285714 82.285714 0 0 0 749.714286 694.857143V365.714286a82.285714 82.285714 0 0 0-82.285715-82.285715h-329.142857A82.285714 82.285714 0 0 0 256 365.714286z m64 256V438.857143a91.428571 91.428571 0 0 1 91.428571-91.428572h182.857143a91.428571 91.428571 0 0 1 91.428572 91.428572v182.857143a91.428571 91.428571 0 0 1-91.428572 91.428571h-182.857143a91.428571 91.428571 0 0 1-91.428571-91.428571z m18.285714 0a73.142857 73.142857 0 0 0 73.142857 73.142857h182.857143a73.142857 73.142857 0 0 0 73.142857-73.142857V438.857143a73.142857 73.142857 0 0 0-73.142857-73.142857h-182.857143a73.142857 73.142857 0 0 0-73.142857 73.142857v182.857143z"/>
                      </svg>
                      {plat.platform}
                    </h3>
                    {platSbom.length > 0 && (
                      <span className="sbom-badge">SBOM</span>
                    )}
                  </div>
                  {displaySbom.length > 0 ? (
                    <div className="platform-sbom">
                      <p className="sbom-count">{displaySbom.length} packages</p>
                      <div className="sbom-scroll">
                        <table className="sbom-table">
                          <thead>
                            <tr>
                              <th>Package</th>
                              <th>Version</th>
                            </tr>
                          </thead>
                          <tbody>
                            {displaySbom.map((pkg, i) => (
                              <tr key={i}>
                                <td>{pkg.name}</td>
                                <td>{pkg.version || '-'}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  ) : platSbom.length > 0 ? (
                    <NoData message="No matches" />
                  ) : (
                    <NoData message="No SBOM available" />
                  )}
                </div>
              )
            })}
          </div>
        </div>
    </div>
  )
}

export default ImageDetail
