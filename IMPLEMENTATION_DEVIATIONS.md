# Implementation deviations

Every departure from a contract, from the project specification, or from the
engineering datum (version 2, 20 September 2026). Each entry gives the rule,
what is done instead, the reason, the author's decision and the covering test.

## DV1: building ahead of the hardware experiments

Recorded 11 November 2026.

**Rule departed from:**

- Datum Stage one, Plan exit: every experiment recorded before the phase that
  depends on it.
- Specification section 4, experiments 1, 2 and 3, on both machines.
- Specification TE5 and TE6, assumptions to verify by experiments 1 and 2.
- Specification TE7, assumption to verify by experiment 3.

**What is done instead:**

- Plan and Execute proceed before the Stream Deck XL is in hand.
- Protocol values (Appendix A rows marked Protocol) are read from open source
  implementations that document the device, chiefly the python-elgato-streamdeck
  source, with the repository, commit and date read recorded in
  `docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`.
- Platform, deep link and GitHub values are read from their published
  documentation, recorded the same way.
- Every such value is marked in the evaluation record as read from source and
  not yet confirmed on hardware, or not yet confirmed by experiment.
- Experiments 1, 2 and 3 run when the deck arrives. Where an experiment
  contradicts a recorded value, the record is corrected and dated first
  (datum 0.2), then a failing test is written for the corrected value, then the
  code follows (datum 0.4).

**Reason:** the deck has not arrived. It is expected on 12 November 2026. The
aim is to build everything that can be built against understood interfaces and
documented open source today, so that what remains when the deck arrives is
small adjustment and polish.

**Author's decision:** in the author's words, "We're recording now that we are
building to understood apis and open source things that document the stream
deck. We are making best effort in building everything we need today so when
the stream deck arrives at some point tomorrow it's just small tweaks and polish
needed." Given on 11 November 2026.

**Covering tests:**

- TE5 byte level fixtures are derived from the recorded reference source, so a
  value corrected by experiment changes exactly one fixture and one table row.
- No author gate (TE4, TE6, TE7, TE8, TE9, TE10) is claimed or inferred by this
  deviation. Each stays outstanding until the author clears it on the device.

**Closes when:** experiments 1, 2 and 3 are recorded on both machines and every
affected Appendix A row is marked verified.
