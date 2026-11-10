(function () {
    'use strict';

    var state = {
        settings: {
            alwaysOnTop: true,
            theme: 'light',
            fontSize: 13,
            pollIntervalMs: 3000,
            maxMessages: 500
        },
        messages: [],
        buffer: [],
        lastSeq: -1,
        loaded: false,
        board: null,
        boardPage: null,
        source: 'url'
    };

    function el(id) {
        return document.getElementById(id);
    }

    function api() {
        return window.go.main.App;
    }

    function whenReady(cb) {
        if (window.go && window.go.main && window.go.main.App && window.runtime) {
            cb();
            return;
        }
        setTimeout(function () {
            whenReady(cb);
        }, 50);
    }

    // =================
    // settings
    // =================

    function applySettings() {
        document.body.className = 'theme-' + state.settings.theme;
        document.documentElement.style.setProperty('--font-size', state.settings.fontSize + 'px');
        var pinned = !!state.settings.alwaysOnTop;
        var pin = el('pinBtn');
        pin.classList.toggle('active', pinned);
        pin.setAttribute('aria-pressed', String(pinned));
        pin.title = pinned ? '已置頂（點擊取消）' : '未置頂（點擊置頂）';
    }

    function openSettings() {
        el('themeSelect').value = state.settings.theme;
        el('fontSize').value = state.settings.fontSize;
        el('pollInterval').value = state.settings.pollIntervalMs;
        el('maxMessages').value = state.settings.maxMessages;
        el('settings').hidden = false;
    }

    function closeSettings() {
        el('settings').hidden = true;
    }

    function toggleSettings() {
        if (el('settings').hidden) {
            openSettings();
        } else {
            closeSettings();
        }
    }

    async function saveSettings() {
        state.settings.theme = el('themeSelect').value;
        state.settings.fontSize = clampInt(el('fontSize').value, 10, 24, 13);
        state.settings.pollIntervalMs = clampInt(el('pollInterval').value, 1000, 30000, 3000);
        state.settings.maxMessages = clampInt(el('maxMessages').value, 50, 3000, 500);
        applySettings();
        trimMessages();
        try {
            await api().SaveSettings(state.settings);
        } catch (e) { /* ignore save errors */ }
        closeSettings();
    }

    function clampInt(value, min, max, fallback) {
        var n = parseInt(value, 10);
        if (isNaN(n)) return fallback;
        return Math.min(max, Math.max(min, n));
    }

    async function togglePin() {
        state.settings.alwaysOnTop = !state.settings.alwaysOnTop;
        applySettings();
        try {
            await api().SetAlwaysOnTop(state.settings.alwaysOnTop);
        } catch (e) { /* ignore */ }
    }

    // =================
    // status
    // =================

    function setStatus(kind, text) {
        var dot = el('statusDot');
        dot.className = 'dot ' + (kind || '');
        el('statusText').textContent = text;
    }

    // =================
    // messages
    // =================

    function resetMessages() {
        state.messages = [];
        state.buffer = [];
        state.lastSeq = -1;
        el('messages').innerHTML = '';
    }

    function tagClass(tag) {
        if (tag && tag.indexOf('噓') >= 0) return 'down';
        if (tag && tag.indexOf('推') >= 0) return 'up';
        return 'neutral';
    }

    function renderMessage(p, animate) {
        var li = document.createElement('li');
        li.className = 'msg ' + tagClass(p.tag) + (animate ? ' enter' : '');

        var meta = document.createElement('div');
        meta.className = 'meta';

        var dot = document.createElement('span');
        dot.className = 'dot';

        var user = document.createElement('span');
        user.className = 'user';
        user.textContent = p.user || '';

        var content = document.createElement('span');
        content.className = 'content';
        if (p.link) {
            var a = document.createElement('a');
            a.href = '#';
            a.textContent = p.content || p.link;
            a.addEventListener('click', function (event) {
                event.preventDefault();
                api().OpenExternal(p.link);
            });
            content.appendChild(a);
        } else {
            content.textContent = p.content || '';
        }

        var time = document.createElement('span');
        time.className = 'time';
        time.textContent = ((p.date || '') + ' ' + (p.time || '')).trim();

        meta.appendChild(dot);
        meta.appendChild(user);
        meta.appendChild(time);
        li.appendChild(meta);
        li.appendChild(content);
        return li;
    }

    function trimMessages() {
        var list = el('messages');
        while (list.children.length > state.settings.maxMessages) {
            list.removeChild(list.firstChild);
        }
        while (state.messages.length > state.settings.maxMessages) {
            state.messages.shift();
        }
    }

    function appendMessage(p, autoScroll, animate) {
        state.messages.push(p);
        el('messages').appendChild(renderMessage(p, animate));
        trimMessages();
        if (autoScroll) {
            scrollToBottom(false);
        }
    }

    function nearBottom() {
        var list = el('messages');
        return list.scrollHeight - list.scrollTop - list.clientHeight < 48;
    }

    function scrollToBottom(smooth) {
        var list = el('messages');
        list.scrollTo({ top: list.scrollHeight, behavior: smooth ? 'smooth' : 'auto' });
    }

    function updateBottomBtn() {
        var btn = el('toBottom');
        btn.hidden = state.buffer.length === 0;
        el('bufferCount').textContent = state.buffer.length ? '(' + state.buffer.length + ')' : '';
    }

    function flushBuffer() {
        if (state.buffer.length === 0) return;
        var items = state.buffer;
        state.buffer = [];
        for (var i = 0; i < items.length; i++) {
            appendMessage(items[i], false);
        }
        scrollToBottom(false);
        updateBottomBtn();
    }

    // =================
    // load article
    // =================

    var BOARD_NAMES = { Stock: '股票板', Gossiping: '八卦板' };

    function errorText(e) {
        return e && e.message ? e.message : String(e);
    }

    function showLoader() {
        el('loader').hidden = false;
        el('boardView').hidden = true;
        el('main').hidden = true;
        el('changeBtn').hidden = true;
        el('board').textContent = '';
        el('articleTitle').textContent = 'Fold Chair PTT';
    }

    function showBoardView() {
        el('loader').hidden = true;
        el('boardView').hidden = false;
        el('main').hidden = true;
        el('changeBtn').hidden = true;
        el('board').textContent = BOARD_NAMES[state.board] || '';
        el('articleTitle').textContent = '文章列表';
    }

    async function openArticle(url, source) {
        var data = await api().LoadArticle(url);
        resetMessages();
        state.source = source;

        el('board').textContent = data.board || '';
        el('articleTitle').textContent = data.title || 'Fold Chair PTT';
        document.title = data.title || 'Fold Chair PTT';

        var pushes = data.pushes || [];
        for (var i = 0; i < pushes.length; i++) {
            appendMessage(pushes[i], false);
            if (typeof pushes[i].seq === 'number') {
                state.lastSeq = Math.max(state.lastSeq, pushes[i].seq);
            }
        }

        el('loader').hidden = true;
        el('boardView').hidden = true;
        el('main').hidden = false;
        el('changeBtn').hidden = false;
        el('changeBtn').textContent = source === 'board' ? '返回列表' : '更換文章';
        state.loaded = true;
        scrollToBottom(false);
        updateBottomBtn();

        await api().StartMonitor();
        setStatus('polling', '監控中');
    }

    async function loadArticle() {
        var url = el('urlInput').value.trim();
        if (!url) {
            el('error').textContent = '請輸入 PTT 文章網址';
            return;
        }
        el('error').textContent = '';
        el('loadBtn').disabled = true;
        el('loadBtn').textContent = '載入中...';

        try {
            await openArticle(url, 'url');
        } catch (e) {
            el('error').textContent = errorText(e);
            state.loaded = false;
            setStatus('error', '載入失敗');
        } finally {
            el('loadBtn').disabled = false;
            el('loadBtn').textContent = '開始監控';
        }
    }

    async function openBoard(board) {
        state.board = board;
        el('error').textContent = '';
        try {
            renderBoard(await api().ListBoard(board, ''));
            showBoardView();
        } catch (e) {
            el('error').textContent = errorText(e);
            setStatus('error', '載入失敗');
        }
    }

    async function loadBoardPage(url) {
        el('boardError').textContent = '';
        try {
            renderBoard(await api().ListBoard(state.board, url || ''));
        } catch (e) {
            el('boardError').textContent = errorText(e);
        }
    }

    async function openFromBoard(url) {
        el('boardError').textContent = '';
        try {
            await openArticle(url, 'board');
        } catch (e) {
            el('boardError').textContent = errorText(e);
            setStatus('error', '載入失敗');
        }
    }

    function scoreInfo(score) {
        if (score >= 100) return { text: '爆', cls: 'score-hot' };
        if (score >= 30) return { text: String(score), cls: 'score-high' };
        if (score > 0) return { text: String(score), cls: 'score-low' };
        if (score <= -100) return { text: 'XX', cls: 'score-neg' };
        if (score < 0) return { text: 'X' + Math.floor(-score / 10), cls: 'score-neg' };
        return { text: '', cls: '' };
    }

    function renderBoard(page) {
        state.boardPage = page;
        el('boardName').textContent = BOARD_NAMES[page.board] || page.board;
        el('pageInfo').textContent = /index\d+\.html$/.test(page.url) ? '歷史文章' : '最新文章';
        el('prevPageBtn').disabled = !page.prevUrl;
        el('nextPageBtn').disabled = !page.nextUrl;

        var ul = el('boardItems');
        ul.innerHTML = '';
        ul.scrollTop = 0;
        page.entries.forEach(function (entry) {
            var li = document.createElement('li');
            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'board-item';

            var info = scoreInfo(entry.score);
            var score = document.createElement('span');
            score.className = 'board-item-score ' + info.cls;
            score.textContent = info.text;

            var body = document.createElement('span');
            body.className = 'board-item-body';

            var title = document.createElement('span');
            title.className = 'board-item-title';
            title.textContent = entry.title;

            var author = document.createElement('span');
            author.className = 'board-item-author';
            author.textContent = entry.author;

            body.appendChild(title);
            body.appendChild(author);

            var date = document.createElement('span');
            date.className = 'board-item-date';
            date.textContent = entry.date;

            btn.appendChild(score);
            btn.appendChild(body);
            btn.appendChild(date);
            btn.addEventListener('click', function () {
                openFromBoard(entry.url);
            });
            li.appendChild(btn);
            ul.appendChild(li);
        });
    }

    function backFromMonitor() {
        api().StopMonitor();
        state.loaded = false;
        resetMessages();
        el('settings').hidden = true;
        if (state.source === 'board' && state.boardPage) {
            showBoardView();
        } else {
            showLoader();
        }
        updateBottomBtn();
        setStatus('', '待機');
    }

    // =================
    // realtime
    // =================

    function wireEvents() {
        window.runtime.EventsOn('pushes', function (pushes) {
            if (!pushes || !pushes.length) return;
            var fresh = [];
            for (var i = 0; i < pushes.length; i++) {
                if (typeof pushes[i].seq !== 'number' || pushes[i].seq > state.lastSeq) {
                    fresh.push(pushes[i]);
                    if (typeof pushes[i].seq === 'number') {
                        state.lastSeq = Math.max(state.lastSeq, pushes[i].seq);
                    }
                }
            }
            if (!fresh.length) return;

            if (nearBottom()) {
                for (var j = 0; j < fresh.length; j++) {
                    appendMessage(fresh[j], false, true);
                }
                scrollToBottom(false);
            } else {
                for (var k = 0; k < fresh.length; k++) {
                    state.buffer.push(fresh[k]);
                }
                updateBottomBtn();
            }
            setStatus('polling', '監控中');
        });
    }

    // =================
    // init
    // =================

    function bindUi() {
        el('loadBtn').addEventListener('click', loadArticle);
        el('urlInput').addEventListener('keydown', function (event) {
            if (event.key === 'Enter') loadArticle();
        });
        el('pinBtn').addEventListener('click', togglePin);
        el('settingBtn').addEventListener('click', toggleSettings);
        el('cancelSettingsBtn').addEventListener('click', closeSettings);
        el('saveSettingsBtn').addEventListener('click', saveSettings);
        el('closeBtn').addEventListener('click', function () {
            window.runtime.Quit();
        });
        el('changeBtn').addEventListener('click', backFromMonitor);
        document.querySelectorAll('.board-btn').forEach(function (btn) {
            btn.addEventListener('click', function () {
                openBoard(btn.getAttribute('data-board'));
            });
        });
        el('boardBackBtn').addEventListener('click', showLoader);
        el('boardRefreshBtn').addEventListener('click', function () {
            loadBoardPage(state.boardPage ? state.boardPage.url : '');
        });
        el('prevPageBtn').addEventListener('click', function () {
            loadBoardPage(state.boardPage.prevUrl);
        });
        el('nextPageBtn').addEventListener('click', function () {
            loadBoardPage(state.boardPage.nextUrl);
        });
        el('toBottom').addEventListener('click', function () {
            flushBuffer();
            scrollToBottom(true);
        });
        el('messages').addEventListener('scroll', function () {
            if (nearBottom() && state.buffer.length > 0) {
                flushBuffer();
            }
        });
    }

    whenReady(async function () {
        bindUi();
        wireEvents();
        el('urlInput').value = '';
        try {
            var settings = await api().GetSettings();
            if (settings) {
                state.settings = Object.assign(state.settings, settings);
            }
        } catch (e) { /* use defaults */ }
        applySettings();
        setStatus('', '待機');
    });
})();
