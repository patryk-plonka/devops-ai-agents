#!/bin/sh
# Synthetic Azure CLI only. Unknown commands fail instead of invoking a real CLI.
set -eu
printf '%s\n' "$*" >> "$FAKE_AZ_LOG"
case "$*" in
  'cloud show --output json --only-show-errors')
    printf '%s\n' '{"name":"AzureCloud"}'
    ;;
  'account list --all --output json --only-show-errors')
    if [ "${FAKE_AZ_MODE:-}" = missing ]; then
      printf '%s\n' '[]'
    else
      printf '%s\n' '[{"id":"22222222-2222-2222-2222-222222222222","name":"Default","tenantId":"44444444-4444-4444-4444-444444444444","cloudName":"AzureCloud","isDefault":true,"user":{"type":"user","name":"profile-canary"}},{"id":"11111111-1111-1111-1111-111111111111","name":"Synthetic Guest","tenantId":"33333333-3333-3333-3333-333333333333","cloudName":"AzureCloud","isDefault":false,"user":{"type":"user","name":"profile-canary"}}]'
    fi
    ;;
  'account get-access-token -o json --resource https://management.core.windows.net/ --subscription 11111111-1111-1111-1111-111111111111')
    if [ "${FAKE_AZ_MODE:-}" = expired ]; then
      printf '%s\n' 'Bearer login-secret-canary' >&2
      exit 1
    fi
    printf '%s\n' '{"accessToken":"synthetic-token-canary","expires_on":4102444800}'
    ;;
  *) exit 99 ;;
esac
