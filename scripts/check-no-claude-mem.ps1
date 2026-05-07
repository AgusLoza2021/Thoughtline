# check-no-claude-mem.ps1 — PowerShell equivalent for Windows CI.
# Ensure no AGPL strings from claude-mem were copied into this codebase.
# Exit 1 if any forbidden string is found; exit 0 otherwise.

$ErrorActionPreference = 'Stop'

$forbidden = @(
    'buildObservationPrompt',
    'buildSummaryPrompt',
    '<observed_from_primary_session>',
    'SSEBroadcaster',
    'viewer-bundle.js',
    '"mem-search"'
)

$extensions = @('*.go', '*.json', '*.md', '*.ts', '*.js', '*.sh', '*.ps1')

$fail = $false

foreach ($pattern in $forbidden) {
    $found = Get-ChildItem -Recurse -Include $extensions -Exclude '.git' |
        Select-String -Pattern ([regex]::Escape($pattern)) -SimpleMatch |
        Where-Object { $_.Path -notmatch '\\\.git\\' }

    if ($found) {
        Write-Error "ERROR: forbidden string found: $pattern"
        $found | ForEach-Object { Write-Error "  $($_.Path):$($_.LineNumber): $($_.Line.Trim())" }
        $fail = $true
    }
}

if ($fail) {
    Write-Error ""
    Write-Error "FAIL: One or more claude-mem AGPL strings detected."
    Write-Error "      Do NOT copy code, prompts, or schemas from claude-mem (AGPL-3.0)."
    exit 1
}

Write-Output "OK: no claude-mem strings found."
