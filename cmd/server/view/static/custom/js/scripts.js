/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

function addLikeGroup(cid) {
    const btn = document.getElementById('addLikeGroup');
    const liked = btn.classList.contains('btn-primary');
    const xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/like');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function() {
        if (xhr.status >= 200 && xhr.status < 300) {
            if (liked) {
                btn.classList.remove('btn-primary');
                btn.classList.add('btn-secondary');
                removeLikeTag();
            } else {
                btn.classList.remove('btn-secondary');
                btn.classList.add('btn-primary');
                addLikeTag();
            }
        } else {
            console.error('like request failed:', xhr.status, xhr.responseText);
        }
    };
    xhr.onerror = function() {
        console.error('like request network error');
    };
    xhr.send('cid=' + encodeURIComponent(cid));
}

function findCustomsContainer() {
    const containers = document.querySelectorAll('.tag-container.field-name');
    for (const c of containers) {
        const text = (c.textContent || '').trim();
        if (text.startsWith('Customs')) {
            return c;
        }
    }
    return null;
}

function addLikeTag() {
    const container = findCustomsContainer();
    if (!container) return;
    container.classList.remove('hidden');
    const span = container.querySelector('span.tags');
    if (!span) return;
    if (span.querySelector('.tag-99999')) return;
    const a = document.createElement('a');
    a.href = '/tag/custom/like/';
    a.className = 'tag tag-99999';
    const name = document.createElement('span');
    name.className = 'name';
    name.textContent = 'like';
    const count = document.createElement('span');
    count.className = 'count';
    count.textContent = '1';
    a.appendChild(name);
    a.appendChild(count);
    span.appendChild(a);
}

function removeLikeTag() {
    const container = findCustomsContainer();
    if (!container) return;
    const span = container.querySelector('span.tags');
    if (!span) return;
    const a = span.querySelector('.tag-99999');
    if (a) {
        a.remove();
    }
    if (!span.querySelector('a.tag')) {
        container.classList.add('hidden');
    }
}

function toggleLikeTag(type, name, id) {
    const btn = document.getElementById('toggleLikeTag');
    if (!btn) return;
    const liked = btn.classList.contains('btn-primary');
    const xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/likeTag');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function() {
        if (xhr.status >= 200 && xhr.status < 300) {
            if (liked) {
                btn.classList.remove('btn-primary');
                btn.classList.add('btn-secondary');
                const link = document.getElementById('currentTagLink');
                if (link) link.classList.remove('tag-like');
            } else {
                btn.classList.remove('btn-secondary');
                btn.classList.add('btn-primary');
                const link = document.getElementById('currentTagLink');
                if (link) link.classList.add('tag-like');
            }
        } else {
            console.error('likeTag request failed:', xhr.status, xhr.responseText);
        }
    };
    xhr.onerror = function() {
        console.error('likeTag request network error');
    };
    var params = 'type=' + encodeURIComponent(type);
    if (id && id > 0) {
        params += '&id=' + encodeURIComponent(id);
    } else if (name) {
        params += '&name=' + encodeURIComponent(name);
    }
    xhr.send(params);
}

function showToast(message, opts) {
    var options = opts || {};
    var type = options.type || 'info';
    var duration = typeof options.duration === 'number' ? options.duration : 5000;
    var dismissible = options.dismissible !== false;
    if (typeof message === 'object' && message !== null) {
        try { message = JSON.stringify(message); } catch (e) {}
    }
    var icons = { success: '✅', error: '❌', info: 'ℹ️', warning: '⚠️' };
    var icon = icons[type] || '';
    var typeClass = 'alert-info';
    if (type === 'success') typeClass = 'alert-success';
    else if (type === 'error') typeClass = 'alert-danger';
    else if (type === 'warning') typeClass = 'alert-warning';
    var container = document.getElementById('messages');
    if (!container) return;
    var alert = document.createElement('div');
    alert.className = 'alert ' + typeClass + ' fade-slide-in open';
    alert.textContent = icon + ' ' + message;
    if (dismissible) {
        alert.style.cursor = 'pointer';
        alert.addEventListener('click', function() {
            if (alert && alert.parentNode) {
                alert.parentNode.removeChild(alert);
            }
        });
    }
    container.appendChild(alert);
    if (duration > 0) {
        setTimeout(function() {
            if (alert && alert.parentNode) {
                alert.parentNode.removeChild(alert);
            }
        }, duration);
    }
}

function formatError(resp) {
    var code = resp && resp.head ? resp.head.code : -1;
    var msg = resp && resp.head ? (resp.head.msg || resp.head.message || '') : '';
    return '[' + code + '] ' + (msg || '请求失败');
}

function highlightInvalidPages(indexes) {
    if (!Array.isArray(indexes) || indexes.length === 0) return;
    var container = document.getElementById('thumbnail-container');
    if (!container) return;
    indexes.forEach(function(it) {
        var idx = it.index || it;
        var link = container.querySelector('a.gallerythumb[href="/g/' + String(window._gallery && window._gallery.cid || '') + '/' + String(idx) + '/"]');
        if (link && link.parentElement) {
            link.parentElement.style.outline = '3px solid #e74c3c';
        }
    });
}

function ensureForceArchiveButton(cid) {
    var existing = document.getElementById('forceArchiveBtn');
    if (existing) return;
    var btns = document.querySelector('#info-block .buttons');
    if (!btns) return;
    var a = document.createElement('a');
    a.id = 'forceArchiveBtn';
    a.href = 'javascript:;';
    a.className = 'btn btn-secondary';
    a.innerHTML = '<i class="fa fa-exclamation-triangle"></i> 强制归档';
    a.onclick = function() { archiveComicForce(cid); };
    btns.appendChild(a);
}


function archiveComic(cid) {
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive');
    xhr.onload = function() {
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('已归档', { type: 'success' });
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
                if (resp.head.code === -1001) {
                    var invalids = (resp.body && resp.body.invalid_images) || [];
                    if (!invalids.length && window._gallery && window._gallery.images && Array.isArray(window._gallery.images.pages)) {
                        window._gallery.images.pages.forEach(function(p, i) {
                            if (p && p.status === false) invalids.push({ index: i + 1 });
                        });
                    }
                    highlightInvalidPages(invalids);
                    ensureForceArchiveButton(cid);
                    showToast('检测到异常图片，建议先“修复漫画状态”，或使用“强制归档”', { type: 'info' });
                }
            }
        } else {
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        showToast('网络错误', { type: 'error' });
    };
    xhr.send();
}

function archiveComicForce(cid) {
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive?force=true');
    xhr.onload = function() {
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('已强制归档', { type: 'success' });
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
            }
        } else {
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        showToast('网络错误', { type: 'error' });
    };
    xhr.send();
}

function restoreComic(cid) {
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/restore');
    xhr.onload = function() {
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('已恢复', { type: 'success' });
                setTimeout(function() {
                    try {
                        location.reload();
                    } catch (e) {}
                }, 300);
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
            }
        } else {
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        showToast('网络错误', { type: 'error' });
    };
    xhr.send();
}

function verifyComic(cid) {
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/verify');
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function() {
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('修复任务已启动', { type: 'success' });
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
            }
        } else {
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        showToast('网络错误', { type: 'error' });
    };
    var body = {
        id: String(cid),
        autoFix: true,
        maxWorkers: 1
    };
    xhr.send(JSON.stringify(body));
}

/**
 * 通用弹窗工具函数
 * 创建 .modal-wrapper > .modal-inner 结构的弹窗
 */
function showCustomModal(title, contentHtml, buttonsHtml) {
    var existing = document.querySelector('.modal-wrapper');
    if (existing) existing.remove();

    var wrapper = document.createElement('div');
    wrapper.className = 'modal-wrapper fade-slide-in open';

    var inner = document.createElement('div');
    inner.className = 'modal-inner' + (buttonsHtml ? '' : ' modal-compact');

    var titleEl = document.createElement('h1');
    titleEl.textContent = title;
    inner.appendChild(titleEl);

    var content = document.createElement('div');
    content.className = 'contents';
    if (typeof contentHtml === 'string') {
        content.innerHTML = contentHtml;
    } else if (contentHtml instanceof HTMLElement) {
        content.appendChild(contentHtml);
    }
    inner.appendChild(content);

    if (buttonsHtml) {
        var btns = document.createElement('div');
        btns.className = 'buttons';
        if (typeof buttonsHtml === 'string') {
            btns.innerHTML = buttonsHtml;
        } else if (buttonsHtml instanceof HTMLElement) {
            btns.appendChild(buttonsHtml);
        }
        inner.appendChild(btns);
    }

    wrapper.appendChild(inner);
    document.body.appendChild(wrapper);

    // 点击遮罩层关闭
    wrapper.addEventListener('click', function(e) {
        if (e.target === wrapper) closeModal(wrapper);
    });

    // Esc 键关闭
    var escHandler = function(e) {
        if (e.key === 'Escape') {
            closeModal(wrapper);
        }
    };
    wrapper._escHandler = escHandler;
    document.addEventListener('keydown', escHandler);

    return wrapper;
}

function closeModal(wrapper) {
    if (wrapper && wrapper.parentNode) {
        // 移除 Esc 监听器
        if (wrapper._escHandler) {
            document.removeEventListener('keydown', wrapper._escHandler);
        }
        wrapper.parentNode.removeChild(wrapper);
    }
}

/**
 * 漫画详情页 Tag 编辑器
 */
function openTagEditor(cid) {
    // 获取当前漫画信息
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/api/comic/getComicInfo');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function() {
        if (xhr.status !== 200) {
            showToast('获取漫画信息失败', { type: 'error' });
            return;
        }
        try {
            var resp = JSON.parse(xhr.responseText);
            var info = resp.body;
            var currentTags = info.tags || [];
            buildTagEditorModal(cid, currentTags);
        } catch (e) {
            showToast('解析响应失败', { type: 'error' });
        }
    };
    xhr.onerror = function() {
        showToast('网络错误', { type: 'error' });
    };
    xhr.send('cid=' + encodeURIComponent(cid));
}

function buildTagEditorModal(cid, currentTags) {
    var added = [];
    var removed = [];
    var tagTypes = [
        {value: 'parody', label: 'Parodies'},
        {value: 'character', label: 'Characters'},
        {value: 'tag', label: 'Tags'},
        {value: 'artist', label: 'Artists'},
        {value: 'group', label: 'Groups'},
        {value: 'language', label: 'Languages'},
        {value: 'category', label: 'Categories'},
        {value: 'custom', label: 'Customs'}
    ];

    // 去重键：匹配服务端 tagKey 逻辑（id > 0 用 type:id，否则用 type:name）
    function dedupKey(t) {
        return t.type + ':' + (t.id || t.name);
    }

    // 构建 tag 展示区域
    var tagsContainer = document.createElement('div');
    tagsContainer.style.cssText = 'margin-bottom: 10px; display: flex; flex-wrap: wrap; gap: 5px;';

    function renderTags() {
        tagsContainer.innerHTML = '';
        var displayTags = [];

        // 现有 tag（排除被移除的）
        currentTags.forEach(function(t) {
            var key = dedupKey(t);
            if (!removed.some(function(r) { return dedupKey(r) === key; })) {
                displayTags.push(t);
            }
        });

        // 新增的 tag
        added.forEach(function(t) {
            displayTags.push(t);
        });

        displayTags.forEach(function(t) {
            var chip = document.createElement('span');
            chip.className = 'tag tag-' + (t.id || 0);
            chip.style.cssText = 'display: inline-flex; align-items: center; margin: 2px; padding: 2px 8px; border-radius: 3px; background: #2a2a2a;';
            chip.innerHTML = '<span class="name" style="margin-right:4px">[' + t.type + '] ' + t.name + '</span>';

            var delBtn = document.createElement('a');
            delBtn.href = 'javascript:;';
            delBtn.textContent = 'x';
            delBtn.style.cssText = 'color: #e74c3c; text-decoration: none; font-weight: bold;';
            delBtn.onclick = function() {
                var key = dedupKey(t);
                var addedIdx = -1;
                for (var i = 0; i < added.length; i++) {
                    if (dedupKey(added[i]) === key) {
                        addedIdx = i;
                        break;
                    }
                }
                if (addedIdx >= 0) {
                    added.splice(addedIdx, 1);
                } else {
                    removed.push(t);
                }
                renderTags();
            };
            chip.appendChild(delBtn);
            tagsContainer.appendChild(chip);
        });
    }
    renderTags();

    // ===== 添加 tag 表单（双模式: Existing / New）=====
    var formContainer = document.createElement('div');

    // Tab 切换条
    var tabBar = document.createElement('div');
    tabBar.style.cssText = 'display: flex; gap: 0; margin-bottom: 10px; border-bottom: 2px solid #444;';
    formContainer.appendChild(tabBar);

    var activeMode = 'existing';
    var existingPanel = document.createElement('div');
    var newPanel = document.createElement('div');

    function switchMode(mode) {
        activeMode = mode;
        tabBar.querySelectorAll('.mode-tab').forEach(function(el) {
            el.style.background = el.getAttribute('data-mode') === mode ? '#444' : 'transparent';
            el.style.color = el.getAttribute('data-mode') === mode ? '#fff' : '#888';
        });
        existingPanel.style.display = mode === 'existing' ? 'flex' : 'none';
        newPanel.style.display = mode === 'new' ? 'flex' : 'none';
    }

    function createTab(label, mode, active) {
        var tab = document.createElement('a');
        tab.href = 'javascript:;';
        tab.className = 'mode-tab';
        tab.setAttribute('data-mode', mode);
        tab.textContent = label;
        tab.style.cssText = 'padding: 6px 16px; cursor: pointer; font-size: 13px; text-decoration: none;' +
            (active ? 'background:#444;color:#fff;' : 'background:transparent;color:#888;');
        tab.onclick = function() { switchMode(mode); };
        tabBar.appendChild(tab);
        return tab;
    }
    createTab('Existing', 'existing', true);
    createTab('New', 'new', false);

    // ----- Existing 模式 -----
    existingPanel.style.cssText = 'display: flex; gap: 5px; align-items: center; flex-wrap: wrap;';

    var exTypeSelect = document.createElement('select');
    exTypeSelect.style.cssText = 'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px;';
    tagTypes.forEach(function(tt) {
        var opt = document.createElement('option');
        opt.value = tt.value; opt.textContent = tt.label;
        exTypeSelect.appendChild(opt);
    });
    existingPanel.appendChild(exTypeSelect);

    var searchWrapper = document.createElement('div');
    searchWrapper.style.cssText = 'position: relative; flex: 1; min-width: 150px;';
    existingPanel.appendChild(searchWrapper);

    var searchInput = document.createElement('input');
    searchInput.type = 'text';
    searchInput.placeholder = 'Search existing tag...';
    searchInput.style.cssText = 'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px; width: 100%; box-sizing: border-box;';
    searchWrapper.appendChild(searchInput);

    var autocompleteDropdown = document.createElement('div');
    autocompleteDropdown.className = 'tag-autocomplete-dropdown';
    autocompleteDropdown.style.cssText = 'display: none; position: absolute; top: 100%; left: 0; right: 0; ' +
        'background: #333; border: 1px solid #555; border-radius: 3px; ' +
        'max-height: 200px; overflow-y: auto; z-index: 1000;';
    searchWrapper.appendChild(autocompleteDropdown);

    function hideAutocomplete() {
        autocompleteDropdown.style.display = 'none';
        autocompleteDropdown.innerHTML = '';
    }

    function renderAutocomplete(tags) {
        autocompleteDropdown.innerHTML = '';
        if (tags.length === 0) { autocompleteDropdown.style.display = 'none'; return; }
        autocompleteDropdown.style.display = 'block';

        tags.forEach(function(t) {
            var item = document.createElement('div');
            item.style.cssText = 'padding: 6px 10px; cursor: pointer; border-bottom: 1px solid #444; ' +
                'display: flex; justify-content: space-between;';
            item.innerHTML = '<span>[' + t.type + '] ' + t.name + '</span>' +
                '<span style="color:#888;font-size:12px;">' + t.count + '</span>';
            item.onclick = function() {
                // 检查是否已存在（用 dedupKey）
                var key = dedupKey(t);
                var exists = added.some(function(a) { return dedupKey(a) === key; }) ||
                    currentTags.some(function(ct) { return dedupKey(ct) === key && !removed.some(function(r) { return dedupKey(r) === dedupKey(ct); }); });
                if (exists) {
                    showToast('该 tag 已存在', { type: 'info' });
                } else {
                    added.push(t);
                    renderTags();
                    showToast('已添加: [' + t.type + '] ' + t.name, { type: 'success' });
                }
                searchInput.value = '';
                hideAutocomplete();
            };
            item.onmouseenter = function() { item.style.background = '#444'; };
            item.onmouseleave = function() { item.style.background = 'transparent'; };
            autocompleteDropdown.appendChild(item);
        });
    }

    var searchTimeout = null;
    searchInput.addEventListener('input', function() {
        if (searchTimeout) clearTimeout(searchTimeout);
        var q = this.value.trim();
        if (!q) { hideAutocomplete(); return; }
        var type = exTypeSelect.value;
        searchTimeout = setTimeout(function() {
            var xhr = new XMLHttpRequest();
            xhr.withCredentials = true;
            xhr.open('GET', '/api/comic/tags/search?type=' + encodeURIComponent(type) +
                '&q=' + encodeURIComponent(q) + '&limit=20');
            xhr.onload = function() {
                if (xhr.status !== 200) return;
                try {
                    var resp = JSON.parse(xhr.responseText);
                    renderAutocomplete((resp.body && resp.body.tags) || []);
                } catch(e) {}
            };
            xhr.send();
        }, 300);
    });

    searchInput.addEventListener('blur', function() {
        setTimeout(hideAutocomplete, 150);
    });

    searchInput.addEventListener('focus', function() {
        if (this.value.trim()) {
            var event = new Event('input');
            this.dispatchEvent(event);
        }
    });

    // 点击外部关闭
    document.addEventListener('click', function(e) {
        if (!e.target.closest('.tag-autocomplete-dropdown') && !e.target.closest('#editTagsSearchWrapper')) {
            // 不立即关闭，保留现有逻辑
        }
    });

    formContainer.appendChild(existingPanel);

    // ----- New 模式 -----
    newPanel.style.cssText = 'display: none; gap: 5px; align-items: center; flex-wrap: wrap;';

    var newTypeSelect = document.createElement('select');
    newTypeSelect.style.cssText = 'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px;';
    tagTypes.forEach(function(tt) {
        var opt = document.createElement('option');
        opt.value = tt.value; opt.textContent = tt.label;
        newTypeSelect.appendChild(opt);
    });
    newPanel.appendChild(newTypeSelect);

    var nameInput = document.createElement('input');
    nameInput.type = 'text';
    nameInput.placeholder = 'New tag name';
    nameInput.style.cssText = 'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px; flex: 1; min-width: 150px;';
    newPanel.appendChild(nameInput);

    var addBtn = document.createElement('a');
    addBtn.href = 'javascript:;';
    addBtn.className = 'btn btn-secondary';
    addBtn.textContent = 'Add';
    addBtn.style.cssText = 'padding: 4px 12px;';
    addBtn.onclick = function() {
        var name = nameInput.value.trim();
        if (!name) { showToast('请输入 tag 名称', { type: 'error' }); return; }
        var type = newTypeSelect.value;
        var urlName = encodeURIComponent(name.toLowerCase().replace(/\s+/g, '-'));
        var url = '/' + type + '/' + urlName + '/';
        var newTag = {id: 0, name: name, type: type, url: url, count: 1};
        var key = dedupKey(newTag);

        if (added.some(function(a) { return dedupKey(a) === key; })) {
            showToast('该 tag 已在添加列表中', { type: 'info' }); return;
        }
        if (currentTags.some(function(ct) { return dedupKey(ct) === key && !removed.some(function(r) { return dedupKey(r) === dedupKey(ct); }); })) {
            showToast('该 tag 已存在', { type: 'info' }); return;
        }

        added.push(newTag);
        nameInput.value = '';
        renderTags();
        showToast('已添加: ' + name, { type: 'success' });
    };
    newPanel.appendChild(addBtn);

    formContainer.appendChild(newPanel);

    // 弹窗内容
    var modalContent = document.createElement('div');
    modalContent.appendChild(tagsContainer);
    modalContent.appendChild(formContainer);

    var wrapper = showCustomModal('Edit Tags', modalContent,
        '<a href="javascript:;" class="btn btn-secondary" onclick="closeModal(this.closest(\'.modal-wrapper\'))">Cancel</a>' +
        '<a href="javascript:;" class="btn btn-primary" id="saveTagsBtn">Save</a>'
    );

    // Save 按钮逻辑
    var saveBtn = wrapper.querySelector('#saveTagsBtn');
    if (saveBtn) {
        saveBtn.onclick = function() {
            if (added.length === 0 && removed.length === 0) {
                showToast('没有变更', { type: 'info' });
                return;
            }
            var payload = JSON.stringify({
                cid: cid,
                added: added,
                removed: removed
            });
            var saveXhr = new XMLHttpRequest();
            saveXhr.withCredentials = true;
            saveXhr.open('POST', '/api/comic/tags/update');
            saveXhr.setRequestHeader('Content-Type', 'application/json');
            saveXhr.onload = function() {
                if (saveXhr.status >= 200 && saveXhr.status < 300) {
                    showToast('Tags 已更新', { type: 'success' });
                    closeModal(wrapper);
                    setTimeout(function() { location.reload(); }, 500);
                } else {
                    try {
                        var r = JSON.parse(saveXhr.responseText);
                        showToast(r.head && r.head.msg || '保存失败', { type: 'error' });
                    } catch(e) {
                        showToast('保存失败: ' + saveXhr.status, { type: 'error' });
                    }
                }
            };
            saveXhr.onerror = function() { showToast('网络错误', { type: 'error' }); };
            saveXhr.send(payload);
        };
    }
}

/**
 * 搜索页 Tag 对齐器
 */
function openTagAligner(query) {
    showToast('正在获取去重标签列表...', { type: 'info' });

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('GET', '/api/comic/tags/search-unique?q=' + encodeURIComponent(query) + '&limit=500');
    xhr.onload = function() {
        if (xhr.status !== 200) {
            showToast('获取标签列表失败', { type: 'error' });
            return;
        }
        try {
            var resp = JSON.parse(xhr.responseText);
            var data = resp.body;
            var tags = data.tags || [];
            var cidList = data.cidList || [];
            var total = data.total || 0;

            if (tags.length === 0) {
                showToast('搜索结果中没有标签', { type: 'info' });
                return;
            }

            buildTagAlignerModal(cidList, tags, query);
        } catch (e) {
            showToast('解析响应失败', { type: 'error' });
        }
    };
    xhr.onerror = function() {
        showToast('网络错误', { type: 'error' });
    };
    xhr.send();
}

function buildTagAlignerModal(cidList, tags, query) {
    var selectedTag = null;

    // 按 type 分组
    var tagGroups = {};
    tags.forEach(function(t) {
        if (!tagGroups[t.type]) tagGroups[t.type] = [];
        tagGroups[t.type].push(t);
    });

    var typeLabels = {
        'parody': 'Parodies', 'character': 'Characters', 'tag': 'Tags',
        'artist': 'Artists', 'group': 'Groups', 'language': 'Languages',
        'category': 'Categories', 'custom': 'Customs'
    };

    var content = document.createElement('div');
    var infoPara = document.createElement('p');
    infoPara.style.cssText = 'margin-bottom:10px';
    infoPara.textContent = '搜索 "' + query + '" 匹配 ' + cidList.length + ' 本漫画，共 ' + tags.length + ' 个去重标签。选择要批量添加的标签：';
    content.appendChild(infoPara);

    Object.keys(tagGroups).sort().forEach(function(type) {
        var groupTags = tagGroups[type];
        var groupDiv = document.createElement('div');
        groupDiv.style.cssText = 'margin-bottom: 8px;';

        var header = document.createElement('h4');
        header.style.cssText = 'margin: 0 0 4px 0; color: #888; font-size: 13px;';
        header.textContent = typeLabels[type] || type;
        groupDiv.appendChild(header);

        groupTags.forEach(function(t) {
            var tagEl = document.createElement('a');
            tagEl.href = 'javascript:;';
            tagEl.className = 'tag tag-' + (t.id || 0);
            tagEl.style.cssText = 'display: inline-block; margin: 2px; padding: 2px 8px; cursor: pointer;';
            tagEl.innerHTML = '<span class="name">' + t.name + '</span><span class="count">' + t.count + '</span>';

            tagEl.onclick = function() {
                // 取消之前选中
                content.querySelectorAll('.tag.selected').forEach(function(el) {
                    el.classList.remove('selected');
                    el.style.outline = '';
                });
                tagEl.classList.add('selected');
                tagEl.style.outline = '2px solid #4CAF50';
                selectedTag = {id: t.id, name: t.name, type: t.type, url: t.url, count: 1};

                // 更新"Apply"按钮状态
                var applyBtn = content.querySelector('#applyTagBtn');
                if (applyBtn) {
                    applyBtn.style.opacity = '1';
                    applyBtn.style.pointerEvents = 'auto';
                }
            };
            groupDiv.appendChild(tagEl);
        });
        content.appendChild(groupDiv);
    });

    var btnContent = document.createElement('div');
    btnContent.style.cssText = 'display: flex; gap: 8px; align-items: center; margin-top: 10px;';

    var applyBtn = document.createElement('a');
    applyBtn.id = 'applyTagBtn';
    applyBtn.href = 'javascript:;';
    applyBtn.className = 'btn btn-primary';
    applyBtn.textContent = 'Apply to All (' + cidList.length + ')';
    applyBtn.style.cssText = 'opacity: 0.4; pointer-events: none;';

    applyBtn.onclick = function() {
        if (!selectedTag) { showToast('请先选择一个标签', { type: 'error' }); return; }
        applyBtn.style.opacity = '0.6';
        applyBtn.style.pointerEvents = 'none';

        var payload = JSON.stringify({
            cidList: cidList,
            tag: selectedTag
        });

        var xhr = new XMLHttpRequest();
        xhr.withCredentials = true;
        xhr.open('POST', '/api/comic/tags/batch-add');
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.onload = function() {
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    var resp = JSON.parse(xhr.responseText);
                    var data = resp.body;
                    var msg = '标签 "' + selectedTag.name + '" 已添加到 ' + data.updated + '/' + cidList.length + ' 本漫画';
                    if (data.errors && data.errors.length > 0) {
                        msg += '，' + data.errors.length + ' 本失败';
                    }
                    showToast(msg, { type: 'success' });
                    closeModal(content.closest('.modal-wrapper'));
                    setTimeout(function() { location.reload(); }, 500);
                } catch(e) {
                    showToast('处理完成', { type: 'success' });
                    location.reload();
                }
            } else {
                try {
                    var r = JSON.parse(xhr.responseText);
                    showToast(r.head && r.head.msg || '批量添加失败', { type: 'error' });
                } catch(e) {
                    showToast('批量添加失败: ' + xhr.status, { type: 'error' });
                }
            }
        };
        xhr.onerror = function() { showToast('网络错误', { type: 'error' }); };
        xhr.send(payload);
    };
    btnContent.appendChild(applyBtn);

    var cancelBtn = document.createElement('a');
    cancelBtn.href = 'javascript:;';
    cancelBtn.className = 'btn btn-secondary';
    cancelBtn.textContent = 'Cancel';
    cancelBtn.onclick = function() {
        closeModal(content.closest('.modal-wrapper'));
    };
    btnContent.appendChild(cancelBtn);

    content.appendChild(btnContent);
    showCustomModal('Align Tags', content, '');
}

/**
 * Tag 页面加载关联 tag
 */
function loadRelatedTags(type, name) {
    var container = document.getElementById('related-tags-content');
    if (!container) return;

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('GET', '/api/comic/tags/related?type=' + encodeURIComponent(type) + '&name=' + encodeURIComponent(name) + '&limit=30');
    xhr.onload = function() {
        if (xhr.status !== 200) {
            container.innerHTML = '<p>加载失败</p>';
            return;
        }
        try {
            var resp = JSON.parse(xhr.responseText);
            var tags = resp.body && resp.body.tags;

            if (!tags || tags.length === 0) {
                container.innerHTML = '<p>No related tags found.</p>';
                return;
            }

            // 按 type 分组
            var groups = {};
            var typeLabels = {
                'parody': 'Parodies', 'character': 'Characters', 'tag': 'Tags',
                'artist': 'Artists', 'group': 'Groups', 'language': 'Languages',
                'category': 'Categories', 'custom': 'Customs'
            };

            tags.forEach(function(t) {
                if (!groups[t.type]) groups[t.type] = [];
                groups[t.type].push(t);
            });

            var html = '';
            Object.keys(groups).sort().forEach(function(type) {
                var groupTags = groups[type];
                html += '<div class="tag-container field-name"><strong>' + (typeLabels[type] || type) + ':</strong> <span class="tags">';
                groupTags.forEach(function(t) {
                    var likeClass = t.like ? ' tag-like' : '';
                    html += '<a href="/tag' + t.url + '" class="tag tag-' + (t.id || 0) + likeClass + '">' +
                        '<span class="name">' + t.name + '</span>' +
                        '<span class="count">' + t.count + '</span></a>';
                });
                html += '</span></div>';
            });
            container.innerHTML = html;
        } catch (e) {
            container.innerHTML = '<p>解析失败</p>';
        }
    };
    xhr.onerror = function() {
        container.innerHTML = '<p>网络错误</p>';
    };
    xhr.send();
}
/**
 * Tag 关系管理弹窗
 */
function openTagRelationManager(type, name, id) {
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('GET', '/api/comic/tags/relation?type=' + encodeURIComponent(type) + '&name=' + encodeURIComponent(name) + '&id=' + encodeURIComponent(id));
    xhr.onload = function() {
        if (xhr.status !== 200) { showToast('获取关系列表失败', { type: 'error' }); return; }
        try {
            var resp = JSON.parse(xhr.responseText);
            var groups = (resp.body && resp.body.groups) || [];
            buildRelationModal(type, name, id, groups);
        } catch(e) { showToast('解析失败', { type: 'error' }); }
    };
    xhr.onerror = function() { showToast('网络错误', { type: 'error' }); };
    xhr.send();
}

/**
 * 详情页缩略图缩放控制
 */
function initThumbnailZoom() {
    var slider = document.getElementById('thumbZoomSlider');
    var zoomValue = document.getElementById('zoomValue');
    var zoomInBtn = document.getElementById('zoomInBtn');
    var zoomOutBtn = document.getElementById('zoomOutBtn');
    var container = document.getElementById('thumbnail-container');
    if (!slider || !container) return;

    // 从 localStorage 恢复
    var saved = localStorage.getItem('thumbZoom');
    if (saved) {
        var v = parseInt(saved, 10);
        if (!isNaN(v) && v >= 60 && v <= 1200) {
            slider.value = v;
        }
    }

    function applyZoom(val) {
        container.style.setProperty('--thumb-w', val + 'px');
        if (zoomValue) zoomValue.textContent = val;
        localStorage.setItem('thumbZoom', String(val));
    }

    // 初始应用
    applyZoom(parseInt(slider.value, 10));

    slider.addEventListener('input', function() {
        applyZoom(parseInt(this.value, 10));
    });

    if (zoomInBtn) {
        zoomInBtn.addEventListener('click', function() {
            var v = Math.min(1200, parseInt(slider.value, 10) + 20);
            slider.value = v;
            applyZoom(v);
        });
    }

    if (zoomOutBtn) {
        zoomOutBtn.addEventListener('click', function() {
            var v = Math.max(60, parseInt(slider.value, 10) - 20);
            slider.value = v;
            applyZoom(v);
        });
    }
}

// 页面加载后执行
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initThumbnailZoom);
} else {
    initThumbnailZoom();
}

/**
 * LoadingManager — 按钮级 loading 状态管理
 * 用法: LoadingManager.start(btnEl); LoadingManager.done(btnEl);
 */
const LoadingManager = {
  start(btn) {
    if (!btn || btn.dataset.loading) return;
    btn.dataset.loading = 'true';
    btn.dataset.origHTML = btn.innerHTML;
    btn.classList.add('btn-loading');
    btn.disabled = true;
  },
  done(btn) {
    if (!btn) return;
    delete btn.dataset.loading;
    btn.classList.remove('btn-loading', 'btn-error');
    btn.disabled = false;
  },
  error(btn) {
    if (!btn) return;
    delete btn.dataset.loading;
    btn.classList.remove('btn-loading');
    btn.classList.add('btn-error');
    btn.disabled = false;
    setTimeout(function() {
      if (btn) btn.classList.remove('btn-error');
    }, 400);
  }
};

function showProgressToast(message, percent) {
  var container = document.getElementById('messages');
  if (!container) return;
  var existing = document.getElementById('progress-toast');
  if (existing) {
    var bar = existing.querySelector('.progress-bar');
    if (bar) bar.style.width = Math.min(100, Math.max(0, percent || 0)) + '%';
    var msg = existing.querySelector('.progress-msg');
    if (msg) msg.textContent = message;
    return;
  }
  var toast = document.createElement('div');
  toast.id = 'progress-toast';
  toast.className = 'alert alert-info fade-slide-in open';
  toast.style.cssText = 'padding: 8px 12px;';
  toast.innerHTML = '<div class="progress-msg">' + message + '</div>' +
    '<div style="margin-top:4px;height:4px;background:#444;border-radius:2px;overflow:hidden;">' +
    '<div class="progress-bar" style="width:' + Math.min(100, Math.max(0, percent || 0)) + '%;height:100%;background:#4CAF50;transition:width 0.3s;"></div></div>';
  container.appendChild(toast);
}

/**
 * OptimisticUpdater — 乐观更新 + 局部刷新工具
 */
const OptimisticUpdater = {
  // 乐观 Toggle：立即切换 class，请求失败回滚
  optimisticToggle(btn, activeClass, inactiveClass) {
    var wasActive = btn.classList.contains(activeClass);
    var rollbackState = { activeClass, inactiveClass, wasActive };
    // 立即切换
    btn.classList.remove(activeClass, inactiveClass);
    btn.classList.add(wasActive ? inactiveClass : activeClass);
    return {
      rollback: function() {
        btn.classList.remove(activeClass, inactiveClass);
        btn.classList.add(rollbackState.wasActive ? activeClass : inactiveClass);
      },
      wasActive: wasActive
    };
  },

  // 局部刷新：fetch 数据并执行 render 函数替换容器内容
  refreshContainer(url, containerSelector, renderFn) {
    var container = document.querySelector(containerSelector);
    if (!container) return Promise.reject('Container not found: ' + containerSelector);
    return fetch(url, { credentials: 'include' })
      .then(function(r) { return r.json(); })
      .then(function(data) {
        if (renderFn && typeof renderFn === 'function') {
          renderFn(container, data);
        }
        return data;
      });
  }
};

/**
 * 自动补全下拉键盘导航
 * @param {HTMLElement} dropdown - 下拉容器
 * @param {Function} onSelect - 选中回调，接收当前高亮项索引
 * @returns {Function} destroy 函数
 */
function enableAutocompleteKeyboardNav(dropdown, onSelect) {
  var selectedIdx = -1;

  function getItems() {
    return dropdown.querySelectorAll('div');
  }

  function highlight(idx) {
    var items = getItems();
    items.forEach(function(el, i) {
      el.classList.remove('keyboard-selected');
      el.style.background = i === idx ? '#444' : 'transparent';
    });
  }

  function handler(e) {
    var items = getItems();
    if (items.length === 0) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIdx = (selectedIdx + 1) % items.length;
      highlight(selectedIdx);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIdx = (selectedIdx - 1 + items.length) % items.length;
      highlight(selectedIdx);
    } else if (e.key === 'Enter' && selectedIdx >= 0 && items[selectedIdx]) {
      e.preventDefault();
      items[selectedIdx].click();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      dropdown.style.display = 'none';
      selectedIdx = -1;
    }
  }

  dropdown.addEventListener('keydown', handler);
  return function destroy() {
    dropdown.removeEventListener('keydown', handler);
  };
}

// 在输入框上绑定键盘导航（输入框的 keydown 事件委托给 dropdown）
function bindAutocompleteKeys(input, dropdown, onEnter) {
  input.addEventListener('keydown', function(e) {
    if (dropdown.style.display === 'none' || !dropdown.children.length) {
      if (e.key === 'Enter' && onEnter) {
        e.preventDefault();
        onEnter();
      }
      return;
    }
    var items = dropdown.querySelectorAll('div');
    var selected = dropdown.querySelector('.keyboard-selected');
    var idx = -1;
    if (selected) {
      for (var i = 0; i < items.length; i++) {
        if (items[i] === selected) { idx = i; break; }
      }
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      idx = (idx + 1) % items.length;
      items.forEach(function(el, i) {
        el.classList.toggle('keyboard-selected', i === idx);
        el.style.background = i === idx ? '#444' : 'transparent';
      });
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      idx = (idx - 1 + items.length) % items.length;
      items.forEach(function(el, i) {
        el.classList.toggle('keyboard-selected', i === idx);
        el.style.background = i === idx ? '#444' : 'transparent';
      });
    } else if (e.key === 'Enter' && idx >= 0 && items[idx]) {
      e.preventDefault();
      items[idx].click();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      dropdown.style.display = 'none';
    }
  });
}

