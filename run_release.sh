set -euo pipefail

BINARY_NAME="pushover"
REPO_NWO="chew-z/Pushover"

# Pre-flight checks
echo "Performing pre-flight checks..."
gh auth status
if ! git diff-index --quiet HEAD --; then
    echo "Error: Uncommitted changes detected. Aborting."
    exit 1
fi
echo "Checks passed."

# Versioning: derive next patch from latest tag, or allow override
latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
IFS='.' read -r major minor patch <<< "${latest_tag#v}"
suggested="v${major}.${minor}.$((patch + 1))"

new_tag="${1:-$suggested}"
echo "Latest tag: ${latest_tag}"
echo "New version: ${new_tag}"
read -rp "Proceed with ${new_tag}? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }

# Build
echo "Building ${BINARY_NAME}..."
mkdir -p bin
go build -ldflags "-X main.version=${new_tag}" -o "bin/${BINARY_NAME}" .
echo "Build complete."

# Create Git tag and push
echo "Creating and pushing Git tag..."
git tag -a "${new_tag}" -m "Release ${new_tag}"
git push origin "${new_tag}"
echo "Tag pushed."

# Create GitHub Release
echo "Creating GitHub release..."
if [ "$(gh repo view "${REPO_NWO}" --json hasDiscussionsEnabled --jq .hasDiscussionsEnabled)" = "true" ]; then
    gh release create "${new_tag}" "bin/${BINARY_NAME}" \
        --title="${new_tag}" --generate-notes --latest \
        --discussion-category="Releases" --verify-tag
else
    gh release create "${new_tag}" "bin/${BINARY_NAME}" \
        --title="${new_tag}" --generate-notes --latest --verify-tag
fi

echo "Release ${new_tag} created successfully!"
