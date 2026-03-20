# Trigger a new release via GitHub Actions
# Usage: .\scripts\release.ps1 v1.0.0

param(
    [Parameter(Mandatory=$true)]
    [string]$Version
)

# Validate version format
if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$') {
    Write-Error "Version must match format v1.0.0 or v1.0.0-beta.1"
    exit 1
}

Write-Host "Triggering release for version: $Version" -ForegroundColor Green

# Using GitHub CLI
gh workflow run release.yml -f version="$Version"

Write-Host ""
Write-Host "Release workflow triggered!" -ForegroundColor Green
Write-Host "View progress: gh run list --workflow=release.yml"

$repo = gh repo view --json nameWithOwner -q ".nameWithOwner"
Write-Host "Or visit: https://github.com/$repo/actions"
