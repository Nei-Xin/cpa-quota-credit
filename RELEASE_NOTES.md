# v1.0.10

This release restores billing corrections on top of v1.0.8 while retaining its password-free dashboard and public statistics endpoints. It does not include v1.0.9's management authentication changes.

- Correct OpenAI fallback prices and long-context input/output multipliers.
- Preserve usage but mark unknown models and missing Priority/Fast prices as unpriced instead of substituting unrelated rates.
- Support explicit `pricing.overrides`, with precedence over cached and remote rates, and read CPA's `config_yaml` lifecycle configuration.
- Label dashboard and userscript amounts as estimates and show unpriced-request notices.

Merge the Sol `pricing.overrides` example into your CPA plugin configuration: the default remote feed may still contain older rates. The example uses the official promotional prices checked on 2026-09-08 and must be reviewed when prices change. Historical costs are not recalculated. Amounts exclude extra tool fees and do not represent subscription allowance consumption.

The separate CPA host fix for canceled asynchronous usage delivery is not included in this plugin binary. Hosts with that behavior require a host update to prevent dropped usage records.
