# License Roadmap — AsymmFlow

**Today, AsymmFlow is licensed under the [GNU AGPL-3.0](LICENSE).**
**Every release also becomes [MIT](#the-mit-destination) two years after it ships.**

This document is a public, standing commitment — not marketing. It explains the
"why," the exact mechanism, and what you can rely on.

---

## The short version

| Age of a release | License it is available under |
|---|---|
| 0–2 years old | **AGPL-3.0** (strong copyleft — share-alike, including over a network) |
| 2+ years old | **MIT** (do anything, no strings) — automatically, forever |

There is *always* a free MIT version of AsymmFlow trailing two years behind the
frontier. The newest work stays AGPL for two years, then joins the commons. The
giving never stops; it just trails by two years. 🌱

## Why we do it this way

We almost lost our hair (and some sanity) building this. We want builders and
businesses to genuinely benefit from that work — and we have bills and dreams too.
AGPL-3.0 keeps the project honest while it's young: if you run a modified AsymmFlow
as a service, you share your changes back. That's the deal that lets a tiny team
keep the lights on without locking anything away.

But we also believe value given freely to the world comes back in ways you don't
expect. So nothing stays locked. On a fixed, automatic, two-year clock, every line
we write becomes MIT — yours to do anything with, no permission, no payment, no
asterisk. We'd rather give people a clock than a mood: the conversion is dated and
inevitable, so you never have to trust our future selves.

We only ever *loosen*. We will never tighten a license that has already been
granted. A version released under AGPL today will be MIT in two years no matter what
happens to us or to this project.

## The mechanism (how the clock actually works)

1. Every release is git-tagged with its publication date (e.g. `v1.4.0`).
2. On the **second anniversary** of a release's tag date, that release — its exact
   source tree at that tag — becomes available under the MIT license below.
3. We record each conversion in the project's release notes and, where practical,
   re-publish the converted tag under MIT so it's unambiguous. Even if we
   forget to re-publish, **the grant in this document is the binding commitment**: a
   release is MIT-licensed from its second anniversary by virtue of this roadmap.
4. The "frontier" (anything less than two years old, including `main`) remains
   AGPL-3.0 until its own clock runs out.

This is the same idea behind the
[Functional Source License](https://fsl.software/), adapted so that AsymmFlow is a
*real, OSI-approved open-source license from day one* (AGPL), not source-available.

## What this means for you

- **Self-hosting an unmodified AsymmFlow?** AGPL asks nothing of you. Run it, own
  your data, enjoy.
- **A builder implementing AsymmFlow for a client?** Totally fine under AGPL — that's
  consulting, not redistribution of a closed competing service. (And in two years
  the version you used is MIT anyway.)
- **Modifying it and running it as a network service?** While it's AGPL (< 2 years),
  publish your changes under AGPL. After two years, do whatever you like under MIT.
- **Want it under MIT *right now*, before the clock?** Talk to us — see
  [`CLA.md`](CLA.md) for why we can offer that, and open a discussion.

## The MIT destination

When a release converts, it is offered under the standard MIT License:

```
MIT License

Copyright (c) 2026 Sarat Chandra Gnanamgari and Rahul Sinha

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

---

*This roadmap is a commitment by the AsymmFlow maintainers — Sarat Chandra
Gnanamgari and Rahul Sinha. It is not legal advice.
The [`LICENSE`](LICENSE) file (AGPL-3.0) governs your current rights; this document
governs how those rights expand over time.*
