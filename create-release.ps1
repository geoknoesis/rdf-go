# Script to create GitHub release via API
param(
    [string]$Token = $env:GITHUB_TOKEN
)

if (-not $Token) {
    Write-Host "GitHub token not found. Please provide a token:" -ForegroundColor Yellow
    Write-Host "1. Set GITHUB_TOKEN environment variable, or" -ForegroundColor Yellow
    Write-Host "2. Pass it as parameter: .\create-release.ps1 -Token 'your-token'" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "To create a token: https://github.com/settings/tokens" -ForegroundColor Cyan
    Write-Host "Required scope: 'repo' (for private repos) or 'public_repo' (for public repos)" -ForegroundColor Cyan
    exit 1
}

$releaseNotes = Get-Content -Path "RELEASE_NOTES_v0.1.0.md" -Raw -Encoding UTF8

$body = @{
    tag_name = "v0.1.0"
    name = "v0.1.0"
    body = $releaseNotes
    draft = $false
    prerelease = $false
} | ConvertTo-Json -Depth 10 -Compress

$headers = @{
    Authorization = "token $Token"
    Accept = "application/vnd.github.v3+json"
}

try {
    Write-Host "Creating release v0.1.0..." -ForegroundColor Green
    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/geoknoesis/rdf-go/releases" `
        -Method Post `
        -Headers $headers `
        -Body $body `
        -ContentType "application/json"
    
    Write-Host "Release created successfully!" -ForegroundColor Green
    Write-Host "Release URL: $($response.html_url)" -ForegroundColor Cyan
} catch {
    Write-Host "Error creating release:" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    if ($_.ErrorDetails.Message) {
        Write-Host $_.ErrorDetails.Message -ForegroundColor Red
    }
    exit 1
}

