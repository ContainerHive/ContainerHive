import type {ProjectReport} from './types'

export const mockData: ProjectReport = {
    generatedAt: new Date().toISOString(),
    source: 'tar',
    images: [
        {
            name: 'alpine',
            description: 'Minimal Docker image based on Alpine Linux with a focus on security and small footprint.',
            platforms: ['linux/amd64', 'linux/arm64'],
            tags: [
                {
                    name: '3.19',
                    buildArgs: {FOO: 'bar'},
                    versions: {distribution_version: '1.0', os_version: '3.19.1'},
                    platforms: [
                        {
                            platform: 'linux/amd64',
                            sbom: [
                                {name: 'alpine-baselayout', version: '3.4.0'},
                                {name: 'busybox', version: '1.36.1'},
                                {name: 'ca-certificates-bundle', version: '20240226'},
                            ],
                        },
                        {
                            platform: 'linux/arm64',
                            sbom: [
                                {name: 'alpine-baselayout', version: '3.4.0'},
                                {name: 'busybox', version: '1.36.1'},
                            ],
                        },
                    ],
                },
                {
                    name: '3.18',
                    buildArgs: {FOO: 'bar'},
                    versions: {distribution_version: '1.0', os_version: '3.18.0'},
                    platforms: [
                        {
                            platform: 'linux/amd64',
                            sbom: [
                                {name: 'alpine-baselayout', version: '3.4.0'},
                                {name: 'busybox', version: '1.38.1-amd'},
                                {name: 'ca-certificates-bundle', version: '20240226'},
                            ],
                        },
                        {
                            platform: 'linux/arm64',
                            sbom: [
                                {name: 'alpine-baselayout', version: '3.4.0'},
                                {name: 'busybox', version: '1.38.1-arm'},
                            ],
                        },
                    ],
                },
            ],
            variants: [
                {
                    name: 'node',
                    tagSuffix: '-node',
                    platforms: ['linux/amd64'],
                    report: {icon: 'devicon-nodejs-plain-wordmark'},
                    tags: [
                        {
                            name: '3.19-node20',
                            buildArgs: {NODE_ENV: 'production'},
                            versions: {distribution_version: '1.0', os_version: '3.18.0', node_version: '22'},
                            platforms: [
                                {
                                    platform: 'linux/amd64',
                                    sbom: [
                                        {name: 'node', version: '20.11.0'},
                                        {name: 'icu', version: '73.2'},
                                    ],
                                },
                            ],
                        },
                    ],
                },
            ],
        },
        {
            name: 'nginx',
            description: 'Web server and reverse proxy with built-in load balancing and caching capabilities.',
            report: {icon: 'devicon-nginx-original'},
            platforms: ['linux/amd64'],
            tags: [
                {
                    name: '1.25',
                    platforms: [
                        {
                            platform: 'linux/amd64',
                            sbom: [
                                {name: 'nginx', version: '1.25.4'},
                                {name: 'openssl', version: '3.3.1'},
                                {name: 'pcre2', version: '10.44'},
                            ],
                        },
                    ],
                },
            ],
        },
        {
            name: 'node',
            platforms: ['linux/amd64', 'linux/arm64'],
            tags: [
                {
                    name: '20',
                    platforms: [
                        {
                            platform: 'linux/amd64',
                            sbom: [
                                {name: 'node', version: '20.11.0'},
                                {name: 'icu', version: '73.2'},
                                {name: 'c-ares', version: '1.26.0'},
                            ],
                        },
                        {
                            platform: 'linux/arm64',
                            sbom: [
                                {name: 'node', version: '20.11.0'},
                                {name: 'icu', version: '73.2'},
                            ],
                        },
                    ],
                },
            ],
        },
    ],
}

export function getReportData(): ProjectReport {
    if (typeof window !== 'undefined' && window.__REPORT_DATA__) {
        return window.__REPORT_DATA__
    }
    return mockData
}
