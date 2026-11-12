# Provenance

Where every vendored file came from (engineering datum Part 2 and 0.13). Each is
kept verbatim and never edited; a fix goes upstream or into a wrapper.

## Tabler icon font

Vendored on 11 November 2026 for TD5, into `internal/render/tablericons/`.

| Item | Value |
|---|---|
| Upstream | `@tabler/icons-webfont` on npm, from `https://github.com/tabler/tabler-icons` |
| Version | 3.49.0, published 5 October 2026 (chosen over 3.51.0, published 8 November, so the release had more than two weeks of exposure) |
| Fetched from | `https://unpkg.com/@tabler/icons-webfont@3.49.0/` |
| Licence | MIT, Copyright (c) 2020-2026 Paweł Kuna; the licence file is vendored beside the font |

| File | Upstream path | Bytes | SHA-256 |
|---|---|---|---|
| `tabler-icons.ttf` | `dist/fonts/tabler-icons.ttf` | 1,827,912 | `d291676dc4c28b386552753bf16cf2bef0e93e5543f8e0b5c5d72c0da41ae1bc` |
| `tabler-icons.css` | `dist/tabler-icons.css` | 260,551 | `a0562b340ab94e3aca60a144f6b95a056f792744f2bde38a5e5352a2aa0918d0` |
| `LICENSE` | `LICENSE` | 1,073 | `b740a1d46122672da62833e97f7e7c8a13fa85cbc7445b584b297cc00dde93db` |

None of the three contains CR characters, so the stored bytes and the upstream
bytes are the same under `.gitattributes`' LF rule; the digests cover both.

To verify:

```bash
shasum -a 256 internal/render/tablericons/tabler-icons.ttf internal/render/tablericons/tabler-icons.css internal/render/tablericons/LICENSE
```

## htmx

Vendored on 12 November 2026 for TD6, into `internal/editor/static/`.

| Item | Value |
|---|---|
| Upstream | `htmx.org` on npm, from `https://github.com/bigskysoftware/htmx` |
| Version | 2.0.11, published to npm 22 September 2026 (the 2.x line TD6 chose; the Plan stage's 1 November date was from GitHub's release list, not npm) |
| Fetched from | `https://unpkg.com/htmx.org@2.0.11/` |
| Licence | Zero Clause BSD; vendored beside the script as `htmx-LICENSE` |

| File | Upstream path | Bytes | SHA-256 |
|---|---|---|---|
| `htmx.min.js` | `dist/htmx.min.js` | 52,182 | `d6fdc75f204e6bdefa99b69bf1e6d4ac69b8a364f77929f45c13476b4000f717` |
| `htmx-LICENSE` | `LICENSE` | 642 | `d3d2456f76414f2456104660ebd65aff1c04cd7966b942bdabd63f3cdb316a38` |

Neither file contains CR characters, so the stored and upstream bytes match.

```bash
shasum -a 256 internal/editor/static/htmx.min.js internal/editor/static/htmx-LICENSE
```
