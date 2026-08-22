# CLIProxyAPI 额度与计费统计插件 (CPA Quota & Credit)

基于 **sub2api** 费用核算体系与模型定价规则，为 **CLIProxyAPI (CPA)** 打造的标准 C-ABI 动态库插件。

## 功能特性

- **sub2api 计费体系深度对齐**：
  - **Token 细粒度计费**：Input、Output、Reasoning/Thinking、Prompt Cache Read、Prompt Cache Creation（区分 5m / 1h）。
  - **长上下文加价（Long Context Multiplier）**：超过阈值自动应用阶梯倍率。
  - **服务等级系数（Service Tier）**：支持 `priority`、`fast` 等等级倍率。
  - **8 位金额量化**：采用 `NUMERIC(20,8)` Half-Away-From-Zero 舍入算法，避免浮点累积误差。
- **双重成本（A $ / U $）**：
  - **`A $` (Actual / Admin Cost)**：上游渠道/账号真实消耗金额（$Base \times Multiplier_{account}$）。
  - **`U $` (User Cost)**：用户/API Key 计费扣除金额（$Base \times Multiplier_{user}$）。
- **LiteLLM 动态价格同步**：
  - 自动从 LiteLLM 远程价格库同步，支持 SHA256 增量检测与本地离线 Fallback 兜底。
  - 兼容 Claude 有序系列匹配、OpenAI 别名降级、Gemini 3.6 Thinking 规整。
- **嵌入式持久化存储**：
  - 基于纯 Go 的 `bbolt` 嵌入式数据库，免去外部数据库依赖。
- **现代化可视化 Web 仪表盘**：
  - 顶部直接呈现 sub2api 风格的 **4 枚 Pill 徽章**：`req` / `tokens` / `A $` / `U $`。
  - 支持 API Key 消耗排行、模型使用量统计、最新请求实时明细表。

---

## 编译指南

编译生成对应操作系统的动态链接库：

### Linux (生成 `.so`)
```bash
go build -buildmode=c-shared -o cpa-quota-credit.so main.go
```

### Windows (生成 `.dll`)
```powershell
go build -buildmode=c-shared -o cpa-quota-credit.dll main.go
```

### macOS (生成 `.dylib`)
```bash
go build -buildmode=c-shared -o cpa-quota-credit.dylib main.go
```

---

## 接入配置

将生成的动态库文件放入 CLIProxyAPI 的 `plugins` 目录下，并在 `config.yaml` 中配置：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpa-quota-credit:
      enabled: true
      priority: 10
      # 数据库保存路径
      db_path: "./data/quota_credit.db"
      # 价格源配置
      pricing:
        remote_url: "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.json"
        hash_url: "https://raw.githubusercontent.com/Wei-Shaw/model-price-repo/main/model_prices_and_context_window.sha256"
        update_interval_hours: 24
      # 倍率配置
      multipliers:
        default_user_multiplier: 1.0     # 默认用户倍率 (U $)
        default_account_multiplier: 1.0  # 默认上游成本倍率 (A $)
        key_multipliers:
          "sk-custom-vip": 1.2           # 指定 Key 自定义倍率
```

---

## 查看仪表盘与 API

- **Web 仪表盘（浏览器打开）**：
  ```text
  http://<CPA_HOST>:<PORT>/v0/resource/plugins/cpa-quota-credit/dashboard
  ```
- **Management JSON 接口**：
  ```bash
  curl http://<CPA_HOST>:<PORT>/v0/management/plugins/cpa-quota-credit/stats
  ```
