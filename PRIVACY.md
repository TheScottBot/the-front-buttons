# Privacy

## What is collected

Nothing leaves the machine except two kinds of request a person configures. A
pipeline key's GitHub request names a repository, workflow and branch and
carries that person's token, sent only to the host the token was stored for. A
Spotify album key fetches the current track's art from Spotify's image server.
There is no telemetry, analytics or crash reporting.

## What is stored, and where

- The configuration file, in the per machine folder `docs/SETUP.md` names. It
  can hold colleagues' identifiers: a Teams chat link names its participants by
  their user principal names, usually email addresses.
- Credentials, in the operating system's credential store only.
- The log, in the state folder, rotated at a megabyte with three older files kept.
  It records kinds, pages, positions, reason categories and counts. It never
  records a URL (which can hold colleagues' addresses), a token or a request
  body.

## Retention

Everything stays until the person deletes it. Removing the configuration folder
removes the configuration and logs; credentials are removed in Keychain Access or
Credential Manager under the name `the-front-buttons`.

## What this does not protect against

- Anyone, or any program, with access to the person's account can read the
  configuration file, including colleagues' addresses in Teams links.
- Opening a key's URL hands it to the browser or app that handles it, which has
  its own privacy terms.
