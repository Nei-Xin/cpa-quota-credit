// ==UserScript==
// @name         CPA Quota & Credit 额度与费用展示助手
// @namespace    https://github.com/router-for-me/cpa-quota-credit
// @version      1.0.4
// @description  基于 sub2api 窗口统计算法，在 CLIProxyAPI 账号卡片上实时呈现当前重置周期内的请求数、Token 消耗与 A/U 费用徽章 (周期重置时自动归零)
// @author       router-for-me
// @match        *://*/management.html*
// @match        https://cpa.zzii.de/management.html*
// @grant        none
// @run-at       document-idle
// ==/UserScript==

(function () {
    'use strict';

    console.log('[CPA Quota Credit] Userscript v1.0.4 (sub2api window mode) loaded');

    // 注入 sub2api 风格 Pill 胶囊徽章样式
    const style = document.createElement('style');
    style.textContent = `
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
            cursor: default !important;
            transition: all 0.2s ease !important;
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
            margin: 8px 0 10px 0 !important;
            padding: 4px 0 !important;
        }
    `;
    document.head.appendChild(style);

    let cachedStats = null;
    let isFetching = false;

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
            '/v0/resource/plugins/cpa-quota-credit/dashboard?format=json',
            window.location.origin + '/v0/resource/plugins/cpa-quota-credit/stats'
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

    // 构建 sub2api 胶囊徽章 (展示当前周期用量，tooltip 展示历史总计)
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

    function renderBadges() {
        if (!cachedStats) return;

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
            
            // sub2api 算法：根据卡片类型选择当前周期窗口（周限额选 seven_day，月度/其他默认七天/今日窗口）
            let windowStat = stat ? (stat.seven_day || stat.today || stat) : null;
            let totalStat = stat ? { total_requests: stat.total_requests, total_tokens: stat.total_tokens, actual_cost: stat.actual_cost, user_cost: stat.user_cost } : null;

            let existingContainer = card.querySelector('.cpa-card-badge-container');
            if (existingContainer) {
                existingContainer.innerHTML = buildPillBadgesHTML(windowStat, totalStat);
                return;
            }

            const badgeBox = document.createElement('div');
            badgeBox.className = 'cpa-card-badge-container';
            badgeBox.innerHTML = buildPillBadgesHTML(windowStat, totalStat);

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

    const observer = new MutationObserver(() => {
        if (!cachedStats) fetchStats();
        else renderBadges();
    });

    observer.observe(document.body, { childList: true, subtree: true });

    fetchStats();
    setInterval(fetchStats, 8000);
})();
