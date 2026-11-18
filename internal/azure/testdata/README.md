# Synthetic Azure wire fixtures

These literal files are maintained independently of Go structs and request
builders. They contain synthetic data only. They establish documented
compatibility, not live service acceptance. Tests use fixed literal expected
requests as a second independent boundary.

## Service Health

[Events List By Subscription Id, API 2025-05-01](https://learn.microsoft.com/en-us/rest/api/resourcehealth/events/list-by-subscription-id?view=rest-resourcehealth-2025-05-01)
was inspected on 2026-11-15. `service-health-events.json` follows the example's
`value`/`nextLink` envelope, global event ID, `properties`, `eventSource` of
ResourceHealth, service/region impact nesting, and `$skipToken` continuation
without `queryStartTime`. The initial date query uses the reference's M/D/YYYY
syntax and last-update semantics.

Intentional substitutions: subscription IDs, event IDs, timestamps, token and
all narrative/identity canaries are synthetic. Most optional example fields are
omitted. The first event omits the optional sensitivity flag. Its mixed-scope
and foreign-only regions exercise filtering. A second synthetic SecurityAdvisory
contains only permitted metadata and `isEventSensitive=true`; no detail retrieval
is needed. Test variants cover the other categories, scoped IDs, malformed
siblings and unknown enums. Nil impact/region cases are adversarial variants.

## Azure Monitor

[Alerts Get All, API 2019-03-01](https://learn.microsoft.com/en-us/rest/api/alerts-management/alerts/alerts/get-all?view=rest-alerts-management-alerts-2019-03-01)
was inspected on 2026-11-15. `monitor-alerts.json` follows the example's nested
`properties.essentials`, `actionStatus.isSuppressed`, extension alert identity,
source condition distinct from response, and `ctoken`/explicit HTTPS port 443
continuation syntax. The documented initial parameters are asserted literally:
30d, Fired, false context/egress, pageCount 100. No signalType query is supported.

Intentional substitutions: IDs, resources, times, token and excluded text are
synthetic. The example's continuation switches subscriptions, uses a preview API
and 1d; the positive fixture explicitly replaces these with the selected scope,
2019-03-01 and 30d. Tests retain the original scope/version/window changes as
negative cases. The second synthetic alert is Metric/Fired/Closed/suppressed to
exercise independent source condition. Context and egress canaries are included
despite false request flags to prove they cannot enter normalized checkpoints.
Additional adversarial fields and malformed records are constructed in tests.
