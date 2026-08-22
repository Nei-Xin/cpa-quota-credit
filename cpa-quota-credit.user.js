// ==UserScript==
// @name         CPA Quota & Credit 额度与费用展示助手
// @namespace    https://github.com/router-for-me/cpa-quota-credit
// @version      1.0.3
// @description  在 CLIProxyAPI 管理面板 (management.html#/quota) 账号卡片上实时呈现 sub2api 风格的请求数、Token 消耗与 A/U 费用徽章 (req / tokens / A $ / U $)
// @author       router-for-me
// @match        *://*/management.html*
// @match        https://cpa.zzii.de/management.html*
// @grant        none
// @run-at       document-idle
// ==/UserScript==

(function () {
    'use strict';

    console.log('[CPA Quota Credit] Userscript v1.0.3 starting...');

    // 注入 sub2api 风格 Pill 胶囊徽章样式
    const style = document.createElement('style');
    style.textContent = `
        /* 胶囊徽章通用样式 */
        .cpa-pill {
            display: inline-flex !important;
            align-items: center !important;
            padding: 3px 8px !important;
            background: #f1f5f9 !important;
            border: 1px solid #cbd5e1 !important;
            border-radius: 6px !important;
            font-size: 12px !important;
            font-weight: 600 !important;
            color: #334155 !important;
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace !important;
            line-height: 1.4 !important;
            box-shadow: 0 1px 2px rgba(0,0,0,0.05) !important;
            white-space: nowrap !important;
            margin-right: 6px !important;
            margin-bottom: 4px !important;
            transition: all 0.2s ease !important;
        }

        /* 暗黑主题适配 */
        @media (prefers-color-scheme: dark), (prefers-dark), [data-theme='dark'], html.dark {
            .cpa-pill {
                background: rgba(255, 255, 255, 0.08) !important;
                border-color: rgba(255, 255, 255, 0.15) !important;
                color: #cbd5e1 !important;
            }
        }

        .cpa-pill:hover {
            transform: translateY(-1px);
            box-shadow: 0 2px 4px rgba(0,0,0,0.12) !important;
        }

        .cpa-pill .pill-prefix-a { color: #a855f7 !important; font-weight: 700 !important; margin-right: 3px !important; }
        .cpa-pill .pill-prefix-u { color: #f59e0b !important; font-weight: 700 !important; margin-right: 3px !important; }
        .cpa-pill .pill-req { color: #0284c7 !important; }
        .cpa-pill .pill-tok { color: #10b981 !important; }

        /* 卡片内徽章行容器 */
        .cpa-card-badge-container {
            display: flex !important;
            align-items: center !important;
            flex-wrap: wrap !important;
            margin: 8px 0 10px 0 !important;
            padding: 4px 0 !important;
        }
    `;
    document.head.appendChild(style);

    let cachedStats = null;
    let isFetching = false;

    // 数值格式化
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

    // 从公开 Resource 路径拉取统计数据 (无需 management key 鉴权，永不触发 403)
    async function fetchStats() {
        if (isFetching) return;
        isFetching = true;
        
        const endpoints = [
            '/v0/resource/plugins/cpa-quota-credit/stats',
            '/v0/resource/plugins/cpa-quota-credit/dashboard?format=json',
            window.location.origin + '/v0/resource/plugins/cpa-quota-credit/stats'
        ];

        for (let url of endpoints) {
            try {
                const res = await fetch(url);
                if (res.ok) {
                    cachedStats = await res.json();
                    console.log('[CPA Quota Credit] Stats fetched successfully:', cachedStats);
                    renderBadges();
                    break;
                }
            } catch (e) {
                // Try next endpoint
            }
        }
        isFetching = false;
    }

    // 构建徽章 HTML
    function buildPillBadgesHTML(reqCount, totalTokens, actualCost, userCost) {
        return `
            <div class="cpa-pill" title="该账号累计处理请求数"><span class="pill-req">${formatNumber(reqCount)} req</span></div>
            <div class="cpa-pill" title="该账号累计 Token 消耗"><span class="pill-tok">${formatNumber(totalTokens)}</span></div>
            <div class="cpa-pill" title="上游真实成本支出 (Actual Cost)"><span class="pill-prefix-a">A</span> ${formatUSD(actualCost)}</div>
            <div class="cpa-pill" title="用户计费额度 (User Cost)"><span class="pill-prefix-u">U</span> ${formatUSD(userCost)}</div>
        `;
    }

    // 模糊匹配账号
    function findAuthStat(accountText) {
        if (!cachedStats || !cachedStats.auths || !accountText) return null;
        const target = accountText.trim().toLowerCase().replace(/\.\.\.$/, '');

        // 1. 精确匹配
        let matched = cachedStats.auths.find(a => a.auth_id && a.auth_id.toLowerCase() === target);
        if (matched) return matched;

        // 2. 包含匹配
        matched = cachedStats.auths.find(a => {
            const aid = (a.auth_id || '').toLowerCase();
            return aid.includes(target) || target.includes(aid.replace(/\.json$/, ''));
        });
        return matched;
    }

    // 核心渲染逻辑：寻找卡片并注入徽章
    function renderBadges() {
        if (!cachedStats) return;

        // 策略 1: 通过 "刷新额度" 按钮寻找父级卡片
        const allButtons = Array.from(document.querySelectorAll('button, div, span, a'));
        const refreshBtns = allButtons.filter(el => el.textContent && el.textContent.trim() === '刷新额度');

        refreshBtns.forEach(btn => {
            // 向上寻找卡片容器
            let card = btn.parentElement;
            for (let i = 0; i < 6; i++) {
                if (!card || card === document.body) break;
                // 判断是否是卡片容器（包含账号名称或套餐等特征文字）
                const text = card.textContent || '';
                if (text.includes('codex-') || text.includes('claude-') || text.includes('gemini-') || text.includes('@') || text.includes('套餐')) {
                    break;
                }
                card = card.parentElement;
            }

            if (!card || card.querySelector('.cpa-card-badge-container')) return;

            // 提取账号名称
            const cardText = card.textContent || '';
            const match = cardText.match(/([a-zA-Z0-9_\-\.]+@[a-zA-Z0-9_\-\.]+|codex-[a-zA-Z0-9_\-\.]+|claude-[a-zA-Z0-9_\-\.]+|gemini-[a-zA-Z0-9_\-\.]+)/);
            const accountName = match ? match[1] : '';

            const stat = findAuthStat(accountName);
            const req = stat ? stat.total_requests : 0;
            const tok = stat ? stat.total_tokens : 0;
            const actualCost = stat ? stat.actual_cost : 0;
            const userCost = stat ? stat.user_cost : 0;

            const badgeBox = document.createElement('div');
            badgeBox.className = 'cpa-card-badge-container';
            badgeBox.innerHTML = buildPillBadgesHTML(req, tok, actualCost, userCost);

            // 插入到卡片中最佳位置（在"套餐"信息或限额进度条之前）
            const rows = Array.from(card.children);
            let inserted = false;
            for (let row of rows) {
                if (row.textContent && (row.textContent.includes('套餐') || row.textContent.includes('限额') || row.textContent.includes('重置'))) {
                    card.insertBefore(badgeBox, row);
                    inserted = true;
                    break;
                }
            }
            if (!inserted) {
                card.insertBefore(badgeBox, card.firstChild);
            }
        });
    }

    // 监听 DOM 变动 (处理 Vue / SPA 异步加载)
    const observer = new MutationObserver(() => {
        if (!cachedStats) {
            fetchStats();
        } else {
            renderBadges();
        }
    });

    observer.observe(document.body, { childList: true, subtree: true });

    // 初始执行 & 定时 8 秒增量同步
    fetchStats();
    setInterval(fetchStats, 8000);

})();
