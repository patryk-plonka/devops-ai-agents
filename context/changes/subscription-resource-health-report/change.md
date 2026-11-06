---
change_id: subscription-resource-health-report
title: Inspect one subscription's resource health
status: impl_reviewed
created: 2026-11-05
updated: 2026-11-06
archived_at: null
---

## Notes

Roadmap S-01: create the first manually invoked Go CLI flow for one Azure subscription, with existing-work-identity access, Resource Health evidence, and an explicitly incomplete Markdown report.

The user approved the four-phase structure on 2026-11-05. All four phases and all 18 Progress rows are now complete. Automated checks and phase reviews passed, including the approved fixes and regressions for issues discovered during live validation. On 2026-11-06 the operator confirmed "all manual tests are ok", completing live acceptance. The sanitized outcomes and remaining scenario limitations are in `verification.md`; phase reviews are under `reviews/`. S-01 is implemented and accepted, while overall coverage remains intentionally incomplete until the other sources are implemented. Work remains uncommitted at the user's request; completed Progress rows therefore have no commit SHA. No phase or epilogue commit was created.

Operator decisions confirmed during planning:

- Use the operator's guest access through an existing Azure CLI session established with `az login`; do not ask for a pasted token.
- Support public Azure only.
- Store run records and Markdown reports in a directory inside this repository, excluded through `.gitignore`, with a structure that supports traceability.
- Reports are for the operator alone at present.
- Retain subscription and tenant IDs, resource names and IDs, and resource tags. Exclude credentials, secrets, and unrelated personal details; apply this rule to tags as well.
- On Ctrl+C, save gathered evidence and an explicitly incomplete report.
- The operator supplied a target and completed live acceptance on 2026-11-06. Initial attempts exposed an account-schema bug, confusion between tenant and subscription IDs, and a Resource Health continuation-validation bug. Both bugs are corrected and regression-tested. Live runs by resolved name and UUID completed with exit 0 and matching scope/report revisions. The operator confirmed all manual checks, including intended scope, Resource Health comparison, report review, and interruption. Guest access and adverse scenarios not explicitly exercised remain synthetic-only. Actual identifiers/evidence remain outside tracked documents. See `verification.md`.

Seven open questions have been answered, including a follow-up clarification of identifier and tag retention. The user explicitly requested open questions instead of multiple-choice questions.

The investigation of why manual checks required CLI-schema/pagination fixes and
why planning/testing missed them is in `manual-check-investigation.md`. It also
separates the tenant/subscription ID confusion and table-output guidance defect
from the two application bugs. A recurring lesson is awaiting the user-authored
fields and confirmation required by `/10x-lesson`.

The full S-01 implementation review is saved in `reviews/impl-review.md`:
APPROVED with one minor cancellation warning (F1) and one retained-artifact
feedback observation (F2), both fixed and regression-tested on the operator's
"fix both" instruction on 2026-11-06. No full-review decisions remain pending.
Required automated checks pass;
all 18 Progress rows and operator acceptance remain complete. Both requested
parallel reviewers hit a usage limit, so the primary reviewer completed the
full review directly and recorded that limitation. The subsequent triage fixes
preserve saved artifacts, accepted Progress, and manual acceptance. No staging
or commits were performed.
