// ==UserScript==
// @name         CPA Quota & Credit 额度与费用展示助手
// @namespace    https://github.com/router-for-me/cpa-quota-credit
// @version      1.0.5
// @description  基于 sub2api 窗口算法，在 CLIProxyAPI 账号卡片上呈现当前周期用量与 A/U 费用 (轻量、无死循环、高响应)
// @author       router-for-me
// @match        *://*/management.html*
// @match        https://cpa.zzii.de/management.html*
// @grant        none
// @run-at       document-idle
// ==/UserScript==

(function () {
    'use strict';

    console.log('[CPA Quota Credit] Userscript v1.0.5 starting...');

    // 1. 样式注入
    const style = document.createElement('style');
    style.id = 'cpa-quota-credit-style';
    style.textContent = `
        .cpa-pill {
            display: inline-flex !important;
            align-items: center !important;
            padding: 2px 7px !important;
            background: #f1f5f9 !important;
            border: 1px solid #cbd5e1 !important;
            border-radius: 6px !important;
            font-size: 11.5px !important;
            font-weight: 600 !important;
            color: #334155 !important;
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace !important;
            line-height: 1.4 !important;
            box-shadow: 0 1px 2px rgba(0,0,0,0.04) !important;
            white-space: nowrap !important;
            margin-right: 5px !important;
            margin-bottom: 3px !important;
            cursor: default !important;
            transition: all 0.2s ease !important;
            user-select: none !important;
        }

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

        .cpa-card-badge-container {
            display: flex !important;
            align-items: center !important;
            flex-wrap: wrap !important;
            margin: 6px 0 8px 0 !important;
            padding: 2px 0 !important;
        }
    `;
    if (!document.getElementById('cpa-quota-credit-style')) {
        document.head.appendChild(style);
    }

    let cachedStats = null;
    let isFetching = false;
    let isRendering = false;

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

    async function fetchStats() {
        if (isFetching) return;
        isFetching = true;
        
        const endpoints = [
            '/v0/resource/plugins/cpa-quota-credit/stats',
            '/v0/resource/plugins/cpa-quota-credit/dashboard?format=json'
        ];

        for (let url of endpoints) {
            try {
                const res = await fetch(url);
                if (res.ok) {
                    cachedStats = await res.json();
                    renderBadges();
                    break;
                }
            } catch (e) {}
        }
        isFetching = false;
    }

    function buildPillBadgesHTML(wStats, totalStats) {
        const req = wStats ? wStats.total_requests : 0;
        const tok = wStats ? wStats.total_tokens : 0;
        const actualCost = wStats ? wStats.actual_cost : 0;
        const userCost = wStats ? wStats.user_cost : 0;

        const allReq = totalStats ? totalStats.total_requests : req;
        const allTok = totalStats ? totalStats.total_tokens : tok;
        const allA = totalStats ? totalStats.actual_cost : actualCost;
        const allU = totalStats ? totalStats.user_cost : userCost;

        const tip = `【当前周期】请求: ${req} | Token: ${tok} | A成本: ${formatUSD(actualCost)} | U计费: ${formatUSD(userCost)}\n【全部历史】请求: ${allReq} | Token: ${allTok} | A成本: ${formatUSD(allA)} | U计费: ${formatUSD(allU)}`;

        return `
            <div class="cpa-pill" title="${tip}"><span class="pill-req">${formatNumber(req)} req</span></div>
            <div class="cpa-pill" title="${tip}"><span class="pill-tok">${formatNumber(tok)}</span></div>
            <div class="cpa-pill" title="${tip}"><span class="pill-prefix-a">A</span> ${formatUSD(actualCost)}</div>
            <div class="cpa-pill" title="${tip}"><span class="pill-prefix-u">U</span> ${formatUSD(userCost)}</div>
        `;
    }

    function findAuthStat(accountText) {
        if (!cachedStats || !cachedStats.auths || !accountText) return null;
        const target = accountText.trim().toLowerCase().replace(/\.\.\.$/, '');

        let matched = cachedStats.auths.find(a => a.auth_id && a.auth_id.toLowerCase() === target);
        if (matched) return matched;

        matched = cachedStats.auths.find(a => {
            const aid = (a.auth_id || '').toLowerCase();
            return aid.includes(target) || target.includes(aid.replace(/\.json$/, ''));
        });
        return matched;
    }

    // 渲染卡片徽章 (防死循环保护)
    function renderBadges() {
        if (!cachedStats || isRendering) return;
        isRendering = true;

        try {
            const allButtons = Array.from(document.querySelectorAll('button, div, span, a'));
            const refreshBtns = allButtons.filter(el => el.textContent && el.textContent.trim() === '刷新额度');

            refreshBtns.forEach(btn => {
                let card = btn.parentElement;
                for (let i = 0; i < 6; i++) {
                    if (!card || card === document.body) break;
                    const text = card.textContent || '';
                    if (text.includes('codex-') || text.includes('claude-') || text.includes('gemini-') || text.includes('@') || text.includes('套餐')) {
                        break;
                    }
                    card = card.parentElement;
                }

                if (!card) return;

                const cardText = card.textContent || '';
                const match = cardText.match(/([a-zA-Z0-9_\-\.]+@[a-zA-Z0-9_\-\.]+|codex-[a-zA-Z0-9_\-\.]+|claude-[a-zA-Z0-9_\-\.]+|gemini-[a-zA-Z0-9_\-\.]+)/);
                const accountName = match ? match[1] : '';

                const stat = findAuthStat(accountName);
                let windowStat = stat ? (stat.seven_day || stat.today || stat) : null;
                let totalStat = stat ? { total_requests: stat.total_requests, total_tokens: stat.total_tokens, actual_cost: stat.actual_cost, user_cost: stat.user_cost } : null;

                const targetHTML = buildPillBadgesHTML(windowStat, totalStat);

                let existingContainer = card.querySelector('.cpa-card-badge-container');
                if (existingContainer) {
                    if (existingContainer.dataset.renderedHash !== accountName + (windowStat ? windowStat.total_requests : 0)) {
                        existingContainer.innerHTML = targetHTML;
                        existingContainer.dataset.renderedHash = accountName + (windowStat ? windowStat.total_requests : 0);
                    }
                    return;
                }

                const badgeBox = document.createElement('div');
                badgeBox.className = 'cpa-card-badge-container';
                badgeBox.dataset.renderedHash = accountName + (windowStat ? windowStat.total_requests : 0);
                badgeBox.innerHTML = targetHTML;

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
        } finally {
            isRendering = false;
        }
    }

    // 2. 使用防抖监听 DOM 变动 (Debounce 600ms，彻底防止死循环与卡顿)
    let debounceTimer = null;
    const observer = new MutationObserver((mutations) => {
        // 过滤掉自身 badge 产生的变动
        const isSelfMutation = mutations.every(m => {
            return m.target && (m.target.classList?.contains('cpa-card-badge-container') || m.target.classList?.contains('cpa-pill'));
        });
        if (isSelfMutation) return;

        if (debounceTimer) clearTimeout(debounceTimer);
        debounceTimer = setTimeout(() => {
            if (!cachedStats) {
                fetchStats();
            } else {
                renderBadges();
            }
        }, 600);
    });

    observer.observe(document.body, { childList: true, subtree: true });

    // 初始执行 & 定时 10 秒平滑刷新
    fetchStats();
    setInterval(fetchStats, 10000);
})();
