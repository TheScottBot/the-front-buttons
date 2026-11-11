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
