import type { ProjectReport } from './types'

export const mockData: ProjectReport = {
  generatedAt: new Date().toISOString(),
  source: 'tar',
  images: [
{
      name: 'alpine',
      icon: 'docker',
      versions: { distribution: 'alpine', os_version: '3.19.1' },
      aliases: ['docker.io/alpine', 'ghcr.io/alpine'],
      tags: [
                {
                  name: '3.19',
                  platforms: [
                    {
                      platform: 'linux-amd64',
                      digest: 'sha256:abc123def456',
                      size: 5242880,
                      hasSbom: true,
                      sbom: [
                        { name: 'alpine-baselayout', version: '3.4.0' },
                        { name: 'busybox', version: '1.36.1' },
                      ],
                      buildArgs: { BUILDKIT_TEMPLATE_UBI: 'ubi8' }
                    },
                    {
                      platform: 'linux-arm64',
                      digest: 'sha256:def456ghi789',
                      size: 4980736,
                      hasSbom: true,
                      sbom: [
                        { name: 'alpine-baselayout', version: '3.4.0' },
                      ],
                      buildArgs: { BUILDKIT_TEMPLATE_UBI: 'ubi8-arm64' }
                    }
                  ]
                }
              ],
      variants: [
        {
          name: 'alpine-node',
          tagSuffix: '-node',
          tags: [
            {
              name: '3.19-node20',
              platforms: [
                {
                  platform: 'linux-amd64',
                  digest: 'sha256:var1digest',
                  size: 180000000,
                  hasSbom: true,
                  sbom: [
                    { name: 'node', version: '20.11.0' },
                  ]
                }
              ]
            }
          ]
        }
      ]
    },
    {
      name: 'nginx',
      icon: 'nginx',
      tags: [
        {
          name: '1.25',
          platforms: [
            {
              platform: 'linux-amd64',
              digest: 'sha256:mno345pqr678',
              size: 142458880,
              hasSbom: true,
              sbom: [
                { name: 'nginx', version: '1.25.4' },
              ]
            }
          ]
        }
      ]
    },
    {
      name: 'node',
      tags: [
        {
          name: '20',
          platforms: [
            {
              platform: 'linux-amd64',
              digest: 'sha256:stu901vwx234',
              size: 178257920,
              hasSbom: true,
              sbom: [
                { name: 'node', version: '20.11.0' },
              ]
            }
          ]
        }
      ]
    },
    {
      name: 'ubuntu',
      tags: Array.from({ length: 20 }, (_, i) => ({
        name: `24.04-${i + 1}`,
        platforms: [
          {
            platform: 'linux-amd64',
            digest: `sha256:ubuntu${String(i).padStart(3, '0')}`,
            size: 80000000 + i * 1000000,
            hasSbom: true,
            sbom: [
              { name: 'base-files', version: '24.04' },
              { name: 'bash', version: '5.2' },
              { name: 'coreutils', version: '9.4' },
            ]
          },
          {
            platform: 'linux-arm64',
            digest: `sha256:ubuntu${String(i).padStart(3, '0')}+arm`,
            size: 78000000 + i * 1000000,
            hasSbom: false,
            sbom: []
          }
        ]
      }))
    }
  ]
}

export function getReportData(): ProjectReport {
  if (typeof window !== 'undefined' && window.__REPORT_DATA__) {
    return window.__REPORT_DATA__
  }
  return mockData
}