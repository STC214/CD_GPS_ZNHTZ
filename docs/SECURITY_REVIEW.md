# Security Review

Review date: 2026-04-28

## Scope

This review covers the local code paths that can affect user data, network traffic, process execution, and generated files.

## Findings

No evidence was found for backdoors, remote-control listeners, keylogging, registry persistence, reverse shells, or intentional local-file exfiltration.

Expected sensitive behavior:

- The app opens user-supplied supported-site URLs.
- Browser-backed flows launch Firefox through Playwright.
- Hitomi resolves metadata and image URLs with backend HTTP requests.
- Image assets are downloaded to the configured output directory.
- Logs, task reports, thumbnails, and frontend settings are persisted under the runtime root.
- A browser install helper can download Playwright-managed Firefox when explicitly invoked.

## Authorization-Safe Documentation Policy

Documentation should avoid examples that ask users to weaken local security policy or copy real browser credentials.

Do not add:

- PowerShell execution-policy bypass examples.
- Examples that copy a personal Firefox profile into the project.
- Proxy URLs containing usernames, passwords, tokens, or private endpoints.
- Account-gated or private gallery URLs.
- Instructions that require administrative privilege unless the code path genuinely requires it.

Use neutral placeholders for smoke-test URLs and ask testers to provide their own local test inputs.

## Current Safety Controls

- Firefox task runs use fresh temporary Playwright profiles.
- Temporary profiles are removed after the task leaves the running state when possible.
- Download folder names are sanitized for Windows path rules.
- Image downloads use normalized, de-duplicated URL lists.
- Backend HTTP downloads support cancellation through context propagation.
- Proxy strings are normalized before use.
- Unsupported sites can be blocked at add time.

## Residual Risks

- Logs include URLs and local paths.
- Image downloads do not yet enforce a maximum response-size limit.
- Proxy values are stored as settings; avoid embedding credentials in proxy strings.
- Browser install downloads rely on the Playwright package ecosystem and should be run only from trusted builds.

## Recommendations

- Add optional log redaction before sharing logs externally.
- Add maximum response-size limits for image downloads.
- Keep fresh temporary profiles as the default profile policy.
- Keep documentation free of authorization-bypass commands and credential-bearing examples.
