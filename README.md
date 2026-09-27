# Highway Pursuit Modern Display Patcher

A small, dependency-free display compatibility patcher for the free PC game **Highway Pursuit**.

**Original game / official game page:**  
https://adamdawes.com/games/highway-pursuit.html

This project patches the user's existing `HighwayPursuit.exe`. No original game executable or game assets are distributed by this repository.

## What the patch adds

The patched game keeps the original resolutions and adds modern display modes:

- 640x480
- 800x600
- 1024x768
- 1280x960
- 1920x1080
- 2160x1440
- 3840x2160
- 5120x1440

It also includes:

- Borderless fullscreen using the game's windowed Direct3D 8 path
- DPI-aware window positioning, including Windows display scaling such as 125%
- Ultrawide HUD positioning centered around a 16:9 safe area
- Alt-Tab friendly behavior
- Automatic exact backup of the executable before patching
- Safe compatibility checks at every location that will actually be modified

## Installation

1. Install or extract Highway Pursuit from the original game page above.
2. Download the latest patcher release ZIP from this repository.
3. Extract `HighwayPursuit-Ultrawide-Patcher.exe` into the Highway Pursuit game folder, next to `HighwayPursuit.exe`.
4. Make sure the game is closed.
5. Double-click `HighwayPursuit-Ultrawide-Patcher.exe`.
6. Confirm the patch when prompted.
7. Start the game and select the desired resolution in **Options -> Graphics options**.

You can also drag `HighwayPursuit.exe` onto the patcher.

The patcher creates an exact backup named `HighwayPursuit.original.exe` before modifying the game. If that filename is already used by a different executable, a numbered backup name is chosen instead.

## Restoring the original

The executable that existed immediately before patching is kept as a backup.

To restore it, run:

```text
HighwayPursuit-Ultrawide-Patcher.exe --restore
```

The patcher finds the backup that exactly corresponds to the currently patched executable and restores it byte-for-byte.

## No runtime installation required

The release patcher is a standalone Windows executable built with Go and the standard library only. End users do **not** need Python, .NET, Visual C++ redistributables, Go, or an installer.

The published executable is unsigned unless the release maintainer code-signs it. Windows SmartScreen may therefore show a warning for a new or low-reputation build. The complete source is provided so the patcher can be audited or rebuilt.

## Building from source

Developers need Go 1.21 or newer.

On Windows:

```bat
build.bat
```

On Linux/macOS with Go installed:

```sh
./build.sh
```

The build scripts produce a 32-bit Windows GUI executable. The 32-bit binary also runs normally on modern 64-bit Windows systems.

## License and disclaimer

The patcher source is released under the MIT License. Highway Pursuit itself is not included and is not covered by this license.

This is an unofficial fan-made compatibility/display patch and is not affiliated with the original game author or publishers.