#!/usr/bin/env bash
#
# release-snapshot.sh — build and push a snapshot image from a pull request,
# a git ref, or the current HEAD.
#
# The Go build runs inside a docker container, artifacts are written back to
# the host, and the resulting image is built for amd64 and arm64 and pushed
# to the target registry.
#
# Usage:
#   ./hack/release-snapshot.sh [--ref <git-ref>] [--pr <id>] \
#       --target <registry.tld/image> [--tag <tag>] [--no-push]
#
# Examples:
#   ./hack/release-snapshot.sh --pr 42 --target registry.tld/containerhive
#   ./hack/release-snapshot.sh --ref HEAD --target ghcr.io/containerhive/containerhive
#   ./hack/release-snapshot.sh --target registry.tld/containerhive --tag 1.4.2-internal.3
#   ./hack/release-snapshot.sh            # interactive prompts
#
# Requires: bash 3+, git, docker with buildx. Registry credentials are up to
# the caller (docker login, credential helpers, or a company tool).
set -e

TOOLKIT_VERSION="1.10.3"
BUILDER_IMAGE="${CH_SNAPSHOT_BUILDER_IMAGE:-timoreymann/ci-go:1.26}"
WEB_IMAGE="${CH_SNAPSHOT_WEB_IMAGE:-node:24-bookworm}"
BUILDX_BUILDER_NAME="containerhive-snapshot"
PLATFORMS="linux/amd64,linux/arm64"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
source "$ROOT/hack/lib/bash-tui-toolkit.bash"

HOST_UID="$(id -u)"
HOST_GID="$(id -g)"

# Deletes a directory on the host; falls back to a container when a previous
# container run wrote root-owned files into it.
purge_dir() {
    rm -rf "$1" 2>/dev/null || {
        docker run --rm \
            --user root \
            -v "${1%/*}:/purge" \
            "$BUILDER_IMAGE" \
            rm -rf "/purge/${1##*/}" >/dev/null 2>&1
    }
}

usage() {
    sed -n '3,16p' "$0" | sed 's/^# \{0,1\}//'
    exit 1
}

log "LOG_INFO" "Starting release-snapshot"

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
REF=""
PR_ID=""
TARGET=""
TAG=""
NO_PUSH=0

while [ $# -gt 0 ]; do
    case "$1" in
        --ref) REF="$2"; shift 2 ;;
        --pr) PR_ID="$2"; shift 2 ;;
        --target) TARGET="$2"; shift 2 ;;
        --tag) TAG="$2"; shift 2 ;;
        --no-push) NO_PUSH=1; shift ;;
        --help | -h) usage ;;
        *)
            show_error "Unknown option: $1"
            usage
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
for cmd in git docker; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        show_error "$cmd is not installed"
        exit 1
    fi
done

if ! docker info >/dev/null 2>&1; then
    show_error "docker daemon is not reachable"
    exit 1
fi

if ! docker buildx version >/dev/null 2>&1; then
    show_error "docker buildx is not installed"
    exit 1
fi

if [ ! -f "$ROOT/hack/lib/bash-tui-toolkit.bash" ]; then
    show_error "bash-tui-toolkit bundle not found in hack/lib"
    exit 1
fi

HOST_ARCH="$(uname -m)"
MISSING_BINFMT=""
case "$HOST_ARCH" in
    x86_64) MISSING_BINFMT="/proc/sys/fs/binfmt_misc/qemu-aarch64" ;;
    aarch64 | arm64) MISSING_BINFMT="/proc/sys/fs/binfmt_misc/qemu-x86_64" ;;
esac
if [ -n "$MISSING_BINFMT" ] && [ ! -e "$MISSING_BINFMT" ]; then
    log "LOG_WARN" "Cross-arch emulation not detected. If the arm64 build fails in the bin stage, run: docker run --privileged --rm tonistiigi/binfmt --install all"
fi

# ---------------------------------------------------------------------------
# Interactive prompts for anything not provided via flags
# ---------------------------------------------------------------------------
INTERACTIVE=1
if ! [ -t 0 ] || ! [ -t 1 ]; then
    INTERACTIVE=0
fi

if [ -z "$REF" ] && [ -z "$PR_ID" ]; then
    if [ "$INTERACTIVE" = "0" ]; then
        show_error "No source given. Use --ref <git-ref> or --pr <id>, or run interactively in a terminal."
        usage
    fi
    options=("Current HEAD" "Pull request" "Other git ref (commit hash, branch, tag)")
    idx=$(list "Build from which source?" "${options[@]}")
    case "$idx" in
        0) REF="HEAD" ;;
        1) PR_ID=$(with_validate 'input "Pull request number"' validate_present) ;;
        2) REF=$(with_validate 'input "Git ref (commit hash, branch or tag)"' validate_present) ;;
    esac
fi

sanitize_tag() {
    printf '%s' "$1" \
        | tr '[:upper:]' '[:lower:]' \
        | sed -E 's/[^a-z0-9._-]+/-/g; s/^[-.]+//; s/^(.{0,127}).*/\1/'
}

sanitize_target() {
    printf '%s' "$1" \
        | tr '[:upper:]' '[:lower:]' \
        | sed -E 's/[^a-z0-9._/-]+/-/g; s/^-+//'
}

case "$TARGET" in
    *:*)
        show_error "Target must not contain a tag. Pass the tag via --tag, e.g. --target registry.tld/image --tag my-tag"
        exit 1
        ;;
esac

if [ -z "$TARGET" ]; then
    if [ "$INTERACTIVE" = "0" ]; then
        show_error "No target image given. Use --target <registry.tld/image>."
        usage
    fi
    TARGET=$(with_validate 'input "Target image without tag (e.g. registry.tld/containerhive)"' validate_present)
fi

if [ -z "$REF" ] && [ -z "$PR_ID" ]; then
    REF="HEAD"
fi

if [ -n "$PR_ID" ]; then
    DEFAULT_TAG="pr-$(sanitize_tag "$PR_ID")"
else
    DEFAULT_TAG="rev-$(git -C "$ROOT" rev-parse --short "$REF" 2>/dev/null || git -C "$ROOT" rev-parse --short HEAD)"
fi

if [ -z "$TAG" ] && [ "$INTERACTIVE" = "1" ]; then
    TAG="$DEFAULT_TAG"
    confirmed=$(confirm "Use the derived tag '$TAG'? (N = enter a custom tag)")
    if [ "$confirmed" = "0" ]; then
        TAG=$(with_validate 'input "Custom tag"' validate_present)
    fi
fi
if [ -z "$TAG" ]; then
    TAG="$DEFAULT_TAG"
fi

TAG="$(sanitize_tag "$TAG")"
if [ -z "$TAG" ]; then
    show_error "Tag sanitized to empty — provide a docker-safe tag"
    exit 1
fi
TARGET="$(sanitize_target "$TARGET")"

if [ "$NO_PUSH" = "0" ] && [ -t 0 ]; then
    if [ -n "$PR_ID" ]; then
        SOURCE_DESC="pull request #$PR_ID"
    else
        SOURCE_DESC="ref $REF ($(git -C "$ROOT" rev-parse --short "$REF" 2>/dev/null || git -C "$ROOT" rev-parse --short HEAD))"
    fi
    confirmed=$(confirm "Build [$PLATFORMS] and push $TARGET:$TAG from $SOURCE_DESC?")
    if [ "$confirmed" = "0" ]; then
        log "LOG_INFO" "Aborted by user"
        exit 1
    fi
fi

# ---------------------------------------------------------------------------
# Resolve the source into a detached worktree so the checkout stays clean
# ---------------------------------------------------------------------------
WORKDIR="$(mktemp -d "${TMPDIR:-/tmp}/containerhive-snapshot.XXXXXX")"
WORKTREE="$WORKDIR/src"
cleanup() {
    git -C "$ROOT" worktree remove --force "$WORKTREE" >/dev/null 2>&1 || {
        purge_dir "$WORKTREE"
        rm -rf "$WORKDIR"
        git -C "$ROOT" worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true
        git -C "$ROOT" worktree prune >/dev/null 2>&1 || true
    }
    purge_dir "$WORKDIR"
}
trap cleanup EXIT

if [ -n "$PR_ID" ]; then
    log "LOG_INFO" "Fetching refs/pull/$PR_ID/head from origin"
    git -C "$ROOT" fetch origin "refs/pull/$PR_ID/head"
    git -C "$ROOT" worktree add --detach "$WORKTREE" FETCH_HEAD >/dev/null 2>&1
else
    if [ "$REF" = "HEAD" ]; then
        REF_SHA="$(git -C "$ROOT" rev-parse HEAD)"
        if [ -n "$(git -C "$ROOT" status --porcelain)" ]; then
            show_error "Working tree has uncommitted changes. Building the committed state at $REF_SHA only."
        fi
    fi
    git -C "$ROOT" worktree add --detach "$WORKTREE" "$REF" 2>/dev/null || {
        show_error "Could not resolve ref '$REF'"
        exit 1
    }
fi

SOURCE_SHA="$(git -C "$WORKTREE" rev-parse --short HEAD)"

NOW="$(date +%Y-%m-%d_%H:%M:%S)"

# ---------------------------------------------------------------------------
# Build the web report first — the Go binary embeds pkg/report/assets
# ---------------------------------------------------------------------------
log "LOG_INFO" "Building web report in $WEB_IMAGE"
docker run --rm \
    --user "$HOST_UID:$HOST_GID" \
    -e "HOME=/tmp/build-home" \
    -v "$WORKTREE:/src" \
    -w /src \
    "$WEB_IMAGE" \
    bash -ec '
        mkdir -p "$HOME"
        export NPM_CONFIG_PREFIX="$HOME/.npm-global"
        npm install -g yarn
        export PATH="$HOME/.npm-global/bin:$PATH"
        cd web/report
        yarn install
        cd ../..
        make build-web-report
    '

# ---------------------------------------------------------------------------
# Compile binaries inside a container, artifacts land on the host
# ---------------------------------------------------------------------------
DIST_SNAPSHOT="${DIST_SNAPSHOT_DIR:-$ROOT/dist-snapshot}"
case "$DIST_SNAPSHOT" in /*) ;; *) DIST_SNAPSHOT="$ROOT/$DIST_SNAPSHOT" ;; esac
OUT_DIR="$DIST_SNAPSHOT/$TAG"
OUT_DIST="$OUT_DIR/dist"
purge_dir "$OUT_DIR"
mkdir -p "$OUT_DIST"

log "LOG_INFO" "Building binaries in $BUILDER_IMAGE for $SOURCE_SHA"
docker run --rm \
    --user "$HOST_UID:$HOST_GID" \
    -e "HOME=/tmp/build-home" \
    -e "GOPATH=/tmp/gopath" \
    -e "GOMODCACHE=/tmp/gopath/pkg/mod" \
    -e "GOCACHE=/tmp/gocache" \
    -e "VERSION=$TAG" \
    -e "COMMIT_REF=$SOURCE_SHA" \
    -e "NOW=$NOW" \
    -v "$WORKTREE:/src" \
    -v "$OUT_DIST:/src/dist" \
    -w /src \
    "$BUILDER_IMAGE" \
    make build-linux pack

if [ ! -f "$OUT_DIST/linux-amd64.tar.zst" ] || [ ! -f "$OUT_DIST/linux-arm64.tar.zst" ]; then
    show_error "Expected archives are missing in $OUT_DIST"
    exit 1
fi
show_success "Binaries built: $OUT_DIST"

# ---------------------------------------------------------------------------
# Assemble the docker build context (Dockerfile, licenses, dist)
# ---------------------------------------------------------------------------
cp "$WORKTREE/Dockerfile" "$WORKTREE/LICENSE" "$WORKTREE/NOTICE" "$OUT_DIR/"

# ---------------------------------------------------------------------------
# Build and push the image for both architectures
# ---------------------------------------------------------------------------
if ! docker buildx inspect "$BUILDX_BUILDER_NAME" >/dev/null 2>&1; then
    log "LOG_INFO" "Creating buildx builder $BUILDX_BUILDER_NAME"
    docker buildx create --name "$BUILDX_BUILDER_NAME" --driver docker-container >/dev/null
fi

BUILD_ARGS=(
    --builder "$BUILDX_BUILDER_NAME"
    --build-arg "BUILD_TIME=$NOW"
    --build-arg "BUILD_VERSION=$TAG"
    --build-arg "BUILD_COMMIT_REF=$SOURCE_SHA"
    --tag "$TARGET:$TAG"
)

if [ "$NO_PUSH" = "1" ]; then
    SERVER_ARCH="$(docker version --format '{{.Server.Arch}}' 2>/dev/null || uname -m)"
    case "$SERVER_ARCH" in
        arm64 | aarch64) LOCAL_PLATFORM="linux/arm64" ;;
        *) LOCAL_PLATFORM="linux/amd64" ;;
    esac
    log "LOG_INFO" "Building single-arch local image ($LOCAL_PLATFORM) without push"
    docker buildx build --platform "$LOCAL_PLATFORM" --load "${BUILD_ARGS[@]}" "$OUT_DIR"
else
    log "LOG_INFO" "Building and pushing $TARGET:$TAG for $PLATFORMS"
    docker buildx build \
        --platform "$PLATFORMS" \
        --tag "$TARGET:sha-$SOURCE_SHA" \
        --push \
        "${BUILD_ARGS[@]}" \
        "$OUT_DIR"

    DIGEST="$(docker buildx imagetools inspect "$TARGET:$TAG" -f '{{.Manifest.Digest}}' 2>/dev/null | head -n 1)"
    show_success "Pushed $TARGET:$TAG (sha-$SOURCE_SHA) ${DIGEST:+digest: $DIGEST}"
fi
