# Stops the daemon and removes the logon task that install-windows.ps1
# registered. The task's name comes from the binary, as at install.
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

$binary = (Resolve-Path -LiteralPath $BinaryPath).Path

$taskName = (& $binary -print-name | Out-String).Trim()
if ($LASTEXITCODE -ne 0) { exit 1 }

# Ask the daemon to stop, so it blanks the deck on the way out, as the macOS
# script's bootout does. Task Scheduler cannot: Stop-ScheduledTask kills the
# process, leaving its last images on the keys. Piping the output makes
# PowerShell wait for this windowless binary to exit, and so set its status.
$stopOutput = (& $binary -stop | Out-String).Trim()
$stopStatus = $LASTEXITCODE
if ($stopOutput) { Write-Output $stopOutput }

$task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue

if ($null -ne $task) {
    # The task shows Running until the daemon has exited; a clean stop takes
    # under a second, plus up to the editor's five second shutdown.
    $deadline = (Get-Date).AddSeconds(10)
    while ($stopStatus -eq 0 -and (Get-ScheduledTask -TaskName $taskName).State -eq 'Running' -and
        (Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 200
    }

    if ((Get-ScheduledTask -TaskName $taskName).State -eq 'Running') {
        Write-Warning "$taskName did not stop when asked; ending it. The deck may keep its last images until it is unplugged."
        Stop-ScheduledTask -TaskName $taskName -ErrorAction Stop
    }

    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction Stop
    Write-Output "Removed $taskName."
}
else {
    Write-Output "$taskName was not registered to start at logon."
}

if ($stopStatus -ne 0) { exit 1 }
