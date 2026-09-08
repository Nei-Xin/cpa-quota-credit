# v1.0.9

Public plugin resource routes now serve only static dashboard HTML. Statistics require the CPA management key at `/v0/management/plugins/cpa-quota-credit/stats`; the dashboard and userscript accept the key in a page-local input. The public stats endpoint and `format=json` statistics bypass have been removed (issue #1).

This release also reads CPA's `config_yaml` lifecycle payload, fixes management route registration, corrects OpenAI fallback rates and long-context pricing, and marks missing model or Priority/Fast rates as unpriced instead of silently substituting prices. Dashboard amounts are labeled as API cost estimates.

## Upgrade

- Replace the plugin and update the userscript. Enter your CPA management key to load statistics.
- Merge `pricing.overrides` from `config.example.yaml` into your plugin configuration. The current default remote feed still contains older GPT-5.6 Sol prices; the explicit override takes precedence. Review the override when official prices change.
- Apply `patches/cpa-usage-context.patch` to compatible CPA source with `git apply`, then rebuild CPA. Local integration found that CPA can cancel asynchronous plugin usage delivery when the HTTP request ends. The patch detaches usage delivery from request cancellation. It has not been published upstream; this plugin release alone does not fix that host behavior.
- Restart CPA. Existing cost records are preserved and are not recalculated. Unpriced requests retain usage but do not contribute to monetary totals.

## Validation

Full plugin Go tests passed with cgo on Linux amd64. The CPA cancellation regression test and CPA build passed. Integration with a real local CPA and mock upstream verified 404 for public stats, 401 for absent/invalid management keys, authenticated JSON with no-store caching, and identical static HTML with `format=json`.

A Sol request with 200,000 input tokens (100,000 cached) and 10,000 output tokens (including 5,000 reasoning tokens) recorded 210,000 tokens and $0.64. Dashboard and userscript JavaScript connected to the real authenticated endpoint using a minimal DOM harness. Full browser layout and actual Tampermonkey installation were not tested.
