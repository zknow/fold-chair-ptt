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
        loaded: false
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

    function renderMessage(p) {
        var li = document.createElement('li');
        li.className = 'msg ' + tagClass(p.tag);

        var tag = document.createElement('span');
        tag.className = 'tag';
        tag.textContent = p.tag || '';

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

        li.appendChild(tag);
        li.appendChild(user);
        li.appendChild(content);
        li.appendChild(time);
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

    function appendMessage(p, autoScroll) {
        state.messages.push(p);
        el('messages').appendChild(renderMessage(p));
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
            var data = await api().LoadArticle(url);
            resetMessages();

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
            el('main').hidden = false;
            el('changeBtn').hidden = false;
            state.loaded = true;
            scrollToBottom(false);
            updateBottomBtn();

            await api().StartMonitor();
            setStatus('polling', '監控中');
        } catch (e) {
            var message = e && e.message ? e.message : String(e);
            el('error').textContent = message;
            el('main').hidden = true;
            el('loader').hidden = false;
            state.loaded = false;
            setStatus('error', '載入失敗');
        } finally {
            el('loadBtn').disabled = false;
            el('loadBtn').textContent = '開始監控';
        }
    }

    function backToLoader() {
        api().StopMonitor();
        state.loaded = false;
        resetMessages();
        el('main').hidden = true;
        el('settings').hidden = true;
        el('loader').hidden = false;
        el('changeBtn').hidden = true;
        el('board').textContent = '';
        el('articleTitle').textContent = 'Fold Chair PTT';
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
                    appendMessage(fresh[j], false);
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
        el('changeBtn').addEventListener('click', backToLoader);
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
