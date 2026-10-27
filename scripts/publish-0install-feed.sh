#!/usr/bin/env bash
set -euo pipefail

# Generates the 0install feed for a release version and publishes it to the
# dedicated feed repository (ContainerHive/0install, gh-pages branch).
#
# Usage: scripts/publish-0install-feed.sh <version>
# The version is the release version without the leading v, for example 1.2.0.
#
# Requires: 0template, gpg, python3, git. Expects the context variables
# GITHUB_TOKEN and GPG_KEY (base64 encoded armored private key without a
# passphrase; CircleCI context variables cannot hold multi-line values,
# so the armored key text is decoded with base64 first).
# The run must start after the release archives are uploaded, because
# 0template downloads them to compute the manifest digests.
# The merge and signature come from scripts/merge-feed.py, because the
# Python 0publish cannot sign with current Python versions (0publish issue 5).

cd "$(dirname "$0")/.."

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <version>"
  echo "The version is the release version without the leading v, for example 1.2.0."
  exit 1
fi

version="$1"
released="$(date -u +%F)"
feed_owner=ContainerHive
feed_repo=0install
feed_domain=0install.container-hive.eu
feed_base="https://${feed_domain}"
feed_url="${feed_base}/containerhive.xml"
feed_repo_url="https://x-access-token:${GITHUB_TOKEN}@github.com/${feed_owner}/${feed_repo}.git"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Import the release key. The key must have no passphrase for unattended use.
if [[ $GPG_KEY == -----BEGIN* ]]; then
  printf '%s\n' "$GPG_KEY" | gpg --batch --import
else
  printf '%s' "$GPG_KEY" | base64 --decode | gpg --batch --import
fi
key_id="$(gpg --list-secret-keys --with-colons | awk -F: '/^sec/ { print $5; exit }')"
if [[ -z "$key_id" ]]; then
  echo "error: no signing key found after GPG import" >&2
  exit 1
fi
echo "Signing feeds with key $key_id"

# Generate the feed for the new version. 0template downloads the four release
# archives from the GitHub release and fills in sizes, digests and IDs.
feed_dir="$work/feed"
mkdir "$feed_dir"
cp packaging/0install/containerhive.xml.template "$feed_dir/"
(cd "$feed_dir" && 0template containerhive.xml.template version="$version" released="$released")
generated="$feed_dir/containerhive-$version.xml"

# Check out the feed branch. The first run starts a new branch.
if git clone --branch gh-pages "$feed_repo_url" "$work/feedpages" 2>/dev/null; then
  master="$work/feedpages/containerhive.xml"
else
  echo "No gh-pages branch found in ${feed_owner}/${feed_repo}. Creating a new one."
  rm -rf "$work/feedpages"
  mkdir "$work/feedpages"
  git -C "$work/feedpages" init -q
  git -C "$work/feedpages" symbolic-ref HEAD refs/heads/gh-pages
  git -C "$work/feedpages" remote add origin "$feed_repo_url"
  master=""
fi

if [[ -f "$master" ]]; then
  # Merge the new version into the master feed and re-sign it. --canonical
  # re-points the feed-for element of an existing master feed to the current
  # feed URL, so a URL change survives the next merge.
  python3 scripts/merge-feed.py --canonical "$feed_url" "$master" "$generated" "$key_id"
else
  # The generated feed becomes the first version of the master feed.
  # merge-feed.py signs it and the public key is exported next to it below,
  # so 0install clients can fetch the key from the same directory.
  python3 scripts/merge-feed.py --init "$feed_url" --canonical "$feed_url" "$work/feedpages/containerhive.xml" "$generated" "$key_id"
fi

# Keep the old local versions findable: 0install resolves a version check
# against the master feed only, so per-version side feeds are not required.

# Serve the feed branch from the custom domain. GitHub Pages redirects the
# containerhive.github.io URL to it.
echo "$feed_domain" > "$work/feedpages/CNAME"

# Publish the icon next to the feed. The icon shows up in 0install launchers.
cp docs/images/logo.png "$work/feedpages/containerhive.png"

# Store the public key next to the feed, so 0install clients can fetch the
# key from the same directory when they import the feed.
gpg --export "$key_id" > "$work/feedpages/$key_id.gpg"

# Configure the CI identity for the commit. The push token is private and
# scoped, so the commit author stays the configured private user.
git -C "$work/feedpages" config user.name "CircleCI"
git -C "$work/feedpages" config user.email "no-reply@timo-reymann.de"
git -C "$work/feedpages" add -A
git -C "$work/feedpages" commit -q -m "chore: Publish feed for ${version} [skip ci]"
git -C "$work/feedpages" push -q origin gh-pages

echo "Published feed: ${feed_url}"
