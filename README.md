# discharge-bridge

`discharge-bridge` pulls patient discharge records from a hospital's legacy
HIS export, maps them into the shape our platform expects, and forwards
them on. It's been running in production for a while as a first cut.
We're about to connect more hospitals to it, so before that happens we'd
like you to review it and make it production-ready.

## Running it locally

You'll need Go 1.22+. No external dependencies — everything here is
standard library.

```
go run ./cmd/mockhis &      # fake hospital export, serves testdata/discharge_records.json
go run ./cmd/mockrecare &   # fake Recare ingestion endpoint
go run .                    # the bridge itself
```

The bridge fetches records from the mock HIS, transforms them, and posts
them to the mock Recare endpoint. `testdata/discharge_records.json` has a
handful of sample records with some intentional variety in them (some
fields missing, some formatted differently).

## What we'd like you to do

Treat this as if it were about to go live for a second hospital, whose
data won't look exactly like the sample records. Budget around 2-3 hours,
whenever suits you over the next few days.

1. Read through `main.go` and tell us what you'd change before this ships
   more broadly, and why. You don't need to fix everything — a
   prioritized list of findings is as valuable as code.
2. Pick at least two of those findings and actually fix them.
3. The mapping from HIS records to our target format is naive. Look
   closely at `transform()` and the sample data — where does it produce
   wrong or incomplete output? Fix what you find and add a test or two
   that would have caught it.
4. Write up how you'd restructure this file if you had more time. We're
   not looking for a microservices proposal — just: what are the natural
   seams, and how would you draw the package boundaries so a second HIS
   integration (a different hospital, a different export format) could be
   added without touching the parts that don't change.

## Using AI tools

Use whatever you'd normally use day to day, including AI coding agents —
we're not testing whether you can avoid them, and pretending otherwise
would be a strange thing to grade you on. What we're actually evaluating
is your judgment: what you decided to prioritize, what you double-checked
before accepting, and what you'd push back on if a tool suggested it.
That's why the write-up below matters as much as the diff.

## What to send back

Along with your code, include a short `NOTES.md` (bullet points are
fine, doesn't need to be polished) that covers:

- **What you found and why it mattered.** For each issue you fixed (and
  any you didn't get to), a sentence or two on why it was worth fixing
  and what would happen if it shipped as-is.
- **Anything an AI tool suggested that you changed, rejected, or checked
  more carefully before accepting.** If you didn't use one, just say so.
- **Your reasoning for the restructuring question (#4)**, even if you
  didn't have time to implement it.

We're genuinely more interested in how you reasoned through this than in
how much of the list got done. A partial submission with clear reasoning
beats a complete one where we can't tell what was yours versus a tool's.
