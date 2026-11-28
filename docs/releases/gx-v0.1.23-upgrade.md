# Upgrade to Shaiya GX gx-v0.1.23

1. Close the client and run `shutdown server` in the GX Server console. Make a
   [consistent SQLite backup](../server.md#back-up-and-restore) of your runtime
   database and save your local configuration.
2. Extract the new client and server packages into a fresh writable directory.
   Reuse compatible loose client data and preserve the supplied fonts and
   licenses.
3. Before starting the new server, restore the backup to
   `Server/runtime/rapture.sqlite` and copy the previous `.local/runtime.env` to
   `Server/.local/runtime.env` if you used one. If `RAPTURE_DB_PATH` overrides
   the default, point it at the restored database. Keep the supplied
   `Server/data/rapture.sqlite` as the development seed.
4. From the source checkout, refresh the interface in your loose client data.
   Replace the example path with your `Client/data` path. On Windows, use
   `py -3` in place of `python3`.

   ```sh
   python3 tools/patch_sysmsg.py "/path/to/Client/data"
   python3 tools/patch_option_graphic.py "/path/to/Client/data"
   python3 tools/patch_option_sound.py "/path/to/Client/data"
   ```

   The option-art tools require Pillow. They keep a pristine backup for each
   client data folder in `.local/option-art/<data-dir-name>` beside it. If you
   already patched the art with an older tool, pass that client's original PNG
   as the optional second argument. Skip the corresponding patch if the data
   already includes the updated rows or artwork.
5. Update `Server/data/game/Obelisk.ini` from the server package.
   `[RandomRespawn]` now takes an optional leading percentage followed by up to
   16 coordinates. The old minimum/maximum time pair is ignored.
6. Remove any language setting from your launcher. The client now reads the
   base data text.
7. Start GX Server and open the updated client. The new default ports are
   `31000` for Login and `31010` for World. If you kept the old local ports,
   pass `-login 127.0.0.1:30800 -world-port 30810` to the client. For custom
   ports, use the Login address and World port from your server configuration.
   Verify login, character selection and expected progress before removing the
   previous installation.

See the [server guide](../server.md) and the
[portable guide](../client-portable.md) for the data layout.
