// ==UserScript==
// @name         CPA Quota & Credit 额度与费用展示助手
// @namespace    https://github.com/router-for-me/cpa-quota-credit
// @version      1.0.1
// @description  在 CLIProxyAPI 管理面板 (management.html#/quota) 账号卡片上实时呈现 sub2api 风格的请求数、Token 消耗与 A/U 费用徽章 (req / tokens / A $ / U $)
// @author       router-for-me
// @match        *://*/management.html*
// @match        https://cpa.zzii.de/management.html*
// @grant        none
// @run-at       document-idle
// ==/UserScript==

(function () {
    'use strict';

    console.log('[CPA Quota Credit] Userscript loaded v1.0.1');

    // 样式注入 (sub2api 风格 Pill 胶囊徽章)
    const style = document.createElement('style');
    style.textContent = `
        /* 顶部全局徽章栏 */
        .cpa-global-badge-bar {
            display: flex;
            align-items: center;
            gap: 8px;
            margin: 12px 0 16px 0;
            padding: 8px 14px;
            background: rgba(30, 41, 59, 0.7);
            backdrop-filter: blur(8px);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 10px;
            width: fit-content;
            box-shadow: 0 2px 6px rgba(0,0,0,0.15);
        }
        .cpa-global-title {
            font-size: 13px;
            font-weight: 600;
            color: #94a3b8;
            margin-right: 4px;
        }

        /* 胶囊徽章通用样式 */
        .cpa-pill {
            display: inline-flex;
            align-items: center;
            padding: 2px 8px;
            background: rgba(241, 245, 249, 0.9);
            border: 1px solid rgba(203, 213, 225, 0.8);
            border-radius: 6px;
            font-size: 11.5px;
            font-weight: 600;
            color: #334155;
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
            line-height: 1.4;
            box-shadow: 0 1px 2px rgba(0,0,0,0.04);
            white-space: nowrap;
            transition: all 0.2s ease;
        }
        @media (prefers-color-scheme: dark), (prefers-dark) {
            .cpa-pill {
                background: rgba(255, 255, 255, 0.08);
                border-color: rgba(255, 255, 255, 0.15);
                color: #cbd5e1;
            }
        }

        .cpa-pill:hover {
            transform: translateY(-1px);
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }

        .cpa-pill .pill-prefix-a { color: #a855f7; font-weight: 700; margin-right: 3px; }
        .cpa-pill .pill-prefix-u { color: #f59e0b; font-weight: 700; margin-right: 3px; }
        .cpa-pill .pill-req { color: #0284c7; }
        .cpa-pill .pill-tok { color: #10b981; }

        /* 卡片内徽章容器 */
        .cpa-card-badge-container {
            display: flex;
            align-items: center;
            gap: 6px;
            flex-wrap: wrap;
            margin: 6px 0 8px 0;
            padding: 4px 0;
        }
    `;
    document.head.appendChild(style);

    let cachedStats = null;
    let isFetching = false;
    let authFailed = false;

    // 工具函数：数值格式化
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

    // 智能探测 CPA 管理密钥
    let detectedToken = '';
    
    // 监听全局 fetch 抓取管理面板已登录携带的 Token
    const originalFetch = window.fetch;
    window.fetch = async function (...args) {
        try {
            const [url, config] = args;
            if (config && config.headers) {
                let token = '';
                if (typeof config.headers.get === 'function') {
                    token = config.headers.get('Authorization') || config.headers.get('X-Management-Key');
                } else if (config.headers['Authorization']) {
                    token = config.headers['Authorization'];
                } else if (config.headers['X-Management-Key']) {
                    token = config.headers['X-Management-Key'];
                }
                if (token && token.startsWith('Bearer ')) {
                    detectedToken = token.replace('Bearer ', '').trim();
                    authFailed = false;
                } else if (token) {
                    detectedToken = token.trim();
                    authFailed = false;
                }
            }
        } catch (e) {}
        return originalFetch.apply(this, args);
    };

    function getManagementKey() {
        if (detectedToken) return detectedToken;
        const candidates = [
            'management_secret_key',
            'secret_key',
            'cpa_secret_key',
            'management_key',
            'auth_key',
            'token',
            'cpa_token'
        ];
        for (const k of candidates) {
            const v = localStorage.getItem(k) || sessionStorage.getItem(k);
            if (v && v.trim()) return v.trim();
        }
        return '';
    }

    // 从 CPA 后端拉取统计数据 (带熔断机制，绝不重复触发 403 封禁)
    async function fetchStats() {
        if (isFetching || authFailed) return;
        isFetching = true;
        try {
            const headers = {};
            const key = getManagementKey();
            if (key) {
                headers['Authorization'] = 'Bearer ' + key;
                headers['X-Management-Key'] = key;
            }

            const res = await fetch('/v0/management/plugins/cpa-quota-credit/stats', {
                method: 'GET',
                headers: headers
            });

            if (res.ok) {
                cachedStats = await res.json();
                authFailed = false;
                renderAllBadges();
            } else if (res.status === 403 || res.status === 401) {
                // 遇到 403/401 立即熔断暂停，等待下次用户登录/输入 Key 抓取成功后再请求，绝不刷屏造成封禁
                authFailed = true;
                console.warn('[CPA Quota Credit] 管理接口需要鉴权，将在捕获管理密钥后自动重试');
            }
        } catch (e) {
            console.error('[CPA Quota Credit] Fetch stats error:', e);
        } finally {
            isFetching = false;
        }
    }

    // 构建 4 枚 Pill 徽章 HTML
    function buildPillBadgesHTML(reqCount, totalTokens, actualCost, userCost) {
        return `
            <div class="cpa-pill" title="总请求次数"><span class="pill-req">${formatNumber(reqCount)} req</span></div>
            <div class="cpa-pill" title="总 Token 消耗"><span class="pill-tok">${formatNumber(totalTokens)}</span></div>
            <div class="cpa-pill" title="上游真实成本 (Actual Cost)"><span class="pill-prefix-a">A</span> ${formatUSD(actualCost)}</div>
            <div class="cpa-pill" title="用户计费额度 (User Cost)"><span class="pill-prefix-u">U</span> ${formatUSD(userCost)}</div>
        `;
    }

    // 匹配上游账号数据
    function findAuthStat(accountName) {
        if (!cachedStats || !cachedStats.auths || !accountName) return null;
        const target = accountName.trim().toLowerCase();

        let matched = cachedStats.auths.find(a => a.auth_id && a.auth_id.toLowerCase() === target);
        if (matched) return matched;

        const cleanName = target.replace(/\.\.\.$/, '').trim();
        matched = cachedStats.auths.find(a => {
            const aid = (a.auth_id || '').toLowerCase();
            return aid.includes(cleanName) || cleanName.includes(aid);
        });
        return matched;
    }

    // 渲染卡片内徽章与全局徽章
    function renderAllBadges() {
        if (!cachedStats) return;

        // 1. 渲染全局总额度条
        const quotaContainer = document.querySelector('.quota-container, .quota-view, .content-container, main, #app');
        if (quotaContainer && !document.getElementById('cpa-global-quota-bar')) {
            const summary = cachedStats.summary || { total_requests: 0, total_tokens: 0, actual_cost: 0, user_cost: 0 };
            const bar = document.createElement('div');
            bar.id = 'cpa-global-quota-bar';
            bar.className = 'cpa-global-badge-bar';
            bar.innerHTML = `
                <span class="cpa-global-title">📊 总额度消耗:</span>
                ${buildPillBadgesHTML(summary.total_requests, summary.total_tokens, summary.actual_cost, summary.user_cost)}
            `;
            quotaContainer.insertBefore(bar, quotaContainer.firstChild);
        }

        // 2. 遍历注入每个卡片
        const cards = document.querySelectorAll('.el-card, .n-card, .quota-card, [class*="card"]');
        cards.forEach(card => {
            if (card.querySelector('.cpa-card-badge-container')) return;
            const textNodes = Array.from(card.querySelectorAll('*')).map(el => el.textContent.trim());
            let accountName = '';
            for (const text of textNodes) {
                if (text.includes('codex-') || text.includes('claude-') || text.includes('gemini-') || text.includes('@') || text.includes('.json')) {
                    accountName = text.split('\n')[0].trim();
                    break;
                }
            }
            if (!accountName) return;

            const stat = findAuthStat(accountName);
            const req = stat ? stat.total_requests : 0;
            const tok = stat ? stat.total_tokens : 0;
            const actualCost = stat ? stat.actual_cost : 0;
            const userCost = stat ? stat.user_cost : 0;

            const container = document.createElement('div');
            container.className = 'cpa-card-badge-container';
            container.innerHTML = buildPillBadgesHTML(req, tok, actualCost, userCost);

            const progressBar = card.querySelector('.el-progress, [class*="progress"], [class*="limit"]');
            if (progressBar && progressBar.parentNode) {
                progressBar.parentNode.insertBefore(container, progressBar);
            } else {
                card.appendChild(container);
            }
        });
    }

    // 监听 SPA 路由变动与 DOM 渲染
    let timer = null;
    const observer = new MutationObserver(() => {
        if (timer) clearTimeout(timer);
        timer = setTimeout(() => {
            if (window.location.href.includes('quota') || window.location.href.includes('management')) {
                if (!cachedStats) fetchStats();
                else renderAllBadges();
            }
        }, 300);
    });
    observer.observe(document.body, { childList: true, subtree: true });

    fetchStats();
    setInterval(fetchStats, 15000);
})();
