# Download W3C RDF test suites
# Usage: .\download-w3c-tests.ps1 [output-directory]

param(
    [string]$OutputDir = ".\w3c-tests"
)

Write-Host "Downloading W3C test suites to: $OutputDir" -ForegroundColor Cyan
Write-Host ""

# Create output directory
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# Function to download a GitHub repo
function Download-Repo {
    param(
        [string]$Repo,
        [string]$Name,
        [string]$Subdir = ""
    )
    
    Write-Host "Downloading $Name..." -ForegroundColor Yellow
    
    $repoPath = Join-Path $OutputDir $Name
    $zipPath = Join-Path $env:TEMP "$Name.zip"
    
    # Try git clone first (if git is available)
    if (Get-Command git -ErrorAction SilentlyContinue) {
        if (Test-Path $repoPath) {
            Write-Host "  $Name already exists, updating..." -ForegroundColor Gray
            Push-Location $repoPath
            git pull 2>&1 | Out-Null
            Pop-Location
        } else {
            Push-Location $OutputDir
            git clone "https://github.com/w3c/$Repo.git" $Name 2>&1 | Out-Null
            Pop-Location
        }
    } else {
        # Fallback to zip download
        Write-Host "  Downloading zip archive..." -ForegroundColor Gray
        $url = "https://github.com/w3c/$Repo/archive/refs/heads/main.zip"
        Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing
        
        Write-Host "  Extracting..." -ForegroundColor Gray
        Expand-Archive -Path $zipPath -DestinationPath $OutputDir -Force
        Remove-Item $zipPath
        
        $extractedPath = Join-Path $OutputDir "${Repo}-main"
        if (Test-Path $extractedPath) {
            Move-Item -Path $extractedPath -Destination $repoPath -Force
        }
    }
    
    Write-Host "✓ Downloaded $Name" -ForegroundColor Green
    Write-Host ""
}

# Download test suites
Download-Repo -Repo "rdf-tests" -Name "rdf-tests"
Download-Repo -Repo "rdf-star" -Name "rdf-star-tests"
Download-Repo -Repo "json-ld-api" -Name "json-ld-tests"

# Organize test files
Write-Host "Organizing test files..." -ForegroundColor Cyan

$formatDirs = @{
    "turtle" = @("rdf-tests\turtle", "rdf-star-tests\turtle")
    "ntriples" = @("rdf-tests\ntriples", "rdf-star-tests\ntriples")
    "trig" = @("rdf-tests\trig", "rdf-star-tests\trig")
    "nquads" = @("rdf-tests\nquads", "rdf-star-tests\nquads")
    "rdfxml" = @("rdf-tests\rdf-xml", "rdf-tests\rdfxml")
    "jsonld" = @("json-ld-tests\tests")
}

foreach ($format in $formatDirs.Keys) {
    $targetDir = Join-Path $OutputDir $format
    New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
    
    foreach ($sourceDir in $formatDirs[$format]) {
        $sourcePath = Join-Path $OutputDir $sourceDir
        if (Test-Path $sourcePath) {
            Copy-Item -Path "$sourcePath\*" -Destination $targetDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Write-Host ""
Write-Host "✓ All test suites downloaded and organized!" -ForegroundColor Green
Write-Host "Set `$env:W3C_TESTS_DIR=$OutputDir to run conformance tests." -ForegroundColor Cyan

