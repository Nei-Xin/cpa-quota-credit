# CLIProxyAPI 额度与计费统计插件 (cpa-quota-credit)

<p align="center">
  <a href="README.md"><b>中文</b></a> | <a href="README_EN.md"><b>English</b></a>
</p>

为 **CLIProxyAPI (CPA)** 打造的高精度 Token 计量与费用核算插件。支持上游估算费用与下游计费分离（A $ / U $）、内置 Web 控制看板，并提供配套油猴助手在账号卡片上实时呈现用量徽章。

---

## 📸 效果展示

### 1. 独立 Web 控制看板

![看板](assets/dashboard_preview.png)

### 2. CPA 账号卡片徽章（油猴助手）

![卡片](assets/tampermonkey_preview.png)

---

## ✨ 核心特性

- 🎯 **高精度计费**：覆盖 Input/Output、Reasoning 思考、Prompt Cache 缓存与长上下文阶梯计价。
- 💰 **双重账单**：上游 API 估算费用（**A $**）与下游用户计费（**U $**）独立核算，支持自定义倍率。
- ⏳ **动态周期**：支持 **7天周期**（到期自动归零）、**今日**、**全部历史** 3 档时间窗口。
- 🛡️ **安全脱敏**：前端与流水表全链路对 API Key 自动掩码脱敏（`fX4A****n29@`）。

---

## 🚀 3 步极速使用

金额是按模型价表计算的 Token 费用估算：A = 基础费用 × 账号倍率，U = 基础费用 × 用户倍率；不包含工具调用等额外费用，也不是 ChatGPT 订阅额度或官方账单。5h/7d 百分比来自上游响应头快照。

OpenAI 备用价格已按 2026-09-08 官方文档更新（GPT-5.6 Sol 使用官方促销价）；本地缓存和远程价表仍会覆盖备用价。未知模型或缺少对应 Priority/Fast 价格时保留用量、标记未计价，金额汇总不包含这些请求。更新只影响之后的新请求，不重算历史费用。

价格依据：[GPT-5.1-Codex](https://developers.openai.com/api/docs/models/gpt-5.1-codex)、[GPT-5.2-Codex](https://developers.openai.com/api/docs/models/gpt-5.2-codex)、[GPT-5.4](https://developers.openai.com/api/docs/models/gpt-5.4)、[GPT-5.6 Sol](https://developers.openai.com/api/docs/models/gpt-5.6-sol)。

### 步骤 1：下载插件
在 [Releases](https://github.com/Nei-Xin/cpa-quota-credit/releases) 页面下载对应系统的压缩包（如 Linux 下载 `cpa-quota-credit-linux-amd64.tar.gz`），解压得到 `cpa-quota-credit.so` 放入 `plugins/` 目录。

### 步骤 2：添加配置
在 CLIProxyAPI 的 `config.yaml` 中添加以下配置：

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

### 步骤 3：重启 CPA 并查看
- **独立看板地址**：`http://<你的CPA地址>:<端口>/v0/resource/plugins/cpa-quota-credit/dashboard`
- **统计鉴权**：看板及油猴助手均需在各自的输入框中输入 CPA 管理密钥（不是调用 API 使用的 Key）。密钥仅保留在当前页面，刷新页面后需重新输入。油猴输入框位于页面右下角。
- **接口边界**：公开看板只返回静态 HTML；统计仅通过 `GET /v0/management/plugins/cpa-quota-credit/stats` 提供，请求需携带 `Authorization: Bearer <管理密钥>`，由 CPA 统一鉴权。旧公开 `/stats` 和 `?format=json` 不再提供统计数据。跨域嵌入不受支持。
- **油猴助手**：[👉 点击一键直装脚本](https://raw.githubusercontent.com/Nei-Xin/cpa-quota-credit/main/cpa-quota-credit.user.js)（安装后打开 `management.html#/quota` 即可查看卡片徽章）。

---

## 🙏 致谢 (Acknowledgements)

- [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) - AI API 路由代理与管理网关
- [sub2api](https://github.com/Wei-Shaw/sub2api) - AI 订阅分发与计费管理系统

---

## 📄 开源许可证

本项目基于 [MIT License](LICENSE) 开源。
