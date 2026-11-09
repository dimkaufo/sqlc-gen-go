#!/usr/bin/env bash
# Build a WASM release and print what to do to publish it. Pushes and publishes nothing.
#
#   make release                         # prompts for the version, suggests the next one
#   make release VERSION=v1.6.1-rc.4     # skip the version prompt
#
# Writes into bin/ (gitignored):
#   <ASSET_PREFIX>_<version without v>.wasm   the release asset
#   release-<version>-notes.md                release description: sqlc.yaml snippet (url + sha256)
#   release-<version>-tag.txt                 annotated tag message, prefilled from the commits
set -euo pipefail

GH_REPO="${GH_REPO:-dimkaufo/sqlc-gen-go}"
ASSET_PREFIX="${ASSET_PREFIX:-dimkaufo-sqlc-gen-go}"
BRANCH="${BRANCH:-main}"
VERSION="${VERSION:-}"

cd "$(dirname "$0")/.."

die() { echo "release: $*" >&2; exit 1; }

go version >/dev/null 2>&1 || die "go does not run here; install it (mise install) first"

[ "$(git rev-parse --abbrev-ref HEAD)" = "$BRANCH" ] || die "not on $BRANCH"
git diff --quiet && git diff --cached --quiet || die "uncommitted changes to tracked files; commit or drop them first (the asset must match the tag)"
git fetch -q origin "$BRANCH" --tags
git merge-base --is-ancestor "origin/$BRANCH" HEAD || die "$BRANCH is behind origin/$BRANCH; pull first"

last_tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
echo "Recent tags:"
git tag -l --sort=-v:refname | head -5 | sed 's/^/  /'

# Next version: bump -rc.N if the last tag is an rc, else bump the patch.
suggest=""
if [[ "$last_tag" =~ ^(v[0-9]+\.[0-9]+\.[0-9]+)-rc\.([0-9]+)$ ]]; then
	suggest="${BASH_REMATCH[1]}-rc.$((BASH_REMATCH[2] + 1))"
elif [[ "$last_tag" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
	suggest="v${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.$((BASH_REMATCH[3] + 1))"
fi

if [ -n "$last_tag" ]; then
	echo "Commits since $last_tag:"
	git log --format='  %h %s' "$last_tag..HEAD"
	[ -n "$(git log --format=%h "$last_tag..HEAD")" ] || die "nothing to release since $last_tag"
fi

if [ -z "$VERSION" ]; then
	read -r -p "Version [${suggest}]: " VERSION </dev/tty || true
	VERSION="${VERSION:-$suggest}"
fi
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]] || die "version must look like v1.2.3 or v1.2.3-rc.4, got '$VERSION'"
git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null && die "tag $VERSION already exists"

asset="bin/${ASSET_PREFIX}_${VERSION#v}.wasm"
notes="bin/release-${VERSION}-notes.md"
tagmsg="bin/release-${VERSION}-tag.txt"
url="https://github.com/${GH_REPO}/releases/download/${VERSION}/${ASSET_PREFIX}_${VERSION#v}.wasm"

echo "Building $asset from $(git rev-parse --short HEAD) ..."
rm -f bin/sqlc-gen-go bin/sqlc-gen-go.wasm
make -s all
cp bin/sqlc-gen-go.wasm "$asset"
sha="$(shasum -a 256 "$asset" | awk '{print $1}')"

cat >"$notes" <<EOF
\`\`\`yaml
version: '2'
plugins:
  - name: golang
    wasm:
      url: ${url}
      sha256: ${sha}
\`\`\`
EOF

# One commit: its full message. Several: newest subject, then every subject as a bullet.
{
	if [ -n "$last_tag" ] && [ "$(git rev-list --count "$last_tag..HEAD")" = 1 ]; then
		git log -1 --format=%B HEAD
	else
		git log -1 --format=%s HEAD
		echo
		git log --format='- %s' "${last_tag:+$last_tag..}HEAD"
	fi
} >"$tagmsg"

cat <<EOF

================ Release $VERSION ================

Actions:
  1. git push origin $BRANCH
  2. git tag -a $VERSION -e -F $tagmsg    # edit the prefilled message
  3. git push origin $VERSION
  4. Open https://github.com/${GH_REPO}/releases/new?tag=${VERSION}&title=${VERSION}
     - title:       $VERSION
     - description: below (also in $notes)
     - attach file: $PWD/$asset
     - keep "Set as the latest release" checked; not a pre-release
  5. platform-backend: make platform-backend.platform.sqlc.gen.go.use-release SQLC_GEN_GO_VERSION=$VERSION

Tag message ($tagmsg):
$(sed 's/^/  /' "$tagmsg")

Description ($notes):
$(cat "$notes")
EOF
