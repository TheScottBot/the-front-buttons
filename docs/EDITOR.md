# The editor

The editor is at <http://127.0.0.1:48080/>, listening on this machine only. If
another program already holds port 48080, the daemon uses any free port instead,
logs a warning, and prints the address it chose; `-editor-port` picks another
number, and `-editor-port 0` always lets the system choose. Every key, page and
form in it works from the keyboard.

Every page needs a session, given by a one time login link, so other accounts on
the machine cannot use it. Open the editor with an `open_editor` key on the deck,
or with:

```bash
the-front-buttons -open-editor
```

which asks the running daemon for a link and opens it in your browser. The
session lasts until the browser closes, and survives the daemon restarting. On
macOS the install script puts the binary in
`~/Library/Application Support/the-front-buttons/bin`, so from a terminal:

```bash
"$HOME/Library/Application Support/the-front-buttons/bin/the-front-buttons" -open-editor
```
