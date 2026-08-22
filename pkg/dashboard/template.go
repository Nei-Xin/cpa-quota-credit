package dashboard

// HTMLDashboardTemplate provides the modern dashboard UI with sub2api badge style
const HTMLDashboardTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CLIProxyAPI 额度与计费控制中心 (CPA Quota & Credit)</title>
    <style>
        :root {
            --bg-color: #0f172a;
            --card-bg: #1e293b;
            --card-border: #334155;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --accent-blue: #38bdf8;
            --accent-emerald: #34d399;
            --accent-purple: #c084fc;
            --accent-amber: #fbbf24;
            --badge-bg: rgba(255, 255, 255, 0.07);
            --badge-border: rgba(255, 255, 255, 0.12);
        }
        * { box-sizing: border-box; margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; }
        body { background-color: var(--bg-color); color: var(--text-primary); padding: 24px; }
        .container { max-width: 1320px; margin: 0 auto; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; padding-bottom: 16px; border-bottom: 1px solid var(--card-border); }
        .title-group h1 { font-size: 22px; font-weight: 600; color: var(--text-primary); }
        .title-group p { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }
        
        /* sub2api Style Pill Badges */
        .badge-bar { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
        .sub2api-pill {
            display: inline-flex;
            align-items: center;
            padding: 6px 14px;
            background: var(--badge-bg);
            border: 1px solid var(--badge-border);
            border-radius: 8px;
            font-size: 13px;
            font-weight: 600;
            color: #cbd5e1;
            box-shadow: 0 1px 3px rgba(0,0,0,0.2);
            transition: all 0.2s;
        }
        .sub2api-pill:hover { background: rgba(255, 255, 255, 0.12); border-color: rgba(255,255,255,0.25); }
        .sub2api-pill .highlight-req { color: var(--accent-blue); margin-left: 2px; }
        .sub2api-pill .highlight-tok { color: var(--accent-emerald); margin-left: 2px; }
        .sub2api-pill .highlight-a { color: var(--accent-purple); margin-right: 4px; }
        .sub2api-pill .highlight-u { color: var(--accent-amber); margin-right: 4px; }
        
        .refresh-btn {
            background: #3b82f6;
            color: white;
            border: none;
            padding: 6px 14px;
            border-radius: 6px;
            font-size: 13px;
            cursor: pointer;
            transition: 0.2s;
        }
        .refresh-btn:hover { background: #2563eb; }

        /* Grid */
        .grid-3 { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16px; margin-bottom: 24px; }
        @media (max-width: 1100px) { .grid-3 { grid-template-columns: 1fr; } }
        
        .card {
            background-color: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 12px;
            padding: 18px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.3);
        }
        .card h2 { font-size: 15px; font-weight: 600; margin-bottom: 14px; color: var(--text-primary); border-left: 4px solid var(--accent-blue); padding-left: 8px; }
        
        table { width: 100%; border-collapse: collapse; font-size: 12px; text-align: left; }
        th { color: var(--text-secondary); font-weight: 500; padding: 8px 10px; border-bottom: 1px solid var(--card-border); }
        td { padding: 10px; border-bottom: 1px solid rgba(255, 255, 255, 0.05); }
        tr:hover td { background-color: rgba(255, 255, 255, 0.02); }
        
        .badge-cell { display: inline-block; padding: 2px 6px; border-radius: 4px; font-size: 11px; background: rgba(56, 189, 248, 0.15); color: var(--accent-blue); max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
        .badge-auth { background: rgba(192, 132, 252, 0.15); color: var(--accent-purple); }
        .text-num { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
        .cost-u { color: var(--accent-amber); font-weight: 600; }
        .cost-a { color: var(--accent-purple); font-weight: 600; }
        .empty-hint { text-align: center; color: var(--text-secondary); padding: 20px; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="title-group">
                <h1>CLIProxyAPI 额度与计费控制中心</h1>
                <p>实时基于 sub2api 算法核算 Token 消耗与多维账单 (A: 上游真实成本 | U: 客户端计费额度)</p>
            </div>
            <div class="badge-bar">
                <!-- 4 Pills Aligned with sub2api UI -->
                <div class="sub2api-pill" id="pill-req">0 req</div>
                <div class="sub2api-pill" id="pill-tok">0</div>
                <div class="sub2api-pill" id="pill-a"><span class="highlight-a">A</span> $0.00</div>
                <div class="sub2api-pill" id="pill-u"><span class="highlight-u">U</span> $0.00</div>
                <button class="refresh-btn" onclick="fetchStats()">刷新</button>
            </div>
        </div>

        <div class="grid-3">
            <!-- 1. Client / User Keys -->
            <div class="card">
                <h2>调用者 (API Key) 消耗</h2>
                <table>
                    <thead>
                        <tr>
                            <th>API Key</th>
                            <th>请求</th>
                            <th>Token</th>
                            <th>扣费 (U $)</th>
                        </tr>
                    </thead>
                    <tbody id="key-table-body">
                        <tr><td colspan="4" class="empty-hint">暂无记录</td></tr>
                    </tbody>
                </table>
            </div>

            <!-- 2. Upstream Auths / Accounts -->
            <div class="card">
                <h2>上游账号 (Auth / 渠道) 成本</h2>
                <table>
                    <thead>
                        <tr>
                            <th>上游账号</th>
                            <th>请求</th>
                            <th>Token</th>
                            <th>成本 (A $)</th>
                        </tr>
                    </thead>
                    <tbody id="auth-table-body">
                        <tr><td colspan="4" class="empty-hint">暂无记录</td></tr>
                    </tbody>
                </table>
            </div>

            <!-- 3. Model Stats -->
            <div class="card">
                <h2>模型消耗分布</h2>
                <table>
                    <thead>
                        <tr>
                            <th>模型</th>
                            <th>请求</th>
                            <th>Token</th>
                            <th>扣费 (U $)</th>
                        </tr>
                    </thead>
                    <tbody id="model-table-body">
                        <tr><td colspan="4" class="empty-hint">暂无记录</td></tr>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Recent Logs -->
        <div class="card">
            <h2>最新请求核算明细</h2>
            <table>
                <thead>
                    <tr>
                        <th>时间</th>
                        <th>API Key</th>
                        <th>上游账号</th>
                        <th>模型</th>
                        <th>Prompt / Cache</th>
                        <th>Output / Reasoning</th>
                        <th>总 Token</th>
                        <th>A 成本</th>
                        <th>U 计费</th>
                    </tr>
                </thead>
                <tbody id="log-table-body">
                    <tr><td colspan="9" class="empty-hint">暂无请求记录</td></tr>
                </tbody>
            </table>
        </div>
    </div>

    <script>
        function formatNumber(num) {
            if (!num || isNaN(num)) return '0';
            if (num >= 1000000000) return (num / 1000000000).toFixed(1) + 'B';
            if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
            if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
            return num.toLocaleString();
        }

        function formatUSD(amount) {
            return '$' + Number(amount || 0).toFixed(2);
        }

        function escapeHTML(str) {
            return String(str || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
        }

        // 脱敏遮蔽 API Key (避免明文泄漏)
        function maskAPIKey(key) {
            if (!key) return 'anonymous';
            const k = String(key).trim();
            if (k.length <= 6) return '****';
            if (k.length <= 10) return k.substring(0, 2) + '****' + k.substring(k.length - 2);
            return k.substring(0, 4) + '****' + k.substring(k.length - 4);
        }

        async function fetchStats() {
            const endpoints = [
                '/v0/resource/plugins/cpa-quota-credit/stats',
                '/v0/resource/plugins/cpa-quota-credit/dashboard?format=json',
                '/v0/management/plugins/cpa-quota-credit/stats'
            ];
            
            for (let i = 0; i < endpoints.length; i++) {
                try {
                    const res = await fetch(endpoints[i]);
                    if (res.ok) {
                        const data = await res.json();
                        renderDashboard(data);
                        return;
                    }
                } catch (err) {
                    // Try next endpoint
                }
            }
            
            // If all failed, show default empty view
            renderDashboard({ summary: { total_requests: 0, total_tokens: 0, actual_cost: 0, user_cost: 0 }, keys: [], auths: [], models: [], recent_logs: [] });
        }

        function renderDashboard(data) {
            const summary = data.summary || { total_requests: 0, total_tokens: 0, actual_cost: 0, user_cost: 0 };
            
            // Render 4 Pill Badges
            document.getElementById('pill-req').innerHTML = formatNumber(summary.total_requests) + ' req';
            document.getElementById('pill-tok').innerHTML = formatNumber(summary.total_tokens);
            document.getElementById('pill-a').innerHTML = '<span class="highlight-a">A</span> ' + formatUSD(summary.actual_cost);
            document.getElementById('pill-u').innerHTML = '<span class="highlight-u">U</span> ' + formatUSD(summary.user_cost);

            // 1. Keys Table
            const keyBody = document.getElementById('key-table-body');
            if (!data.keys || data.keys.length === 0) {
                keyBody.innerHTML = '<tr><td colspan="4" class="empty-hint">暂无 API Key 记录 (尚未发起请求)</td></tr>';
            } else {
                var html = '';
                for (var i = 0; i < data.keys.length; i++) {
                    var k = data.keys[i];
                    var masked = maskAPIKey(k.api_key);
                    html += '<tr>' +
                        '<td><span class="badge-cell">' + escapeHTML(masked) + '</span></td>' +
                        '<td class="text-num">' + (k.total_requests || 0).toLocaleString() + '</td>' +
                        '<td class="text-num">' + formatNumber(k.total_tokens) + '</td>' +
                        '<td class="text-num cost-u">' + formatUSD(k.user_cost) + '</td>' +
                        '</tr>';
                }
                keyBody.innerHTML = html;
            }

            // 2. Upstream Auths Table
            const authBody = document.getElementById('auth-table-body');
            if (!data.auths || data.auths.length === 0) {
                authBody.innerHTML = '<tr><td colspan="4" class="empty-hint">暂无上游账号记录 (尚未发起请求)</td></tr>';
            } else {
                var aHtml = '';
                for (var a = 0; a < data.auths.length; a++) {
                    var au = data.auths[a];
                    aHtml += '<tr>' +
                        '<td><span class="badge-cell badge-auth" title="' + escapeHTML(au.auth_id) + '">' + escapeHTML(au.auth_id) + '</span></td>' +
                        '<td class="text-num">' + (au.total_requests || 0).toLocaleString() + '</td>' +
                        '<td class="text-num">' + formatNumber(au.total_tokens) + '</td>' +
                        '<td class="text-num cost-a">' + formatUSD(au.actual_cost) + '</td>' +
                        '</tr>';
                }
                authBody.innerHTML = aHtml;
            }

            // 3. Models Table
            const modelBody = document.getElementById('model-table-body');
            if (!data.models || data.models.length === 0) {
                modelBody.innerHTML = '<tr><td colspan="4" class="empty-hint">暂无模型记录 (尚未发起请求)</td></tr>';
            } else {
                var mHtml = '';
                for (var j = 0; j < data.models.length; j++) {
                    var m = data.models[j];
                    mHtml += '<tr>' +
                        '<td><strong>' + escapeHTML(m.model) + '</strong></td>' +
                        '<td class="text-num">' + (m.total_requests || 0).toLocaleString() + '</td>' +
                        '<td class="text-num">' + formatNumber(m.total_tokens) + '</td>' +
                        '<td class="text-num cost-u">' + formatUSD(m.user_cost) + '</td>' +
                        '</tr>';
                }
                modelBody.innerHTML = mHtml;
            }

            // 4. Recent Logs Table
            const logBody = document.getElementById('log-table-body');
            if (!data.recent_logs || data.recent_logs.length === 0) {
                logBody.innerHTML = '<tr><td colspan="9" class="empty-hint">暂无请求记录 (发送任意模型请求后实时显示)</td></tr>';
            } else {
                var lHtml = '';
                for (var x = 0; x < data.recent_logs.length; x++) {
                    var log = data.recent_logs[x];
                    var c = log.cost || {};
                    var timeStr = new Date(log.timestamp).toLocaleTimeString();
                    var maskedKey = maskAPIKey(log.api_key);
                    lHtml += '<tr>' +
                        '<td style="color: var(--text-secondary);">' + timeStr + '</td>' +
                        '<td><span class="badge-cell">' + escapeHTML(maskedKey) + '</span></td>' +
                        '<td><span class="badge-cell badge-auth">' + escapeHTML(log.auth_id || '-') + '</span></td>' +
                        '<td><strong>' + escapeHTML(log.model) + '</strong></td>' +
                        '<td class="text-num">' + (c.input_tokens || 0) + ' / <span style="color: var(--accent-blue)">' + (c.cache_read_tokens || 0) + '</span></td>' +
                        '<td class="text-num">' + (c.output_tokens || 0) + ' / <span style="color: var(--accent-emerald)">' + (c.reasoning_tokens || 0) + '</span></td>' +
                        '<td class="text-num"><strong>' + (c.total_tokens || 0) + '</strong></td>' +
                        '<td class="text-num cost-a">' + formatUSD(c.actual_cost) + '</td>' +
                        '<td class="text-num cost-u">' + formatUSD(c.user_cost) + '</td>' +
                        '</tr>';
                }
                logBody.innerHTML = lHtml;
            }
        }

        // Fetch on load & auto poll every 5s
        fetchStats();
        setInterval(fetchStats, 5000);
    </script>
</body>
</html>`
