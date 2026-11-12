# Removes the logon task that install-windows.ps1 registered. The task's name
# comes from the binary, as at install.
#
# Usage: scripts\uninstall-windows.ps1 -BinaryPath path\to\the-front-buttons.exe
param(
    [Parameter(Mandatory = $true)]
    [string] $BinaryPath
)

$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $BinaryPath -PathType Leaf)) {
    Write-Error "$BinaryPath is not a file" -ErrorAction Continue
    exit 2
}

$taskName = (& (Resolve-Path -LiteralPath $BinaryPath).Path -print-name | Out-String).Trim()
if ($LASTEXITCODE -ne 0) { exit 1 }

Unregister-ScheduledTask -TaskName $taskName -Confirm:$false

Write-Output "Removed $taskName."
