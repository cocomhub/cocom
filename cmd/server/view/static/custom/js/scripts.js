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
    var typeClass = 'alert-info';
    if (type === 'success') typeClass = 'alert-success';
    else if (type === 'error') typeClass = 'alert-danger';
    var container = document.getElementById('messages');
    if (!container) return;
    var alert = document.createElement('div');
    alert.className = 'alert ' + typeClass + ' fade-slide-in open';
    alert.textContent = message;
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