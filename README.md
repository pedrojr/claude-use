# claude-use

[Leia em português (Brasil)](README_PTBR.md)

Compact, semi-transparent Windows overlay that shows your Claude usage limits
(the same data as **Settings → Usage** / `/usage` in Claude Code). Built in Go with [Fyne](https://fyne.io).

- Docked to the right edge of the primary monitor, vertically centred, always on top.
- Drag it with the mouse to move it anywhere; it reopens at that position from then on.
  **Reset position** in the tray menu docks it back to the right edge.
- Borderless, rounded corners, ~85% opaque, white text, no taskbar button (tray icon only) and never steals focus.
- Single instance: starting the executable again while the overlay is open does nothing.
- Refreshes every 5 minutes, right after a limit resets, and 1 minute after a failed attempt with no
  connection (e.g. when the PC wakes from sleep); the discreet ⟳ button in the footer forces a refresh.
- The "Resets in…" countdown is recomputed locally every 30 s.
- Bars change colour: blue, orange (≥ 75%) and red (≥ 90%).
- Interface in English (default) or Brazilian Portuguese.

## System tray

Tray icon menu:

- **Refresh now**
- **Click-through (ignore clicks)** – clicks pass through the overlay (opacity does not change).
  Use the tray to turn it off and click the refresh button again.
- **Hide / Show**
- **Reset position** – forgets the position you dragged the overlay to.
- **Start with Windows** – turns automatic start on/off for the current user (stores the path of the
  running `.exe` in `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`). If you move the executable,
  turn it off and on again.
- **Language** – English or Português (Brasil). Applied immediately and remembered.
- **Quit**

## Settings

Settings are stored per user in `%APPDATA%\claude-use\config.json`, never next to the executable,
so they work when the app is installed in Program Files (read-only for regular users):

```json
{
  "language": "pt-BR",
  "position": { "x": 1500, "y": 300 }
}
```

- `language`: `en` (default) or `pt-BR`.
- `position`: top-left corner of the overlay in screen pixels. Absent = docked to the right edge.
  If the monitor layout changes, the overlay is kept inside the nearest screen.

Delete the file to go back to the defaults. The only other thing written is the **Start with Windows**
value in `HKCU\...\Run`. Fyne (the UI toolkit) also creates an empty `%APPDATA%\fyne\com.github.claude-use`
folder; nothing is stored there.

## Installation

Download the installer `claude-use-setup-<version>.exe` from the
[Releases](https://github.com/pedrojr/claude-use/releases/latest) page and run it. It:

- asks how to install:
  - **Install for all users** (default) – in `C:\Program Files\claude-use`, requires administrator rights (UAC);
  - **Install for me only** – in `%LOCALAPPDATA%\Programs\claude-use`, no administrator rights needed
    (use this on machines where you are not an administrator);
- lets you change the destination folder in both modes. Without administrator rights it checks that
  the chosen folder is writable before installing;
- creates a Start Menu shortcut (and, if ticked, a Desktop one);
- offers **Start with Windows** (the same as the tray menu item), registered for the user who ran the installer,
  even when the elevation prompt was answered with another account;
- asks you to close the overlay if it is running (handy when updating);
- when installing for all users, removes an existing per-user install (older versions installed only that way);
- on uninstall, also removes the automatic start if it points to that copy. Your settings in `%APPDATA%\claude-use`
  are kept.

Silent install: `claude-use-setup-<version>.exe /VERYSILENT /ALLUSERS` (or `/CURRENTUSER`); `/DIR="C:\path"` changes the folder.
A log can be written with `/LOG="%TEMP%\claude-use-setup.log"`.

The same release has the standalone `claude-use.exe`, which runs without installing.

## Requirements (to build)

- Go 1.26+ (see `go.mod`)
- GCC for CGO (e.g. [MinGW-w64 / WinLibs](https://winlibs.com)) – required by Fyne
- Claude Code signed in with a Claude account (Pro/Max/Team/Enterprise)

## Build

```bash
windres -O coff -o rsrc_windows_amd64.syso claude-use.rc   # executable icon (optional)
go build -ldflags "-H=windowsgui -s -w" -o claude-use.exe .
```

`windres` comes with MinGW. The icon is [assets/claude-use.ico](assets/claude-use.ico), generated from the
tray icon with `go run assets/mkicon.go`. `-H=windowsgui` avoids opening a console. To test without the UI:

```bash
go build -o claude-use-cli.exe . && ./claude-use-cli.exe -print
```

Command-line flags:

- `-print` – prints the usage to the terminal and exits.
- `-autostart=on|off` – turns **Start with Windows** on/off for the current user and exits (used by the installer).

### Installer

The installer is built with [Inno Setup](https://jrsoftware.org/isinfo.php) (script in
[installer/claude-use.iss](installer/claude-use.iss)). Locally:

```bash
windres -O coff -o rsrc_windows_amd64.syso claude-use.rc
go build -trimpath -ldflags "-H=windowsgui -s -w" -o dist/claude-use.exe .
iscc /DAppVersion=1.0.0 installer/claude-use.iss   # produces dist/claude-use-setup-1.0.0.exe
```

## Publishing a version

The workflow [.github/workflows/release.yml](.github/workflows/release.yml) builds the app and the installer
on GitHub Actions. When a `v*` tag is pushed, it creates the release with both files:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow can also be run manually (**Actions → Release → Run workflow**). In that case it only builds
and keeps the files as a workflow artifact, without creating a release.

## How the data is obtained

The app reads the OAuth token that Claude Code keeps in `%USERPROFILE%\.claude\.credentials.json`
(or `%CLAUDE_CONFIG_DIR%\.credentials.json`) and calls `GET https://api.anthropic.com/api/oauth/usage`,
the same endpoint used by the usage screen. The file is re-read on every refresh; the app never stores
or renews the token. If "token expired" appears, open Claude Code once so it renews the token.

This endpoint is not publicly documented and may change.
