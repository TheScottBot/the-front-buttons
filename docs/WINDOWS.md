# Installing on Windows without a C compiler

This guide installs The Front Buttons on Windows 10 or 11 built without cgo,
so it needs no C compiler: the deck is driven through Windows' own `hid.dll`
and `setupapi.dll` by the purego driver, not through hidapi. Nothing here
needs administrator rights. [SETUP.md](SETUP.md) covers the default build,
which needs gcc on Windows; this guide replaces its Windows steps.

Every command is for PowerShell, run from the folder you cloned into unless it
says otherwise.

## What you need

- Windows 10 or 11 on 64 bit Intel or AMD.
- Go 1.27.4 or later, from [go.dev/dl](https://go.dev/dl/). Check with
  `go version`.
- Git, to fetch the code. Check with `git version`.
- The Stream Deck XL, plugged in.
- Elgato's own Stream Deck software quit, including from the notification
  area. A deck it holds is reported as busy in the log.

No C compiler, MSYS2 or Node is needed: the editor's script is built and
checked in already.

## 1. Get the code

```powershell
git clone https://github.com/TheScottBot/the-front-buttons.git
cd the-front-buttons
```

To update later, stop the daemon (step 6), then `git pull` and build again.

## 2. Build without cgo

```powershell
$env:CGO_ENABLED = "0"
go build -ldflags "-H=windowsgui" -o the-front-buttons.exe ./cmd/the-front-buttons
```

- `CGO_ENABLED` set to 0 leaves out every file that needs a C compiler. The
  build then holds only the purego HID driver, so it is the one used whatever
  the setting says.
- `-H=windowsgui` builds it without a console window. The install script
  refuses a console build, which would keep a window open at every logon and
  stop the daemon when that window closed.
- `$env:CGO_ENABLED` lasts until the PowerShell window closes. Set it again in
  a new window before building.

The scheduled task runs the binary from where you built it, so build it in a
folder you will keep.

## 3. Say purego in the configuration

The configuration file is `%AppData%\the-front-buttons\config.yaml`. If you
have one from before, add this line under `schema_version`:

```yaml
schema_version: 1
platform_bridge: purego
pages:
  ...
```

If you have no file yet, start the daemon first (step 4): it creates one when
the editor saves its first page, and you can add the line then.

A build without cgo uses purego even without the line, but logs a warning at
every start that cgo was asked for and is not in the build. The line keeps the
log quiet and says what you meant. [CONFIGURATION.md](CONFIGURATION.md)
describes the setting in full.

## 4. Start it at every logon

```powershell
scripts\install-windows.ps1 -BinaryPath .\the-front-buttons.exe
```

This registers a task named "The Front Buttons" that runs at your logon, in
your session, at least privilege, and starts it now.

If PowerShell refuses to run scripts on this machine, run it this way instead,
which allows this one script and changes no setting:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\install-windows.ps1 -BinaryPath .\the-front-buttons.exe
```

## 5. Check it is running on purego

The log is in `%AppData%\the-front-buttons\state\`. Its first lines after a
start say which bridge it uses:

```powershell
Select-String -Path "$env:AppData\the-front-buttons\state\the-front-buttons.log" -Pattern '"platform bridge"','"deck"' | Select-Object -Last 3
```

You want a `platform bridge` line with `"asked":"purego"` and
`"hid":"purego"`, then a `deck` line with `"state":"connected"`. A deck reported
as busy is held by another program, most often Elgato's software.

Open the editor with a one time login link:

```powershell
.\the-front-buttons.exe -open-editor | Out-Null
```

## 6. Stop, rebuild and start again

Windows cannot signal a program with no console, and ending the task kills it
with its last images left on the deck. Ask it to stop instead, which blanks the
deck:

```powershell
.\the-front-buttons.exe -stop | Out-Null
```

The `| Out-Null` makes PowerShell wait for the windowless binary. Then rebuild
(step 2) and start it again:

```powershell
Start-ScheduledTask -TaskName "The Front Buttons"
```

## 7. Uninstall

```powershell
scripts\uninstall-windows.ps1 -BinaryPath .\the-front-buttons.exe
```

This asks the daemon to stop, waits up to ten seconds, and removes the task. It
leaves the configuration, the log and any secrets in Credential Manager.

## What works on Windows today

| Key | On Windows |
| --- | --- |
| Pages, page markers, the dashboard, world clocks | Work |
| Opening a link or launching an application | Work, through the shell; an application key switches to the application instead when it is running |
| `press_key` and `press_shortcut` | Work, through SendInput |
| Quitting on a triple press | Closes the application's own windows |
| Pasting a Mural link, the Mural keys | Written; the clipboard, written and read, is untried on Windows |
| GitHub Actions, Mural sign in | Work; secrets live in Credential Manager |
| Application icons on keys | Work, through the shell. An application the shell cannot find shows the crossed box that marks an icon not available |
| Bringing an application forward | Works. An application key, and an app screen's key before its key presses, restores the application's window and puts it in front, holding Alt for that moment so Windows lets a background program take the focus. An application whose only window is hidden, as one closed to the notification area, counts as not running and is launched, which brings most such applications back. A Store app whose window belongs to ApplicationFrameHost is launched too, and the shell brings back the one running. |
| Window cycling, Dock style badges, Spotify | Not available on Windows; the editor marks them so. For window cycling, a `press_shortcut` key with Alt+Tab as its Windows half switches to the window used last |

The deck itself has passed its checks through the purego driver on macOS, and
runs on a Windows machine through the same driver. The checks below test the
Windows driver on its own.

## Checking the deck driver on Windows

With the daemon stopped (step 6) and the deck plugged in and held by nothing
else:

1. Without the deck, the driver's own logic and its Windows calls:

   ```powershell
   go test ./internal/deck/hidtransport/ -run WinHID -v
   ```

2. On the deck, which draws over it and reads its serial number:

   ```powershell
   $env:THE_FRONT_BUTTONS_EXPERIMENT = "1"
   go test ./internal/deck/hidtransport/ -run 'WinHID_OnTheDeck|TestDriverContract_HoldsForEachBuiltInDriver_OnAnAttachedDeck' -v
   ```

3. The drawing speed and key presses, which need you at the deck to press
   keys while experiment 1 runs:

   ```powershell
   $env:THE_FRONT_BUTTONS_EXPERIMENT_DRIVER = "purego"
   go test ./internal/deck/hidtransport/ -run TestExperiment -v
   Remove-Item Env:THE_FRONT_BUTTONS_EXPERIMENT, Env:THE_FRONT_BUTTONS_EXPERIMENT_DRIVER
   ```

4. By hand: with the daemon running, unplug the deck, wait five seconds and
   plug it back in. The log should show the deck disconnected, then connected,
   and the page should come back.

The macOS results to compare against are in the evaluation.

## Checking application icons on Windows

Application icons are drawn by the shell, as File Explorer and the Start menu
draw them, and need neither the deck nor the daemon:

1. Notepad's icon, by its path and by the form the app picker gives it, a
   missing program reported as not found, and every Start menu entry the app
   picker lists. The last logs how many entries were drawn and names any the
   shell could not find:

   ```powershell
   go test ./internal/platform/systemadapter/ -run 'WindowsApplicationIcons' -v
   ```

2. By hand: with the daemon running, open the editor, add an
   `open_application` key, pick an application such as Notepad or Teams and
   tick "Use app icon". The key on the deck and in the editor should show
   the application's icon, on a background matching its edge. An application
   whose icon cannot be read shows the crossed box, and the log gives the
   reason under `application icon not loaded`.

## Troubleshooting

- **The install script says the binary is a console build.** Build again with
  `-ldflags "-H=windowsgui"` (step 2).
- **The log says the deck is busy.** Quit Elgato's Stream Deck software, then
  stop and start the daemon (step 6).
- **The log says cgo was asked for and is not in the build.** Add
  `platform_bridge: purego` to the configuration (step 3).
- **No log file appears.** Check the task is running with
  `Get-ScheduledTask -TaskName "The Front Buttons"`; its state should be
  Running.
