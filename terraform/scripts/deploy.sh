#!/usr/bin/env bash
# Publishes a zip package to a Flex Consumption function app using the One Deploy endpoint
# (/api/publish), mirroring `az functionapp deployment source config-zip`. The azurerm provider's
# `zip_deploy_file` uses the classic Kudu zipdeploy endpoints, which Flex apps don't serve.
#
# Expects: SCM_HOST, DEPLOY_USER, DEPLOY_PASSWORD, PACKAGE
set -euo pipefail

: "${SCM_HOST:?}" "${DEPLOY_USER:?}" "${DEPLOY_PASSWORD:?}" "${PACKAGE:?}"

base="https://${SCM_HOST}"
body="$(mktemp)"
trap 'rm -f "$body"' EXIT

# Passes credentials through a config file descriptor so they never appear in argv or output.
scm() {
  curl -sS -K <(printf 'user = "%s:%s"\n' "$DEPLOY_USER" "$DEPLOY_PASSWORD") -o "$body" -w '%{http_code}' "$@" || echo "000"
}

# Waits for the most recent deployment to finish. Returns 0 on success, 1 on failure/timeout.
await_deployment() {
  local code status
  for _ in $(seq 1 120); do
    sleep 5
    code="$(scm "${base}/api/deployments/latest")"
    status="$(sed -nE 's/.*"status"[[:space:]]*:[[:space:]]*(-?[0-9]+).*/\1/p' "$body" | head -n1)"
    case "$status" in
      4) echo "Deployment succeeded."; return 0 ;;
      3 | -1 | 5 | 6)
        echo "Deployment ended with status ${status} (HTTP ${code}):"
        cat "$body"; echo
        return 1 ;;
    esac
  done
  echo "Timed out waiting for the deployment to complete (last HTTP ${code}, status '${status:-unknown}')."
  return 1
}

# A freshly created app can take a few minutes before its SCM site is routable and its managed
# identity's role assignment on the deployment container has propagated, so retry generously.
attempts=12
for attempt in $(seq 1 "$attempts"); do
  echo "Publishing $(basename "$PACKAGE") to ${SCM_HOST} (attempt ${attempt}/${attempts})..."
  code="$(scm -X POST \
    -H 'Content-Type: application/zip' \
    -H 'Cache-Control: no-cache' \
    --data-binary "@${PACKAGE}" \
    "${base}/api/publish?RemoteBuild=false&Deployer=terraform")"

  case "$code" in
    200 | 202)
      if await_deployment; then
        exit 0
      fi ;;
    *)
      echo "Publish endpoint responded with HTTP ${code}:"
      head -c 2000 "$body"; echo ;;
  esac

  sleep 30
done

echo "Giving up after ${attempts} attempts."
exit 1
