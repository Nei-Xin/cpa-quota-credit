# CLIProxyAPI Quota & Credit Billing Plugin (cpa-quota-credit)

<p align="center">
  <a href="README.md"><b>中文</b></a> | <a href="README_EN.md"><b>English</b></a>
</p>

A high-precision Token measurement and billing calculation plugin built for **CLIProxyAPI (CPA)**. Supports dual cost tracking (Actual Upstream Cost `A $` vs User Billed Cost `U $`), built-in Web dashboard, and a Tampermonkey userscript to inject real-time usage badges onto CPA account cards.

---

## 📸 Screenshots

| 1. Standalone Web Dashboard | 2. CPA Account Card Badges (Tampermonkey) |
| :---: | :---: |
| ![Dashboard](assets/dashboard_preview.png) | ![Cards](assets/tampermonkey_preview.png) |

---

## ✨ Core Features

- 🎯 **High-Precision Billing**: Covers Input/Output, Reasoning tokens, Prompt Cache reads & creations, and tiered long-context pricing.
- 💰 **Dual Cost Accounting**: Independent tracking for upstream actual cost (**`A $`**) and downstream user cost (**`U $`**) with custom multipliers.
- ⏳ **Dynamic Windows**: Supports **7-day rolling cycle** (auto-resets to 0 upon period expiration), **Today (from 00:00)**, and **All Time** windows.
- 🛡️ **Privacy & Masking**: End-to-end automatic masking for client API Keys (`fX4A****n29@`) across all dashboard views.

---

## 🚀 3-Step Quick Start

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
- **Tampermonkey Userscript**: [👉 Click here to install script](https://raw.githubusercontent.com/Nei-Xin/cpa-quota-credit/main/cpa-quota-credit.user.js) (view live badges on `management.html#/quota`).

---

## 🙏 Acknowledgements

- [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) - AI API proxy and routing gateway.
- [sub2api](https://github.com/Wei-Shaw/sub2api) - AI subscription and billing management system.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
