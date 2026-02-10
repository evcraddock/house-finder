---
name: release
description: Create a new release tag. Use when user says "release", "new version", "cut a release", "deploy", "tag a release", or similar.
---

# Release

Create a new release tag (`v*`) that triggers GitHub Actions to build CLI binaries and multi-arch Docker images.

## Quick Reference

```bash
# The script handles: version bump, tag, push, verify
# Path: .pi/skills/release/scripts/release.sh
.pi/skills/release/scripts/release.sh patch   # Bug fixes
.pi/skills/release/scripts/release.sh minor   # New features
.pi/skills/release/scripts/release.sh major   # Breaking changes
.pi/skills/release/scripts/release.sh 1.2.3   # Explicit version
```

## Instructions

### 1. Verify on main branch

```bash
git branch --show-current
git fetch origin main
git log origin/main..HEAD --oneline
```

- Must be on `main`
- No unpushed commits (if any exist, stop and tell user to push or create PR)

### 2. Get changes since last release

```bash
LATEST_TAG=$(git tag --list 'v*' --sort=-v:refname | head -1)
git log ${LATEST_TAG}..HEAD --oneline
```

If no commits since last tag, inform user and stop.

### 3. Analyze commits for version bump

**MAJOR** (breaking): `BREAKING CHANGE:` in message, or `feat!:`, `fix!:`
**MINOR** (feature): `feat:` prefix
**PATCH** (fix): `fix:`, `perf:` prefix

Other types (docs, style, refactor, test, chore, ci, build) don't bump alone.

Priority: MAJOR > MINOR > PATCH. Default to PATCH if unclear.

### 4. Present options to user

Show:

- Current version
- Commits since last release (grouped by type)
- Recommended bump and why

Ask user to choose:

- Recommended version (e.g., "0.2.0 - Minor (Recommended)")
- Alternative bumps if applicable
- Cancel

### 5. Run the release script

Once user chooses, run the script:

```bash
.pi/skills/release/scripts/release.sh <patch|minor|major>
```

The script handles:

1. Validates on main with no unpushed commits
2. Creates annotated tag
3. Pushes tag
4. **Verifies tag exists on remote**

### 6. Confirm success

After the script completes, note that GitHub Actions will:

- Build CLI binaries (linux amd64/arm64)
- Create GitHub release with binaries
- Build multi-arch Docker images (amd64 + arm64)
- Push to ghcr.io/evcraddock/house-finder with semver + latest tags

## Important Notes

- NEVER force push or use `--force` flags
- The script creates annotated tags automatically
- The script **verifies** the tag was pushed (checks remote)
- If script fails, read the error and do not retry blindly
- Version is injected via ldflags at build time — no file to update
