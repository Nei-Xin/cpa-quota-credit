# CLIProxyAPI 额度与计费统计插件 (cpa-quota-credit)

为 **CLIProxyAPI (CPA)** 打造的标准 C-ABI 动态链接库插件。提供高精度的 Token 消耗计量、上游成本与下游用户计费分离（A $ / U $）、动态配额时间窗口统计、嵌入式 Web 控制面板以及配套的油猴管理面板助手。

---

## 📸 界面预览

### 1. 独立 Web 控制看板
提供全局徽章统计、调用方（API Key）脱敏消耗榜、上游渠道账号成本榜、模型分布与实时核算流水明细：

![独立 Web 控制看板](assets/dashboard_preview.png)

### 2. 油猴管理面板助手
在 CPA 官方管理后台（`management.html#/quota`）账号卡片上实时注入当前重置周期内的 4 枚胶囊徽章（到达周期自动归零）：

![油猴管理面板助手](assets/tampermonkey_preview.png)

---

## 🌟 核心特性

- **高精度 Token 与费用核算体系**：
  - **细粒度 Token 计费**：覆盖 Input、Output、Reasoning/Thinking、Prompt Cache Read 与 Prompt Cache Creation（支持 5m / 1h 缓存区分）。
  - **长上下文阶梯倍率（Long Context Multiplier）**：超长上下文自动匹配梯度单价。
  - **服务等级系数（Service Tier）**：支持 `priority`、`fast` 等等级倍率换算。
  - **8 位精度货币量化**：采用 `NUMERIC(20,8)` Half-Away-From-Zero 算法，避免浮点数精度截断误差。
- **双重成本核算模型（A $ / U $）**：
  - **`A $` (Actual / Admin Cost)**：上游凭证账号实际支出金额（$Base \times Multiplier_{account}$）。
  - **`U $` (User Cost)**：下游客户端 / API Key 扣除金额（$Base \times Multiplier_{user}$）。
- **动态配额时间窗口（Window Stats）**：
  - 支持 **`7天配额周期`**、**`今日 (0点起)`** 与 **`全部历史`** 3 档时间窗口滑动统计。
  - 账号卡片数据与上游官方配额周期自动同步，周期重置时徽章用量自动归零。
- **LiteLLM 动态价格同步**：
  - 自动从远程模型价格库定时同步，支持 SHA256 增量校验与本地离线 Fallback 兜底。
  - 智能兼容 Claude 有序家族匹配、OpenAI 别名降级、Gemini 3.6 Thinking 规则。
- **嵌入式持久化存储**：
  - 基于纯 Go 的轻量级 `bbolt` 嵌入式数据库，免去外部数据库搭建与维护成本。
- **调用方密钥安全脱敏**：
  - 看板全链路对 API Key 自动进行掩码遮蔽（如 `fX4A****n29@`），杜绝明文泄露。

---

## 📦 免编译直接使用 (Releases)

无需本地安装 Go 或 C 语言编译器，在 [GitHub Releases](https://github.com/Nei-Xin/cpa-quota-credit/releases) 页面直接下载对应系统的预编译压缩包：

- **Linux x86_64**：`cpa-quota-credit-linux-amd64.tar.gz` (解压出 `.so`)
- **Linux ARM64**：`cpa-quota-credit-linux-arm64.tar.gz` (解压出 `.so`)
- **Windows x64**：`cpa-quota-credit-windows-amd64.zip` (解压出 `.dll`)
- **macOS Apple Silicon**：`cpa-quota-credit-darwin-arm64.tar.gz` (解压出 `.dylib`)

---

## ⚙️ 接入与配置

### 1. 放置插件文件
将下载解压出的 `.so` 或 `.dll` 放置在 CLIProxyAPI 目录下的 `plugins` 文件夹中：
```text
CLIProxyAPI/
├── cli-proxy-api
├── config.yaml
└── plugins/
    └── cpa-quota-credit.so
```

### 2. 配置 `config.yaml`
在 CLIProxyAPI 的 `config.yaml` 中添加以下配置项：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpa-quota-credit:
      enabled: true
      priority: 10
      
      # 数据库保存路径（建议挂载持久化目录）
      db_path: "./data/quota_credit.db"
      
      # 动态价格源配置
      pricing:
        remote_url: "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.json"
        hash_url: "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.sha256"
        update_interval_hours: 24
        
      # 倍率配置 (对应 A $ 实际成本 与 U $ 用户计费)
      multipliers:
        default_user_multiplier: 1.0     # 默认用户扣费倍率 (U $)
        default_account_multiplier: 1.0  # 默认上游成本倍率 (A $)
        
        # 针对特定下游 API Key 单独设置扣费倍率
        key_multipliers:
          "sk-internal-dev": 1.0
          "sk-external-user": 1.5
          "sk-vip-partner": 0.8
          
        # 针对特定上游凭证文件 Auth ID 单独设置成本倍率
        account_multipliers:
          "claude-oauth-discount.json": 0.8
          "codex-prod-team.json": 1.0
```

---

## 🖥️ 访问与使用

### 1. 独立 Web 控制看板
直接在浏览器中打开：
```text
http://<你的CPA服务器地址>:<端口>/v0/resource/plugins/cpa-quota-credit/dashboard
```

### 2. 油猴脚本（浏览器卡片助手）
1. 在浏览器安装 **Tampermonkey** 扩展。
2. 添加新建脚本，将仓库中的 [`cpa-quota-credit.user.js`](https://github.com/Nei-Xin/cpa-quota-credit/blob/main/cpa-quota-credit.user.js) 粘贴并保存。
3. 打开 CPA 官方管理后台的配额页面（`management.html#/quota`），即可在每个账号卡片上实时查看本周期的请求数、Token 消耗与 A/U 费用徽章！

---

## 🛠️ 本地从源码编译

如果需要自行从源码构建：

```bash
# Linux
go build -buildmode=c-shared -ldflags="-s -w" -o cpa-quota-credit.so main.go

# Windows
go build -buildmode=c-shared -ldflags="-s -w" -o cpa-quota-credit.dll main.go

# macOS
go build -buildmode=c-shared -ldflags="-s -w" -o cpa-quota-credit.dylib main.go
```

---

## 🙏 致谢 (Acknowledgements)

本项目在设计与开发过程中参考并受益于以下开源项目，在此表示诚挚的感谢：

- **[CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)** - 优秀的 AI API 路由代理与管理网关。
- **[sub2api](https://github.com/Wei-Shaw/sub2api)** - 优秀的 AI 订阅分发与计费管理系统。

---

## 📄 开源许可证

本项目基于 [MIT License](LICENSE) 开源。
