// ==UserScript==
// @name         CPA Quota & Credit 额度与费用展示助手
// @namespace    https://github.com/router-for-me/cpa-quota-credit
// @version      1.0.0
// @description  在 CLIProxyAPI 管理面板 (management.html#/quota) 账号卡片上实时呈现 sub2api 风格的请求数、Token 消耗与 A/U 费用徽章 (req / tokens / A $ / U $)
// @author       router-for-me
// @match        *://*/management.html*
// @match        https://cpa.zzii.de/management.html*
// @grant        none
// @run-at       document-idle
// ==/UserScript==

(function () {
    'use strict';

    console.log('[CPA Quota Credit] Userscript loaded');

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
        /* 暗黑主题适配 */
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

        .cpa-pill .pill-prefix-a {
            color: #a855f7;
            font-weight: 700;
            margin-right: 3px;
        }
        .cpa-pill .pill-prefix-u {
            color: #f59e0b;
            font-weight: 700;
            margin-right: 3px;
        }
        .cpa-pill .pill-req {
            color: #0284c7;
        }
        .cpa-pill .pill-tok {
            color: #10b981;
        }

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

    // 获取管理 Secret Key
    function getManagementKey() {
        const candidates = [
            'management_secret_key',
            'secret_key',
            'cpa_secret_key',
            'management_key',
            'auth_key',
            'token'
        ];
        for (const k of candidates) {
            const v = localStorage.getItem(k) || sessionStorage.getItem(k);
            if (v && v.trim()) return v.trim();
        }
        return '';
    }

    // 从 CPA 后端拉取统计数据
    async function fetchStats() {
        if (isFetching) return;
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
                renderAllBadges();
            } else if (res.status === 404) {
                console.warn('[CPA Quota Credit] 插件尚未在 config.yaml 中启用或未加载');
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

        // 1. 精确匹配
        let matched = cachedStats.auths.find(a => a.auth_id && a.auth_id.toLowerCase() === target);
        if (matched) return matched;

        // 2. 前缀/模糊包含匹配 (解决页面中带 ... 省略号的情况)
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

        // 1. 渲染顶部全局统计条 (在 quota 容器顶部)
        const quotaContainer = document.querySelector('.quota-container, .quota-view, .content-container, main, #app');
        if (quotaContainer && !document.getElementById('cpa-global-quota-bar')) {
            const summary = cachedStats.summary || { total_requests: 0, total_tokens: 0, actual_cost: 0, user_cost: 0 };
            const bar = document.createElement('div');
            bar.id = 'cpa-global-quota-bar';
            bar.className = 'cpa-global-badge-bar';
            bar.innerHTML = `
                <span class="cpa-global-title">📊 总消耗看板:</span>
                ${buildPillBadgesHTML(summary.total_requests, summary.total_tokens, summary.actual_cost, summary.user_cost)}
            `;
            quotaContainer.insertBefore(bar, quotaContainer.firstChild);
        }

        // 2. 查找页面中所有的账号卡片
        // 兼容 el-card, iview card, custom card 等 DOM 结构
        const cards = document.querySelectorAll('.el-card, .n-card, .quota-card, [class*="card"]');

        cards.forEach(card => {
            // 防止重复插入
            if (card.querySelector('.cpa-card-badge-container')) return;

            // 查找卡片标题（包含 codex-, claude-, 等账号前缀或邮箱名）
            const headerElem = card.querySelector('.el-card__header, .n-card-header, h3, h4, .title, strong, [class*="header"], [class*="title"]') || card;
            const textNodes = Array.from(card.querySelectorAll('*')).map(el => el.textContent.trim());

            // 尝试匹配卡片中的账号文本
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

            // 找到最佳插入位置：插入在标题下方或周限额进度条上方
            const progressBar = card.querySelector('.el-progress, [class*="progress"], [class*="limit"]');
            if (progressBar && progressBar.parentNode) {
                progressBar.parentNode.insertBefore(container, progressBar);
            } else if (headerElem && headerElem.nextSibling) {
                headerElem.parentNode.insertBefore(container, headerElem.nextSibling);
            } else {
                card.appendChild(container);
            }
        });
    }

    // 页面加载及 DOM 变动监听 (SPA 路由/异步加载自适应)
    let observerTimeout = null;
    const observer = new MutationObserver(() => {
        if (observerTimeout) clearTimeout(observerTimeout);
        observerTimeout = setTimeout(() => {
            if (window.location.hash.includes('quota') || window.location.pathname.includes('quota') || window.location.href.includes('management')) {
                if (!cachedStats) {
                    fetchStats();
                } else {
                    renderAllBadges();
                }
            }
        }, 300);
    });

    observer.observe(document.body, { childList: true, subtree: true });

    // 初始执行 & 定时 10 秒增量刷新
    fetchStats();
    setInterval(fetchStats, 10000);

})();
