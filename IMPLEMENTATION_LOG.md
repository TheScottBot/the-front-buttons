# Implementation log

Phase status for The Front Buttons. Entries are history and are not rewritten;
a later change is a dated note.

## Phase status

| Phase | Status | Outstanding author gate |
|---|---|---|
| Plan | Exited under DV1, 11 November 2026; TD1 to TD13 answered | Experiments 1, 2 and 3 deferred until the deck arrives (DV1); experiment 5 needs the author's token |
| TE1 | Automated criteria green on this Mac; every check proven on a planted violation | Reproduction on both machines; CI green on both runners |
| TE2 | Automated criteria green on this Mac, fuzz run recorded | None; CI on both runners after the author pushes |
| TE3 | Automated criteria green on this Mac | None; key order awaits experiment 1 (DV1) |
| TE4 | Automated criteria green on this Mac | Legibility on the physical deck at arm's length, both machines |
| TE5 | Automated criteria green on this Mac, fuzz runs recorded | None; every value awaits experiment 1 (DV1) |
| TE6 | Automated criteria green on this Mac; device contract skipped without a deck | Device contract, keys, paging and ten dock switches on both machines |
| TE7 | Automated criteria green on this Mac; binary smoke run | Every preset and both bricks on each machine |
| TE8 | Automated criteria green on this Mac | A real workflow red then green, press opening the failing run; credential contract with consent |
| TE9 | Automated criteria green on this Mac; live editor probed | A full page built in the editor on each machine, once keyboard only |
| TE10 | Automated criteria green on this Mac; the Windows script's refusal proven only in CI | Start at login on both machines with the deck live and no terminal |

## Entries

### 11 November 2026

The author decided to build ahead of the hardware, against documented interfaces
and open source implementations that describe the Stream Deck XL, so that the
work remaining when the deck arrives (expected 12 November 2026) is adjustment
and polish. Recorded as DV1 in `IMPLEMENTATION_DEVIATIONS.md`.

Plan stage readings recorded in `docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`.
TD1 to TD13 answered by the author. The settled description line was corrected to
read: This is for the lions living in the wiry frames of my peripherals' bodies.
Plan exited under DV1 with experiments 1 to 5 outstanding.

### 12 November 2026

The deck arrived. Experiment 1 and the device contract ran on the MacBook; two
defects were found and fixed (an abort when two watchers initialised hidapi at
once, and paging taking seconds), recorded in the evaluation under deck testing.

### Commits made by the agent under DV2

Each phase's commit is listed here by its subject; its hash is added by the next
commit, since a commit cannot name its own hash.

| Phase | Commit subject | Hash |
|---|---|---|
| TE4 | feat(render): add key renderer, status glyphs and bundled icons | `f11a68d` |
| TE5 | feat(deck): add Stream Deck XL model table and protocol codec | `fa6a3f4` |
| TE6 | feat(daemon): add HID transport, deck watching and the main loop | `d36a8ac` |
| TE7 | feat(bricks): add URL and application bricks, presets and the composition root | `ec1820f` |
| TE8 | feat(live): add live hub, GitHub Actions pipeline status and credentials | `5643a74` |
| TE9 | feat(editor): add the local editor and the open_editor brick | `6c543d1` |
| TE10 | docs(release): add start at login, generated brick reference and project documents | `90e8ce8` |

DV2 closed on 12 November 2026 when the author took back control of commits;
no commit after `90e8ce8` is the agent's.
