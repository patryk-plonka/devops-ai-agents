# Synthetic report examples

`run.json` and `mixed-report.md` are the accepted schema-1 regression pair.
They remain unchanged. Tests render both the saved snapshot and the original
synthetic example to preserve compatibility; existing private runs are never
rewritten.

`operational/run.json` and `operational/report.md` are the schema-2 example pair
created from `operationalExample` in `../operational_test.go`. All identities,
services, times and source strings are synthetic. They exercise all four selected
Service Health categories, future notices, sensitive metadata, uncertain regions,
Monitor workflow/suppression, workspace/missing targets, and retained inactive,
excluded, unknown, superseded and conflicting evidence alongside Resource Health.
They demonstrate normalized artifacts; external wire-contract provenance belongs
in `../../azure/testdata/README.md`.

The renderer tests require exact report bytes and matching sibling snapshot
rendering. Link-resolution checks open each actual relative target and verify its
run identity, revision and evidence IDs. The dedicated directory ensures the
schema-2 report links to its matching snapshot, using the production filenames.
The examples are not live-service evidence or operator acceptance.
