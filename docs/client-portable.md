# Shaiya GX portable client

Download the complete [portable client for Linux and Windows](https://drive.google.com/file/d/15BFHAGpLi_JJZbZ4fi8jpDsFQLdHmOyq/view?usp=sharing) from Google Drive and extract the entire folder to a writable location. It includes the shared `data/` directory and both executables:

```text
Shaiya GX/
  Shaiya GX
  Shaiya GX.exe
  data/
  licenses/
```

On Linux, open `Shaiya GX`; on Windows, open `Shaiya GX.exe`. Keep `data/` beside the executables and preserve the licenses. Client game data is hosted separately from this source repository.

[GitHub releases](https://github.com/Spelunkern/Shaiya-GX/releases/latest) provide updated executable-only packages for each OS, with `SHA256SUMS.txt`. To update, close the client, extract the appropriate package and replace only that OS's executable while keeping your existing `data/`. Retain the accompanying notices and licenses. The GitHub client packages contain no game data or server database; GX Server is a separate download. A future release may require a matching data update, so check its release notes.

Both executables can share the same folder and `data/` directory. Keep the included licenses and notices when redistributing. Start GX Server separately, then open the executable for your OS. The client defaults to login address 127.0.0.1:30800; use `-login IP:PORT` and `-world-port PORT` for another server. Liberation Sans (Arial-compatible metrics) and Vorbis decoding are embedded. No launcher, Go installation or media tools are needed.

Windows requires a 64-bit OS and a GPU driver supporting OpenGL 4.5. Linux requires a 64-bit OS, OpenGL 4.5, ALSA and the X11/Wayland graphics libraries. The published Linux build targets the Ubuntu 24.04 runner; use glibc 2.39 or newer. An executable built locally on another Linux distribution may have different runtime requirements. If an archive tool loses execute permissions on Linux, run `chmod +x 'Shaiya GX'`.

The executable discovers adjacent data independently of the working directory. For development, `-data PATH` or `SHAIYA_DATA` overrides it. Compatible data uses PNG interface images and cursors, Ogg Vorbis audio, and an EP5 `Character/Skill.SData`. Legacy SAH/SAF archives do not load directly.

Source, build instructions and third-party font sources: https://github.com/Spelunkern/Shaiya-GX

For gx-v0.1.4, update GX Server as well as the client: both endpoints now use
UTF-8 text with fixed protocol byte budgets. The Drive bundle has not been
reuploaded by this release. See the [accumulated patch notes](releases/gx-v0.1.4.md).
