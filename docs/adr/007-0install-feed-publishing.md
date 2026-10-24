---
id: 7
status: accepted
date: 2026-10-25
---

# Publish the 0install feed to a dedicated repository

## Context and Problem Statement

ContainerHive binaries come from GitHub release archives. An alternative install channel is a signed 0install feed, so
`0install add ch <feed-url>` gives users automatic updates. The feed must be generated on a release, because 0template
downloads the release archives to compute the manifest digests. The docs site publishes from the `gh-pages` branch with
`npx gh-pages`, and that tool replaces the whole branch content on every run. A feed on the docs site would be wiped by
the next docs build on `main` and would stay unusable until the next release re-added it.

Should the feed live on the docs site or somewhere else?

## Decision Drivers

* The feed can only be generated from a tag pipeline, not from the docs build on `main`
* The docs publish replaces the full `gh-pages` tree, so it destroys any file it does not know about
* 0install clients need a stable, always available feed URL, because they fetch it on every update check
* A second publish mechanism (merge-style git push) in the docs job would couple two release cycles to one branch

## Considered Options

* Option 1: Host the feed on the docs `gh-pages` branch and change the docs publish to a merge-style push that keeps unknown files
* Option 2: Host the feed in a dedicated repository with its own `gh-pages` branch

## Decision Outcome

Chosen option: "Option 2 — Dedicated feed repository", because it keeps every change isolated. The docs pipeline stays
as it is, the feed pipeline owns its branch, and the feed URL stays valid between releases. The cost is a second
repository and a second URL to document.

Components:

* `packaging/0install/containerhive.xml.template` describes the Linux and macOS archives from the GitHub release
* `scripts/publish-0install-feed.sh` runs `0template`, merges the version into the master feed, and signs it
* A CircleCI `publish-0install-feed` job runs it on tags after the artifacts are attached to the release

The feed lives in `ContainerHive/0install` and is served at
`https://containerhive.github.io/0install/containerhive.xml`. Installation and updates need no bootstrapper; users
install through `0install add ch <feed-url>` and trust the release key once.
