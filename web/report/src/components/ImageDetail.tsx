import { useState } from 'react'
import { Link } from 'react-router-dom'
import ThemeToggle from './ThemeToggle'
import type { ImageReport } from '../types'
import logo from '../logo.png'

interface ImageDetailProps {
  data: {
    images: ImageReport[]
    source: string
  }
  imageName?: string
  kind?: string
}

function ImageDetail({ data, imageName, kind }: ImageDetailProps) {
  const [activeTag, setActiveTag] = useState<string>('')
  const [sbomSearch, setSbomSearch] = useState('')

  const image = data.images.find(img => img.name === imageName)
  const isVariant = !!kind && kind !== 'base'
  const selectedImage = isVariant && kind ? image?.variants?.find(v => v.name === kind) : null
  const tags = selectedImage ? selectedImage.tags : image?.tags || []
  const displayName = isVariant && selectedImage ? selectedImage.name : (image?.name || '')

  const firstTag = tags[0]?.name || ''

  const currentTag = activeTag || firstTag

  const currentTagData = tags.find(t => t.name === currentTag)

  if (!image) {
    return (
      <>
        <header className="page-header">
          <div className="header-content">
            <div className="header-title">
              <img src={logo} alt="Logo" className="logo-icon" />
              <h1>Image Not Found</h1>
            </div>
            <div className="header-right">
              <ThemeToggle />
            </div>
          </div>
        </header>
        <div className="container">
<Link href="/" className="back-link">← Back to Gallery</Link>
          <div className="no-data">Image not found</div>
        </div>
      </>
    )
  }

  return (
    <>
      <header className="page-header">
        <div className="header-content">
          <div className="header-title">
            <img src={logo} alt="Logo" className="logo-icon" />
            <h1>{displayName}</h1>
          </div>
          <div className="header-right">
            <ThemeToggle />
          </div>
        </div>
      </header>
      <div className="container">
<Link to="/" className="back-link">← Back to Gallery</Link>

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

{currentTagData && (
          <div className="section">
            {image.versions && Object.keys(image.versions).length > 0 && (
              <div className="versions-section">
                <p><strong>Versions</strong></p>
                <ul>
                  {Object.entries(image.versions).map(([key, value]) => (
                    <li key={key}><span className="arg-key">{key}</span>=<span className="arg-value">{value}</span></li>
                  ))}
                </ul>
              </div>
            )}
            {(image.aliases || []).length > 0 && (
              <div className="aliases-section">
                <p><strong>Aliases</strong></p>
                <ul>
                  {image.aliases?.map(alias => (
                    <li key={alias}><span className="alias-tag">{alias}</span></li>
                  ))}
                </ul>
              </div>
            )}
            {isVariant && selectedImage && (selectedImage.aliases || []).length > 0 && (
              <div className="aliases-section">
                <p><strong>Aliases</strong></p>
                <ul>
                  {selectedImage.aliases?.map(alias => (
                    <li key={alias}><span className="alias-tag">{alias}</span></li>
                  ))}
                </ul>
              </div>
            )}
            {(() => {
              const allArgs = (currentTagData.platforms || []).filter(p => p !== null).flatMap(p => Object.entries(p.buildArgs || {}))
              if (allArgs.length === 0) return null
              
              const mergedArgs: Record<string, string> = {}
              for (const [key, value] of allArgs) {
                if (!mergedArgs[key]) mergedArgs[key] = value
              }
              
              return (
                <div className="build-args-section">
                  <p><strong>Build Args</strong></p>
                  <ul>
                    {Object.entries(mergedArgs).map(([key, value]) => (
                      <li key={key}><span className="arg-key">{key}</span>=<span className="arg-value">{value}</span></li>
                    ))}
                  </ul>
                </div>
              )
            })()}
            <h2>Platforms</h2>
            <div className="platforms-grid">
              {(currentTagData.platforms || []).map(platform => {
                if (!platform) return null
                const filteredSbom = platform.sbom
                  ?.filter(pkg => pkg.version)
                  .filter(pkg => {
                    if (!sbomSearch) return true
                    const search = sbomSearch.toLowerCase()
                    return pkg.name.toLowerCase().includes(search) ||
                      (pkg.version && pkg.version.toLowerCase().includes(search))
                  })
                  .sort((a, b) => a.name.localeCompare(b.name)) || []
                return (
                  <div key={platform.platform} className="platform-card">
                    <div className="platform-card-header">
                      <h3>
                        <svg className="processor-icon" viewBox="0 0 1024 1024" width="14" height="14">
                          <path fill="currentColor" d="M356.571429 804.571429v128h-18.285715v-128a109.714286 109.714286 0 0 1-109.348571-100.571429H109.714286v-18.285714h118.857143v-91.428572H109.714286v-18.285714h118.857143v-91.428571H109.714286v-18.285715h118.857143v-91.428571H109.714286v-18.285714h119.222857A109.714286 109.714286 0 0 1 338.285714 256V128h18.285715v128h91.428571V128h18.285714v128h91.428572V128h18.285714v128h91.428571V128h18.285715v129.517714a109.769143 109.769143 0 0 1 91.062857 99.053715H914.285714v18.285714h-137.142857v91.428571H914.285714v18.285715h-137.142857v91.428571H914.285714v18.285714h-137.142857v91.428572H914.285714v18.285714h-137.508571a109.769143 109.769143 0 0 1-91.062857 99.053714V932.571429h-18.285715v-128h-91.428571v128h-18.285714v-128h-91.428572v128h-18.285714v-128h-91.428571zM256 365.714286v329.142857a82.285714 82.285714 0 0 0 82.285714 82.285714h329.142857A82.285714 82.285714 0 0 0 749.714286 694.857143V365.714286a82.285714 82.285714 0 0 0-82.285715-82.285715h-329.142857A82.285714 82.285714 0 0 0 256 365.714286z m64 256V438.857143a91.428571 91.428571 0 0 1 91.428571-91.428572h182.857143a91.428571 91.428571 0 0 1 91.428572 91.428572v182.857143a91.428571 91.428571 0 0 1-91.428572 91.428571h-182.857143a91.428571 91.428571 0 0 1-91.428571-91.428571z m18.285714 0a73.142857 73.142857 0 0 0 73.142857 73.142857h182.857143a73.142857 73.142857 0 0 0 73.142857-73.142857V438.857143a73.142857 73.142857 0 0 0-73.142857-73.142857h-182.857143a73.142857 73.142857 0 0 0-73.142857 73.142857v182.857143z"/>
                        </svg>
                        {platform.platform}
                      </h3>
                      <span className="size-badge">{(platform.size / 1024 / 1024).toFixed(2)} MB</span>
                    </div>
                    <p className="digest"><strong>Digest:</strong> {platform.digest}</p>

                    {platform.hasSbom && platform.sbom && (
                      <div className="platform-sbom">
                        <h4>SBOM ({filteredSbom.length} packages)</h4>
                        <div className="sbom-search">
                          <input
                            type="text"
                            placeholder="Search packages..."
                            value={sbomSearch}
                            onChange={(e) => setSbomSearch(e.target.value)}
                          />
                        </div>
                        <div className="sbom-scroll">
                          <table className="sbom-table">
                            <thead>
                              <tr>
                                <th>Package</th>
                                <th>Version</th>
                              </tr>
                            </thead>
                            <tbody>
                              {filteredSbom.map((pkg, i) => (
                                <tr key={i}>
                                  <td>{pkg.name}</td>
                                  <td>{pkg.version}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      </div>
                    )}

                    {!platform.hasSbom && (
                      <p className="no-data">No SBOM available</p>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>
    </>
  )
}

export default ImageDetail