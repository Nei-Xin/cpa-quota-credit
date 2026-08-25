// ==UserScript==
// @name         CPA Quota & Credit 额度与费用展示助手
// @namespace    https://github.com/router-for-me/cpa-quota-credit
// @version      1.0.7
// @description  在 CLIProxyAPI 账号卡片上分别呈现 5 小时与 7 天周期用量及 A/U 费用徽章
// @author       router-for-me
// @match        *://*/management.html*
// @match        https://cpa.zzii.de/management.html*
// @grant        none
// @run-at       document-idle
// ==/UserScript==

(function () {
    'use strict';

    console.log('[CPA Quota Credit] Userscript v1.0.7 starting...');

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
            flex-direction: column !important;
            align-items: flex-start !important;
            gap: 2px !important;
            margin: 6px 0 8px 0 !important;
            padding: 2px 0 !important;
        }

        .cpa-period-row {
            display: flex !important;
            align-items: center !important;
            flex-wrap: wrap !important;
        }

        .cpa-period-label {
            width: 24px !important;
            flex: 0 0 24px !important;
            color: #64748b !important;
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace !important;
            font-size: 10px !important;
            font-weight: 700 !important;
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

    function quotaTipLine(label, quota) {
        if (!quota) return `【官方 ${label}】暂无数据`;
        const reset = quota.reset_at ? new Date(quota.reset_at).toLocaleString() : '未知';
        return `【官方 ${label}】已用: ${Number(quota.used_percent || 0).toFixed(1)}% | 重置: ${reset}`;
    }

    function periodTipLine(label, stats) {
        const s = stats || {};
        return `【本地 ${label}】请求: ${s.total_requests || 0} | Token: ${s.total_tokens || 0} | A成本: ${formatUSD(s.actual_cost)} | U计费: ${formatUSD(s.user_cost)}`;
    }

    function buildPeriodRowHTML(label, stats, tip) {
        const s = stats || {};
        return `
            <div class="cpa-period-row" title="${tip}">
                <span class="cpa-period-label">${label}</span>
                <div class="cpa-pill"><span class="pill-req">${formatNumber(s.total_requests || 0)} req</span></div>
                <div class="cpa-pill"><span class="pill-tok">${formatNumber(s.total_tokens || 0)}</span></div>
                <div class="cpa-pill"><span class="pill-prefix-a">A</span> ${formatUSD(s.actual_cost)}</div>
                <div class="cpa-pill"><span class="pill-prefix-u">U</span> ${formatUSD(s.user_cost)}</div>
            </div>
        `;
    }

    function buildPillBadgesHTML(fiveHourStats, sevenDayStats, totalStats, quota) {
        const total = totalStats || {};
        const tip = `${periodTipLine('5 小时周期', fiveHourStats)}\n${periodTipLine('7 天周期', sevenDayStats)}\n${quotaTipLine('5 小时', quota && quota.five_hour)}\n${quotaTipLine('7 天', quota && quota.seven_day)}\n【历史】请求: ${total.total_requests || 0} | Token: ${total.total_tokens || 0} | A成本: ${formatUSD(total.actual_cost)} | U计费: ${formatUSD(total.user_cost)}`;

        return buildPeriodRowHTML('5h', fiveHourStats, tip) +
            buildPeriodRowHTML('7d', sevenDayStats, tip);
    }

    function findAuthStat(accountText) {
        if (!cachedStats || !cachedStats.auths || !accountText) return null;
        const target = accountText.trim().toLowerCase().replace(/\.\.\.$/, '');

        // 提取卡片中的邮箱主体 (如 9350205@gmail.com)
        const emailMatch = target.match(/([a-zA-Z0-9_\-\.]+@[a-zA-Z0-9_\-\.]+)/);
        const emailKey = emailMatch ? emailMatch[1] : '';

        // 匹配所有属于该账号的记录 (支持多次重登产生的历史记录)
        const matches = cachedStats.auths.filter(a => {
            const aid = (a.auth_id || '').toLowerCase();
            if (emailKey && aid.includes(emailKey)) return true;
            return aid.includes(target) || target.includes(aid.replace(/\.json$/, ''));
        });

        if (matches.length === 0) return null;
        if (matches.length === 1) return matches[0];

        // 自动将同一邮箱多次认证的历史数据完美合并归一
        const merged = {
            auth_id: matches[0].auth_id,
            total_requests: 0,
            total_tokens: 0,
            actual_cost: 0,
            user_cost: 0,
            five_hour: { total_requests: 0, total_tokens: 0, actual_cost: 0, user_cost: 0 },
            seven_day: { total_requests: 0, total_tokens: 0, actual_cost: 0, user_cost: 0 },
            quota: { five_hour: null, seven_day: null }
        };

        matches.forEach(m => {
            merged.total_requests += (m.total_requests || 0);
            merged.total_tokens += (m.total_tokens || 0);
            merged.actual_cost += (m.actual_cost || 0);
            merged.user_cost += (m.user_cost || 0);

            if (m.five_hour) {
                merged.five_hour.total_requests += (m.five_hour.total_requests || 0);
                merged.five_hour.total_tokens += (m.five_hour.total_tokens || 0);
                merged.five_hour.actual_cost += (m.five_hour.actual_cost || 0);
                merged.five_hour.user_cost += (m.five_hour.user_cost || 0);
            }
            if (m.seven_day) {
                merged.seven_day.total_requests += (m.seven_day.total_requests || 0);
                merged.seven_day.total_tokens += (m.seven_day.total_tokens || 0);
                merged.seven_day.actual_cost += (m.seven_day.actual_cost || 0);
                merged.seven_day.user_cost += (m.seven_day.user_cost || 0);
            }
            ['five_hour', 'seven_day'].forEach(windowName => {
                const candidate = m.quota && m.quota[windowName];
                const current = merged.quota[windowName];
                if (candidate && (!current || new Date(candidate.updated_at) > new Date(current.updated_at))) {
                    merged.quota[windowName] = candidate;
                }
            });
        });
        return merged;
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
                let fiveHourStat = stat ? stat.five_hour : null;
                let sevenDayStat = stat ? stat.seven_day : null;
                let totalStat = stat ? { total_requests: stat.total_requests, total_tokens: stat.total_tokens, actual_cost: stat.actual_cost, user_cost: stat.user_cost } : null;

                const targetHTML = buildPillBadgesHTML(fiveHourStat, sevenDayStat, totalStat, stat && stat.quota);
                const renderedHash = accountName + JSON.stringify({ fiveHourStat, sevenDayStat, quota: stat && stat.quota });

                let existingContainer = card.querySelector('.cpa-card-badge-container');
                if (existingContainer) {
                    if (existingContainer.dataset.renderedHash !== renderedHash) {
                        existingContainer.innerHTML = targetHTML;
                        existingContainer.dataset.renderedHash = renderedHash;
                    }
                    return;
                }

                const badgeBox = document.createElement('div');
                badgeBox.className = 'cpa-card-badge-container';
                badgeBox.dataset.renderedHash = renderedHash;
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
