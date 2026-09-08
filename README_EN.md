# CLIProxyAPI Quota & Credit Billing Plugin (cpa-quota-credit)

<p align="center">
  <a href="README.md"><b>中文</b></a> | <a href="README_EN.md"><b>English</b></a>
</p>

A high-precision Token measurement and billing calculation plugin built for **CLIProxyAPI (CPA)**. Supports dual cost tracking (Estimated Upstream API Cost `A $` vs User Billed Cost `U $`), built-in Web dashboard, and a Tampermonkey userscript to inject real-time usage badges onto CPA account cards.

---

## 📸 Screenshots

### 1. Standalone Web Dashboard

![Dashboard](assets/dashboard_preview.png)

### 2. CPA Account Card Badges (Tampermonkey)

![Cards](assets/tampermonkey_preview.png)

---

## ✨ Core Features

- 🎯 **High-Precision Billing**: Covers Input/Output, Reasoning tokens, Prompt Cache reads & creations, and tiered long-context pricing.
- 💰 **Dual Cost Accounting**: Independent tracking for estimated upstream API cost (**`A $`**) and downstream user cost (**`U $`**) with custom multipliers.
- ⏳ **Dynamic Windows**: Supports **7-day rolling cycle** (auto-resets to 0 upon period expiration), **Today (from 00:00)**, and **All Time** windows.
- 🛡️ **Privacy & Masking**: End-to-end automatic masking for client API Keys (`fX4A****n29@`) across all dashboard views.

---

## 🚀 3-Step Quick Start

Amounts estimate token costs from model rates: A = base cost × account multiplier; U = base cost × user multiplier. They exclude extra tool fees and do not represent an official invoice or ChatGPT subscription allowance. The 5h/7d percentages are upstream response-header snapshots.

OpenAI fallback rates were checked against official model documentation on 2026-09-08 (including GPT-5.6 Sol promotional pricing). Cached and remote rates still override fallback rates. Unknown models or missing Priority/Fast rates retain usage but are marked unpriced and excluded from monetary totals. Changes affect new requests only; historical costs are not recalculated.

### Step 1: Download the Plugin
Download the pre-built archive for your OS from [Releases](https://github.com/Nei-Xin/cpa-quota-credit/releases) (e.g., `cpa-quota-credit-linux-amd64.tar.gz` for Linux), extract `cpa-quota-credit.so`, and place it in the `plugins/` directory.

### Step 2: Add Configuration
Add the plugin configuration in CLIProxyAPI's `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpa-quota-credit:
      enabled: true
      priority: 10
      db_path: "./data/quota_credit.db"
```

### Step 3: Restart CPA & View Dashboard
- **Web Dashboard**: `http://<CPA_IP>:<PORT>/v0/resource/plugins/cpa-quota-credit/dashboard`
- **Authentication**: Enter the CPA management key (not an API client key) in the dashboard or the userscript's bottom-right form. The key stays in the current page only and must be entered again after reloading.
- **API boundary**: The public dashboard serves static HTML only. Stats are available exclusively at `GET /v0/management/plugins/cpa-quota-credit/stats` with `Authorization: Bearer <management-key>`, authenticated by CPA. The old public `/stats` and `?format=json` no longer serve stats. Cross-origin embedding is unsupported.
- **Tampermonkey Userscript**: [👉 Click here to install script](https://raw.githubusercontent.com/Nei-Xin/cpa-quota-credit/main/cpa-quota-credit.user.js) (view live badges on `management.html#/quota`).

---

## 🙏 Acknowledgements

- [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) - AI API proxy and routing gateway.
- [sub2api](https://github.com/Wei-Shaw/sub2api) - AI subscription and billing management system.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
