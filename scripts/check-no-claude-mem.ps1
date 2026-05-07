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

# Threat model: AGPL contagion only happens through code copying. Markdown is
# documentation — referencing the forbidden strings BY NAME (CONTRIBUTING.md,
# openspec/changes/*) is not contagion. We scan only code extensions and skip
# docs/, openspec/, scripts/, and .git/.
$extensions = @('*.go', '*.json', '*.ts', '*.js')

$excludePathPattern = '\\(\.git|scripts|docs|openspec)\\'

$fail = $false

foreach ($pattern in $forbidden) {
    $found = Get-ChildItem -Recurse -Include $extensions |
        Where-Object { $_.FullName -notmatch $excludePathPattern } |
        Select-String -Pattern ([regex]::Escape($pattern)) -SimpleMatch

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
