# Synthetic report examples

Tests render the original schema-1 synthetic example and its saved snapshot
to preserve compatibility. [Report](mixed-report.md) · [Snapshot](run.json)

`operational/run.json` and `operational/report.md` are the schema-2 example pair
created from `operationalExample` in `../operational_test.go`. All identities,
services, times and source strings are synthetic. They exercise all four selected
Service Health categories, future notices, sensitive metadata, uncertain regions,
Monitor workflow/suppression, workspace/missing targets, and retained inactive,
excluded, unknown, superseded and conflicting evidence alongside Resource Health.
They demonstrate normalized artifacts. [Report](operational/report.md) ·
[Snapshot](operational/run.json) · [Wire fixtures](../../azure/testdata/README.md)

[Schema 3](advisor/report.md) and its [sibling snapshot](advisor/run.json) come
from `advisorExample` in `../advisor_test.go`. The example contains repeated
recommendation types, an exact duplicate, all five displayed workflow classes,
unknown targets, and a Dismissed observation that supersedes a New observation.
Use the row's evidence link and ID to find the saved instance and its source times.

[Schema 4](assessment/report.md) and its [sibling snapshot](assessment/run.json)
come from `assessmentExample` in `../assessment_test.go`. The example traces an OK
assessment to a newer Available observation, an empty Service Health query and
Postponed Advisor context. Another resource shows both a known Fired alert and
missing Resource Health evidence.

The renderer tests require exact report bytes and matching sibling snapshot
rendering. Link-resolution checks open each actual relative target and verify its
run identity, revision and references. The schema-2 through schema-4 directories
each pair a report with its matching snapshot, using the production filenames.
All examples use synthetic evidence.
