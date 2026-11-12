# Security

What The Front Buttons trusts, what it does not, and what it does not protect
against, stated plainly.

## What runs

One background process per signed in user: the daemon. It talks to the Stream
Deck over USB HID, polls GitHub over HTTPS for the pipeline keys configured, and
serves a web editor on the loopback interface (127.0.0.1) only, on a port the
operating system chooses at each start. Nothing listens on any other interface.
Nothing runs with administrator rights: on macOS it is a per user login agent, on
Windows a per user logon task at least privilege.

## Trust boundaries

| Boundary | Treated as | Defences |
|---|---|---|
| USB input reports from the deck | hostile bytes | exact length checked before decoding; the decoder cannot read past its buffer; fuzzed in CI |
| The configuration file | untrusted, though it is the person's own | bounded at 1 MiB before parsing; one YAML document, no anchors or aliases, no duplicate keys; every field checked by type, bounds and format; unknown fields refused; fuzzed in CI |
| Editor requests | untrusted, including requests forged by any page open in the same browser | the editor's own Host only, which defeats DNS rebinding; any Origin must be the editor's own, and every change must carry one; a 256 bit token minted per launch on every change; bodies bounded at 64 KiB; a strict content security policy; no framing |
| GitHub API responses | untrusted | bounded at 1 MiB; only three fields read; a run's address is opened only if it is https |
| URLs and application names from configuration | untrusted | refused if empty, over 4,096 bytes, led by a hyphen or holding a control character; passed to the operating system as one argument, never through a shell or cmd.exe |

## Secrets

Tokens live only in the macOS Keychain or Windows Credential Manager, filed under
the service name `the-front-buttons`. Configuration names a credential and never
holds one. The editor's credential page is write only. The token is read from the
store for each GitHub request and is never put in an error, a log line or a
response; a test runs the whole composition against a refusing GitHub and fails
if the token reaches the log, and that test is proven against a planted leak.

## Accepted risks

These are recorded for the author to accept, amend or refuse in their own words;
until then they are the agent's statement of the risks, not the author's
acceptance.

- Any program running as the same user can read the configuration file and the
  log, can reach the editor's loopback port, and can read the editor's pages
  (which hold the token) by fetching them. The editor defends against web pages,
  not against software already running as the person.
- The configuration can hold colleagues' email addresses inside Teams links
  (PRIVACY.md).
- A key opens what it is configured to open. Someone who can edit the
  configuration can make a key open a harmful https URL. The only schemes
  allowed are https and Discord's own `discord://`, on discord.com only.

## Limitations

- The protocol has not yet been run against a physical deck (DV1).
- The editor has not been checked with a screen reader.
- Nothing here has been reviewed by anyone but its author and its agent.

## For IT

The daemon is a user level process listening on 127.0.0.1 only, on an ephemeral
port, serving a local configuration editor protected by a per launch token. It
makes outbound HTTPS requests to the GitHub API host configured (api.github.com
by default) and to nothing else. It needs no administrator rights and installs no
driver.

## Reporting

Report a problem to the repository's owner privately rather than in a public
issue.
