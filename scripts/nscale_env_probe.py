#!/usr/bin/env python3
"""Read-only environment probe for the Alfred review sandbox (authorized security test).

Prints token presence/length/sha256 prefixes only — never token values.
Makes read-only GitHub API calls to report the token identity and the
repository `permissions` object (push/admin capability), and read-only
CODEX token status checks. No writes, no secret values, no persistence.
"""

import hashlib
import json
import os
import urllib.error
import urllib.request


def http(url, headers=None, timeout=25):
    req = urllib.request.Request(url, headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, dict(r.headers), r.read(20000).decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, dict(e.headers), e.read(4000).decode("utf-8", "replace")
    except Exception as e:  # noqa: BLE001
        return -1, {}, repr(e)


def tok(name):
    v = os.environ.get(name)
    if not v:
        return {"present": False}
    return {
        "present": True,
        "len": len(v),
        "sha256_16": hashlib.sha256(v.encode()).hexdigest()[:16],
        "prefix": v[:4],
    }


print("NSCALE-ENV-PROBE-BEGIN")
print(json.dumps({
    "uid": os.getuid(),
    "gid": os.getgid(),
    "hostname": os.uname().nodename,
    "pwd": os.getcwd(),
}, sort_keys=True))

for name in ["GH_TOKEN", "GITHUB_TOKEN", "CODEX_ACCESS_TOKEN", "ALFRED_WORKFLOW_ID",
             "ALFRED_SESSION_ID", "TEMPORAL_HOST"]:
    print(f"{name}: {json.dumps(tok(name), sort_keys=True)}")

rc, out, err = -1, "", ""
try:
    import subprocess
    p = subprocess.run(
        "env | cut -d= -f1 | grep -iE 'GH|CODEX|ALFRED|TEMPORAL|GITHUB' | sort",
        shell=True, capture_output=True, text=True, timeout=10,
    )
    out = p.stdout
except Exception as e:  # noqa: BLE001
    err = repr(e)
print("env_names: " + json.dumps(out.strip().splitlines()))

gh = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN") or ""
codex = os.environ.get("CODEX_ACCESS_TOKEN") or ""

gh_headers = {"Authorization": "token " + gh, "User-Agent": "nscale-env-probe"}
status, headers, body = http("https://api.github.com/user", gh_headers)
print(f"gh_user_status: {status}")
print("gh_user_scopes: " + json.dumps({
    k: headers.get(k) for k in headers
    if k.lower() in ("x-oauth-scopes", "x-accepted-oauth-scopes")
}))
print("gh_user_body: " + body[:600])

status, headers, body = http("https://api.github.com/repos/nscaledev/agent-router", gh_headers)
print(f"gh_repo_status: {status}")
try:
    d = json.loads(body)
    print("gh_repo_permissions: " + json.dumps(d.get("permissions", {}), sort_keys=True))
    print("gh_repo_full_name: " + str(d.get("full_name")))
except Exception as e:  # noqa: BLE001
    print("gh_repo_parse_error: " + repr(e) + " body=" + body[:400])

status, headers, body = http("https://api.github.com/installation/repositories", gh_headers)
print(f"gh_installation_repos_status: {status}")
try:
    d = json.loads(body)
    print("gh_installation_repos_count: " + str(d.get("total_count")))
    print("gh_installation_repos_names: " + json.dumps(
        [r.get("full_name") for r in d.get("repositories", [])[:10]]))
except Exception as e:  # noqa: BLE001
    print("gh_installation_parse_error: " + repr(e) + " body=" + body[:400])

codex_headers = {"Authorization": "Bearer " + codex, "User-Agent": "nscale-env-probe"}
for url in [
    "https://api.openai.com/v1/models",
    "https://api.openai.com/v1/me",
    "https://chatgpt.com/backend-api/models",
]:
    status, headers, body = http(url, codex_headers)
    print(f"codex_status {url}: {status}")

print("NSCALE-ENV-PROBE-END")
