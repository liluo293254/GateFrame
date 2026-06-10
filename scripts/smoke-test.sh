#!/usr/bin/env bash
# End-to-end smoke test: Gateway + user/audit/workflow/search/notification/file + Prometheus via ddev
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

GATEWAY_URL="${GATEWAY_URL:-http://127.0.0.1:3002}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://127.0.0.1:9090}"

echo "Gateway URL: ${GATEWAY_URL}"

echo "==> health"
curl -sf "${GATEWAY_URL}/health" | grep -q ok

echo "==> login"
RESP="$(curl -sf -X POST "${GATEWAY_URL}/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin@123456"}')"
echo "${RESP}" | grep -q '"token"'

TOKEN="$(echo "${RESP}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])")"

echo "==> permissions"
curl -sf "${GATEWAY_URL}/api/auth/permissions" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q user.read

echo "==> oidc config (disabled in dev without IdP)"
curl -sf "${GATEWAY_URL}/api/auth/oidc/config" | grep -q '"enabled":false'

echo "==> list users"
curl -sf "${GATEWAY_URL}/api/users" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q admin

echo "==> list roles"
curl -sf "${GATEWAY_URL}/api/rbac/roles" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q admin

echo "==> list workflows"
curl -sf "${GATEWAY_URL}/api/workflows" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q 'Sample approval'

echo "==> create workflow"
WF_NAME="smoke-wf-$(date +%s)"
CREATE_WF="$(curl -sf -X POST "${GATEWAY_URL}/api/workflows" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"${WF_NAME}\",\"description\":\"Smoke approval request\",\"category\":\"general\",\"priority\":\"normal\",\"requester_label\":\"admin\"}")"
echo "${CREATE_WF}" | grep -q '"id"'
WF_ID="$(echo "${CREATE_WF}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])")"

echo "==> submit workflow for review"
curl -sf -X POST "${GATEWAY_URL}/api/workflows/${WF_ID}/submit" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"actor_label":"admin"}' | grep -q '"status":"pending"'

echo "==> approve workflow"
curl -sf -X POST "${GATEWAY_URL}/api/workflows/${WF_ID}/approve" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"comment":"smoke approved","actor_label":"admin"}' | grep -q '"status":"approved"'

echo "==> delete workflow"
curl -sf -X DELETE "${GATEWAY_URL}/api/workflows/${WF_ID}" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q '"deleted":true'

echo "==> search documents"
curl -sf "${GATEWAY_URL}/api/search?q=welcome" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q 'Welcome guide'

echo "==> index search document"
SMOKE_DOC_TITLE="smoke-doc-$(date +%s)"
CREATE_DOC="$(curl -sf -X POST "${GATEWAY_URL}/api/search/documents" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"${SMOKE_DOC_TITLE}\",\"content\":\"smoke indexed content\"}")"
echo "${CREATE_DOC}" | grep -q '"id"'
DOC_ID="$(echo "${CREATE_DOC}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])")"

echo "==> search indexed document"
curl -sf "${GATEWAY_URL}/api/search?q=${SMOKE_DOC_TITLE}" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q "${SMOKE_DOC_TITLE}"

echo "==> delete search document"
curl -sf -X DELETE "${GATEWAY_URL}/api/search/documents/${DOC_ID}" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q '"deleted":true'

echo "==> list notifications"
curl -sf "${GATEWAY_URL}/api/notifications" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q 'Welcome'

echo "==> create notification"
NOTIF_TITLE="smoke-notif-$(date +%s)"
CREATE_NOTIF="$(curl -sf -X POST "${GATEWAY_URL}/api/notifications" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"${NOTIF_TITLE}\",\"body\":\"Smoke notification body\"}")"
echo "${CREATE_NOTIF}" | grep -q '"id"'
NOTIF_ID="$(echo "${CREATE_NOTIF}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])")"

echo "==> mark notification read"
curl -sf -X PATCH "${GATEWAY_URL}/api/notifications/${NOTIF_ID}/read" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{}' | grep -q '"read_at"'

echo "==> platform admin login (tenant lifecycle)"
PLATFORM_RESP="$(curl -sf -X POST "${GATEWAY_URL}/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"platform","password":"Platform@123456"}')"
PLATFORM_TOKEN="$(echo "${PLATFORM_RESP}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])")"

echo "==> admin forbidden on tenant.read"
ADMIN_TENANT_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/api/tenants" \
  -H "Authorization: Bearer ${TOKEN}")"
if [[ "${ADMIN_TENANT_CODE}" != "403" ]]; then
  echo "expected admin GET /api/tenants to return 403, got ${ADMIN_TENANT_CODE}" >&2
  exit 1
fi

echo "==> list tenants (platform)"
curl -sf "${GATEWAY_URL}/api/tenants" \
  -H "Authorization: Bearer ${PLATFORM_TOKEN}" | grep -q '"slug"'

echo "==> create tenant (platform)"
TENANT_SLUG="smoke-$(date +%s)"
curl -sf -X POST "${GATEWAY_URL}/api/tenants" \
  -H "Authorization: Bearer ${PLATFORM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Smoke Tenant\",\"slug\":\"${TENANT_SLUG}\"}" \
  | grep -q '"slug"'

echo "==> list files"
curl -sf "${GATEWAY_URL}/api/files" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q '\['

echo "==> reject disallowed file type (.exe)"
EXE_B64="$(printf 'MZ' | base64 | tr -d '\n')"
EXE_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/files" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"filename\":\"malware.exe\",\"content_type\":\"application/octet-stream\",\"content_base64\":\"${EXE_B64}\"}")"
if [[ "${EXE_CODE}" != "400" ]]; then
  echo "expected admin POST .exe to return 400, got ${EXE_CODE}" >&2
  exit 1
fi

echo "==> upload 11MB file (within 50MB limit)"
GATEWAY_URL="${GATEWAY_URL}" TOKEN="${TOKEN}" python3 <<'PY'
import base64, json, os, urllib.error, urllib.request

gateway = os.environ["GATEWAY_URL"]
token = os.environ["TOKEN"]
payload = {
    "filename": "eleven-mb.txt",
    "content_type": "text/plain",
    "content_base64": base64.b64encode(b"0" * (11 * 1024 * 1024)).decode(),
}
req = urllib.request.Request(
    f"{gateway}/api/files",
    data=json.dumps(payload).encode(),
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    method="POST",
)
with urllib.request.urlopen(req, timeout=180) as resp:
    body = json.loads(resp.read())
if "id" not in body.get("data", {}):
    raise SystemExit("11MB upload missing id")
print("11MB upload ok")
PY

echo "==> reject file over 50MB limit"
GATEWAY_URL="${GATEWAY_URL}" TOKEN="${TOKEN}" python3 <<'PY'
import base64, json, os, urllib.error, urllib.request

gateway = os.environ["GATEWAY_URL"]
token = os.environ["TOKEN"]
payload = {
    "filename": "too-large.txt",
    "content_type": "text/plain",
    "content_base64": base64.b64encode(b"0" * (50 * 1024 * 1024 + 1)).decode(),
}
req = urllib.request.Request(
    f"{gateway}/api/files",
    data=json.dumps(payload).encode(),
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    method="POST",
)
try:
    urllib.request.urlopen(req, timeout=180)
except urllib.error.HTTPError as err:
    if err.code == 400:
        print("oversize rejected with 400")
        raise SystemExit(0)
    raise SystemExit(f"expected 400, got {err.code}")
raise SystemExit("expected HTTPError for oversize upload")
PY

echo "==> upload file"
FILE_B64="$(printf 'smoke-file-%s' "$(date +%s)" | base64 | tr -d '\n')"
UPLOAD_FILE="$(curl -sf -X POST "${GATEWAY_URL}/api/files" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"filename\":\"smoke.txt\",\"content_type\":\"text/plain\",\"content_base64\":\"${FILE_B64}\"}")"
echo "${UPLOAD_FILE}" | grep -q '"id"'
FILE_ID="$(echo "${UPLOAD_FILE}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])")"

echo "==> download file content"
curl -sf "${GATEWAY_URL}/api/files/${FILE_ID}/content" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q 'smoke-file'

echo "==> update tenant (platform)"
curl -sf -X PUT "${GATEWAY_URL}/api/tenants/00000000-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer ${PLATFORM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Default Tenant"}' | grep -q '"status":"active"'

echo "==> dashboard stats"
curl -sf "${GATEWAY_URL}/api/dashboard/stats" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q '"users"'

echo "==> role permissions"
curl -sf "${GATEWAY_URL}/api/rbac/role-permissions" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q user.read

echo "==> rbac PUT audited (admin)"
ADMIN_ROLE_ID="$(curl -sf "${GATEWAY_URL}/api/rbac/roles" \
  -H "Authorization: Bearer ${TOKEN}" \
  | python3 -c "import json,sys; d=json.load(sys.stdin)['data']; print(next(r['id'] for r in d if r['name']=='admin'))")"
ADMIN_PERMS_JSON="$(curl -sf "${GATEWAY_URL}/api/rbac/role-permissions" \
  -H "Authorization: Bearer ${TOKEN}" \
  | python3 -c "import json,sys; d=json.load(sys.stdin)['data']; rid='${ADMIN_ROLE_ID}'; print(json.dumps(next(b['permissions'] for b in d if b['role_id']==rid)))")"
STALE_TOKEN="${TOKEN}"
curl -sf -X PUT "${GATEWAY_URL}/api/rbac/roles/${ADMIN_ROLE_ID}/permissions" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"permissions\": ${ADMIN_PERMS_JSON}}" | grep -q user.read

sleep 1

echo "==> stale JWT rejected after role permission bump (perm_ver)"
STALE_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/api/users" \
  -H "Authorization: Bearer ${STALE_TOKEN}")"
if [[ "${STALE_CODE}" != "401" ]]; then
  echo "expected stale token after rbac PUT to return 401, got ${STALE_CODE}" >&2
  exit 1
fi

echo "==> re-login after perm_ver bump"
RESP="$(curl -sf -X POST "${GATEWAY_URL}/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin@123456"}')"
TOKEN="$(echo "${RESP}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])")"

echo "==> audit log contains rbac PUT"
curl -sf "${GATEWAY_URL}/api/audit?path=/api/rbac/roles" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q 'PUT'

echo "==> viewer forbidden on rbac.manage"
VIEWER_RESP="$(curl -sf -X POST "${GATEWAY_URL}/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"viewer","password":"Viewer@123456"}')"
VIEWER_TOKEN="$(echo "${VIEWER_RESP}" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['token'])")"
VIEWER_PUT_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${GATEWAY_URL}/api/rbac/roles/${ADMIN_ROLE_ID}/permissions" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"permissions":["user.read"]}')"
if [[ "${VIEWER_PUT_CODE}" != "403" ]]; then
  echo "expected viewer PUT to return 403, got ${VIEWER_PUT_CODE}" >&2
  exit 1
fi

echo "==> viewer forbidden on workflow.read"
VIEWER_WF_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/api/workflows" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}")"
if [[ "${VIEWER_WF_CODE}" != "403" ]]; then
  echo "expected viewer GET /api/workflows to return 403, got ${VIEWER_WF_CODE}" >&2
  exit 1
fi

echo "==> viewer forbidden on workflow.manage"
VIEWER_WF_POST_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/workflows" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"blocked","description":"x","status":"draft"}')"
if [[ "${VIEWER_WF_POST_CODE}" != "403" ]]; then
  echo "expected viewer POST /api/workflows to return 403, got ${VIEWER_WF_POST_CODE}" >&2
  exit 1
fi

echo "==> viewer forbidden on search.read"
VIEWER_SEARCH_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/api/search?q=welcome" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}")"
if [[ "${VIEWER_SEARCH_CODE}" != "403" ]]; then
  echo "expected viewer GET /api/search to return 403, got ${VIEWER_SEARCH_CODE}" >&2
  exit 1
fi

echo "==> viewer forbidden on search.manage"
VIEWER_SEARCH_POST_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/search/documents" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"title":"blocked","content":"x"}')"
if [[ "${VIEWER_SEARCH_POST_CODE}" != "403" ]]; then
  echo "expected viewer POST /api/search/documents to return 403, got ${VIEWER_SEARCH_POST_CODE}" >&2
  exit 1
fi

echo "==> viewer forbidden on notification.manage"
VIEWER_NOTIF_POST_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/notifications" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"title":"blocked","body":"x"}')"
if [[ "${VIEWER_NOTIF_POST_CODE}" != "403" ]]; then
  echo "expected viewer POST /api/notifications to return 403, got ${VIEWER_NOTIF_POST_CODE}" >&2
  exit 1
fi

echo "==> viewer forbidden on tenant.read"
VIEWER_TENANT_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/api/tenants" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}")"
if [[ "${VIEWER_TENANT_CODE}" != "403" ]]; then
  echo "expected viewer GET /api/tenants to return 403, got ${VIEWER_TENANT_CODE}" >&2
  exit 1
fi

echo "==> viewer can list files (file.read)"
curl -sf "${GATEWAY_URL}/api/files" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}" | grep -q '\['

echo "==> viewer forbidden on file.manage upload"
VIEWER_FILE_POST_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/files" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"filename\":\"blocked.txt\",\"content_type\":\"text/plain\",\"content_base64\":\"$(printf blocked | base64 | tr -d '\n')\"}")"
if [[ "${VIEWER_FILE_POST_CODE}" != "403" ]]; then
  echo "expected viewer POST /api/files to return 403, got ${VIEWER_FILE_POST_CODE}" >&2
  exit 1
fi

echo "==> viewer forbidden on file.manage delete"
VIEWER_FILE_DEL_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${GATEWAY_URL}/api/files/${FILE_ID}" \
  -H "Authorization: Bearer ${VIEWER_TOKEN}")"
if [[ "${VIEWER_FILE_DEL_CODE}" != "403" ]]; then
  echo "expected viewer DELETE /api/files/:id to return 403, got ${VIEWER_FILE_DEL_CODE}" >&2
  exit 1
fi

echo "==> delete file"
curl -sf -X DELETE "${GATEWAY_URL}/api/files/${FILE_ID}" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q '"deleted":true'

echo "==> admin forbidden on tenant.manage"
ADMIN_TENANT_PUT_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${GATEWAY_URL}/api/tenants/00000000-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Blocked"}')"
if [[ "${ADMIN_TENANT_PUT_CODE}" != "403" ]]; then
  echo "expected admin PUT /api/tenants/:id to return 403, got ${ADMIN_TENANT_PUT_CODE}" >&2
  exit 1
fi

echo "==> platform admin forbidden on workflow.manage"
PLATFORM_WF_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/workflows" \
  -H "Authorization: Bearer ${PLATFORM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"blocked","description":"x","status":"draft"}')"
if [[ "${PLATFORM_WF_CODE}" != "403" ]]; then
  echo "expected platform POST /api/workflows to return 403, got ${PLATFORM_WF_CODE}" >&2
  exit 1
fi

echo "==> create user (audit)"
NEW_USER="smoke$(date +%s)"
curl -sf -X POST "${GATEWAY_URL}/api/users" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${NEW_USER}\",\"password\":\"SmokeTest1\",\"display_name\":\"Smoke\"}" \
  | grep -q '"id"'

sleep 1

echo "==> audit log"
curl -sf "${GATEWAY_URL}/api/audit" \
  -H "Authorization: Bearer ${TOKEN}" | grep -q '/api/users'

echo "==> gateway prometheus metrics"
curl -sf "${GATEWAY_URL}/metrics" | grep -q login_failures_total
curl -sf "${GATEWAY_URL}/metrics" | grep -q audit_persist_failures_total

if curl -sf "${PROMETHEUS_URL}/-/healthy" >/dev/null 2>&1; then
  echo "==> prometheus scrape"
  curl -sf "${PROMETHEUS_URL}/api/v1/query?query=login_failures_total" | grep -q '"status":"success"'
else
  echo "==> prometheus (skipped — start ddev with docker-compose.prometheus.yaml)"
fi

echo "==> login rate limit (failed attempts)"
# Must exceed LOGIN_RATE_LIMIT_MAX in .ddev/docker-compose.backend.yaml (default 10).
LOGIN_RL_MAX=10
RATE_CODE=""
for ((i = 1; i <= LOGIN_RL_MAX + 1; i++)); do
  RATE_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"wrong-password"}')"
done
if [[ "${RATE_CODE}" != "429" ]]; then
  echo "expected 429 on repeated failed login, got ${RATE_CODE}" >&2
  exit 1
fi

echo "==> clear login rate limit (dev cleanup)"
"${ROOT}/scripts/clear-login-rate-limit.sh" >/dev/null

echo "==> disable default tenant blocks refresh (platform)"
curl -sf -X PUT "${GATEWAY_URL}/api/tenants/00000000-0000-0000-0000-000000000001" \
  -H "Authorization: Bearer ${PLATFORM_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"status":"disabled"}' | grep -q '"status":"disabled"'
REFRESH_CODE="$(curl -s -o /dev/null -w "%{http_code}" -X POST "${GATEWAY_URL}/api/auth/refresh" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{}')"
if [[ "${REFRESH_CODE}" != "401" ]]; then
  echo "expected refresh on disabled tenant to return 401, got ${REFRESH_CODE}" >&2
  exit 1
fi
USERS_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${GATEWAY_URL}/api/users" \
  -H "Authorization: Bearer ${TOKEN}")"
if [[ "${USERS_CODE}" != "401" ]]; then
  echo "expected GET /api/users on disabled tenant to return 401, got ${USERS_CODE}" >&2
  exit 1
fi
DB_CONTAINER="ddev-gateframe-db"
if docker ps --format '{{.Names}}' | grep -qx "${DB_CONTAINER}"; then
  docker exec "${DB_CONTAINER}" psql -U db -d db -c \
    "UPDATE tenants SET status='active', name='Default Tenant' WHERE id='00000000-0000-0000-0000-000000000001'" \
    >/dev/null
else
  echo "warning: ${DB_CONTAINER} not running; re-enable default tenant manually before next smoke run" >&2
fi

echo "ALL SMOKE TESTS PASSED"
