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

# Refuse a console build: Windows gives it a console window at every logon,
# and closing that window stops the daemon. Built with -H=windowsgui it runs
# with no window, as the macOS login agent does; its log is in the state folder.
# The subsystem is the 16 bit field 0x5C bytes after the PE header's start
# (signature 4, file header 20, then offset 68 in the optional header).
$stream = [System.IO.File]::OpenRead($binary)
try {
    $reader = New-Object System.IO.BinaryReader($stream)
    $stream.Position = 0x3C
    $peOffset = $reader.ReadInt32()
    $stream.Position = $peOffset
    $signature = $reader.ReadUInt32()
    $stream.Position = $peOffset + 0x5C
    $subsystem = $reader.ReadUInt16()
}
finally {
    $stream.Dispose()
}

$peSignature = 0x00004550 # "PE\0\0"
$windowsGUI = 2
if ($signature -ne $peSignature) {
    Write-Error "$BinaryPath is not a Windows executable" -ErrorAction Continue
    exit 2
}
if ($subsystem -ne $windowsGUI) {
    Write-Error ("$BinaryPath is a console build, which would keep a console window open. " +
        'Build it with: go build -ldflags "-H=windowsgui" -o the-front-buttons.exe ./cmd/the-front-buttons') -ErrorAction Continue
    exit 2
}

$taskName = (& $binary -print-name | Out-String).Trim()
if ($LASTEXITCODE -ne 0) { exit 1 }

$definition = (& $binary -print-scheduled-task -user $user | Out-String)
if ($LASTEXITCODE -ne 0) { exit 1 }

# -ErrorAction Stop on each: these cmdlets do not always honour the script's
# $ErrorActionPreference, and a failed registration must not report success.
Register-ScheduledTask -TaskName $taskName -Xml $definition -Force -ErrorAction Stop | Out-Null
Start-ScheduledTask -TaskName $taskName -ErrorAction Stop

Write-Output "Registered $taskName. It is running now and starts at every logon."
