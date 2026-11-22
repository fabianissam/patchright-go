# Bump github.com/fabianissam/patchright-go on the hazebuli_revived dev worktree
# and open a PR against origin/dev. Does not touch a dirty checkout: it uses a
# sibling worktree from origin/dev.
param(
    [string]$HazebuliRoot = "C:\Users\Fabian\repos\hazebuli_revived",
    [string]$Worktree = "C:\Users\Fabian\orca\workspaces\hazebuli_revived\patchright-go"
)

$ErrorActionPreference = "Stop"

git -C $HazebuliRoot fetch origin
if (-not (Test-Path $Worktree)) {
    git -C $HazebuliRoot worktree add -B chore/patchright-go $Worktree origin/dev
} else {
    git -C $Worktree fetch origin
    git -C $Worktree checkout -B chore/patchright-go origin/dev
}

New-Item -ItemType Directory -Force -Path (Join-Path $Worktree "internal\patchrightpin") | Out-Null
$pin = @"
package patchrightpin

// Pins github.com/fabianissam/patchright-go so go mod tidy keeps the daily bump.
import _ "github.com/fabianissam/patchright-go"
"@
[System.IO.File]::WriteAllText((Join-Path $Worktree "internal\patchrightpin\pin.go"), $pin.Replace("`r`n", "`n") + "`n")

Push-Location $Worktree
try {
    $env:GOPROXY = "direct"
    go get github.com/fabianissam/patchright-go@main
    go mod tidy
    git add go.mod go.sum internal/patchrightpin
    git diff --cached --quiet
    if ($LASTEXITCODE -eq 0) {
        Write-Output "hazebuli already on latest patchright-go"
        return
    }
    git commit -m "Bump patchright-go on dev."
    git push -u origin chore/patchright-go --force
    gh pr view --repo fabianissam/hazebuli_revived --base dev --head fabianissam:chore/patchright-go 2>$null
    if ($LASTEXITCODE -ne 0) {
        gh pr create --repo fabianissam/hazebuli_revived --base dev --head chore/patchright-go --title "Bump patchright-go" --body "Automated bump of github.com/fabianissam/patchright-go onto the dev branch."
    }
} finally {
    Pop-Location
}
