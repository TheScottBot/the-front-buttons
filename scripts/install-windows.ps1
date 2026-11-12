# Registers The Front Buttons to start at the current user's logon on Windows
# (TE10), with no administrator rights: the task runs in the person's own
# interactive session at least privilege, which Microsoft documents a
# non administrator may register for their own account. The task definition and
# its name come from the binary itself, so they are written once, in Go.
#
# Usage: scripts\install-windows.ps1 -BinaryPath path\to\the-front-buttons.exe
param(
    [Parameter(Mandatory = $true)]
    [string] $BinaryPath
)

$ErrorActionPreference = 'Stop'

# Refuse anything but an existing file before touching the system.
if (-not (Test-Path -LiteralPath $BinaryPath -PathType Leaf)) {
    Write-Error "$BinaryPath is not a file" -ErrorAction Continue
    exit 2
}

$binary = (Resolve-Path -LiteralPath $BinaryPath).Path
$user = "$env:USERDOMAIN\$env:USERNAME"

$taskName = (& $binary -print-name | Out-String).Trim()
if ($LASTEXITCODE -ne 0) { exit 1 }

$definition = (& $binary -print-scheduled-task -user $user | Out-String)
if ($LASTEXITCODE -ne 0) { exit 1 }

Register-ScheduledTask -TaskName $taskName -Xml $definition -Force | Out-Null
Start-ScheduledTask -TaskName $taskName

Write-Output "Registered $taskName. It is running now and starts at every logon."
