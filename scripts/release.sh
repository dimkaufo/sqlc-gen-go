#!/usr/bin/env bash
# Tag, build and publish a WASM release of the plugin.
#
#   make release                         # prompts for the version, suggests the next one
#   make release VERSION=v1.6.1-rc.4     # skip the version prompt
#   make release DRY_RUN=1               # build + print notes; no tag, push or release
#
# Asset: bin/<ASSET_PREFIX>_<version without v>.wasm. Release body: the sqlc.yaml plugin
# snippet (url + sha256), as for every earlier release.
set -euo pipefail

GH_REPO="${GH_REPO:-dimkaufo/sqlc-gen-go}"
ASSET_PREFIX="${ASSET_PREFIX:-dimkaufo-sqlc-gen-go}"
BRANCH="${BRANCH:-main}"
VERSION="${VERSION:-}"
DRY_RUN="${DRY_RUN:-}"

cd "$(dirname "$0")/.."

die() { echo "release: $*" >&2; exit 1; }
ask() { # ask <prompt> <default> → answer on stdout; reads the terminal, not stdin
	local answer
	read -r -p "$1" answer </dev/tty || true
	echo "${answer:-$2}"
}

go version >/dev/null 2>&1 || die "go does not run here; install it (mise install) first"
gh --version >/dev/null 2>&1 || die "gh not installed: brew install gh"
gh auth status >/dev/null 2>&1 || die "gh not logged in: gh auth login"

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

[ -n "$VERSION" ] || VERSION="$(ask "Version [${suggest}]: " "$suggest")"
[[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$ ]] || die "version must look like v1.2.3 or v1.2.3-rc.4, got '$VERSION'"
git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null && die "tag $VERSION already exists"

asset="bin/${ASSET_PREFIX}_${VERSION#v}.wasm"
url="https://github.com/${GH_REPO}/releases/download/${VERSION}/${ASSET_PREFIX}_${VERSION#v}.wasm"

echo "Building $asset from $(git rev-parse --short HEAD) ..."
rm -f bin/sqlc-gen-go bin/sqlc-gen-go.wasm
make -s all
cp bin/sqlc-gen-go.wasm "$asset"
sha="$(shasum -a 256 "$asset" | awk '{print $1}')"

notes="$(mktemp)"
trap 'rm -f "$notes" "${tagmsg:-}"' EXIT
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
echo
echo "Release notes:"
cat "$notes"

if [ -n "$DRY_RUN" ]; then
	echo "DRY_RUN: stopping before tag, push and release."
	exit 0
fi

# Annotated tag; message opens in the editor prefilled with the commit subjects.
tagmsg="$(mktemp)"
{
	if [ -n "$last_tag" ] && [ "$(git rev-list --count "$last_tag..HEAD")" = 1 ]; then
		git log -1 --format=%B HEAD
	else
		git log -1 --format=%s HEAD
		echo
		git log --format='- %s' "${last_tag:+$last_tag..}HEAD"
	fi
} >"$tagmsg"

login="$(gh api user --jq .login 2>/dev/null || echo unknown)"
echo
[ "$(ask "Tag $VERSION, push $BRANCH + tag to origin and publish the release on $GH_REPO as '$login'? [y/N] " n)" = y ] || die "aborted; nothing tagged or pushed"

git tag -a "$VERSION" -e -F "$tagmsg"
git push origin "$BRANCH"
git push origin "$VERSION"
gh release create "$VERSION" "$asset" -R "$GH_REPO" --title "$VERSION" --notes-file "$notes" --latest

echo
echo "Released $VERSION: https://github.com/${GH_REPO}/releases/tag/${VERSION}"
echo "  url:    $url"
echo "  sha256: $sha"
echo "platform-backend: make platform-backend.platform.sqlc.gen.go.use-release SQLC_GEN_GO_VERSION=$VERSION"
