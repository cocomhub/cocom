/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Gallery action functions: like, archive, restore, verify, etc.
 */
(function () {
  'use strict';

  window.addLikeGroup = function addLikeGroup(cid) {
    var btn = document.getElementById('sidebarLikeBtn');
    if (!btn) btn = document.getElementById('addLikeGroup');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var liked = btn.classList.contains('btn-primary');
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/like');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');

    // Optimistic update: immediately switch UI
    var toggle = window.OptimisticUpdater.optimisticToggle(
      btn,
      'btn-primary',
      'btn-secondary',
    );
    var label = btn.querySelector('.label');
    if (label) label.textContent = liked ? 'Like' : 'Liked';

    xhr.onload = function () {
      window.LoadingManager.done(btn);
      if (xhr.status >= 200 && xhr.status < 300) {
        var detailLikeTag = document.querySelector('.tag-99999');
        if (liked && detailLikeTag) {
          detailLikeTag.remove();
        } else if (!liked) {
          window.addLikeTag();
        }
        window.showToast(liked ? '已取消 Like' : '已添加 Like', {
          type: 'success',
        });
      } else {
        toggle.rollback();
        if (label) label.textContent = liked ? 'Liked' : 'Like';
        window.showToast('操作失败', { type: 'error' });
      }
    };
    xhr.onerror = function () {
      window.LoadingManager.done(btn);
      toggle.rollback();
      if (label) label.textContent = liked ? 'Liked' : 'Like';
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send('cid=' + encodeURIComponent(cid));
  };

  window.findCustomsContainer = function findCustomsContainer() {
    const containers = document.querySelectorAll('.tag-container.field-name');
    for (const c of containers) {
      const text = (c.textContent || '').trim();
      if (text.startsWith('Customs')) {
        return c;
      }
    }
    return null;
  };

  window.addLikeTag = function addLikeTag() {
    const container = window.findCustomsContainer();
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
  };

  window.removeLikeTag = function removeLikeTag() {
    const container = window.findCustomsContainer();
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
  };

  window.formatError = function formatError(resp) {
    var code = resp && resp.head ? resp.head.code : -1;
    var msg = resp && resp.head ? resp.head.msg || resp.head.message || '' : '';
    return '[' + code + '] ' + (msg || '请求失败');
  };

  window.highlightInvalidPages = function highlightInvalidPages(indexes) {
    if (!Array.isArray(indexes) || indexes.length === 0) return;
    var container = document.getElementById('thumbnail-container');
    if (!container) return;
    indexes.forEach(function (it) {
      var idx = it.index || it;
      var link = container.querySelector(
        'a.gallerythumb[href="/g/' +
          String((window._gallery && window._gallery.cid) || '') +
          '/' +
          String(idx) +
          '/"]',
      );
      if (link && link.parentElement) {
        link.parentElement.style.outline = '3px solid #e74c3c';
      }
    });
  };

  window.ensureForceArchiveButton = function ensureForceArchiveButton(cid) {
    var existing = document.getElementById('forceArchiveBtn');
    if (existing) return;
    var sidebar = document.querySelector('.left-action-sidebar');
    if (!sidebar) return;
    var a = document.createElement('a');
    a.id = 'forceArchiveBtn';
    a.href = 'javascript:;';
    a.className = 'sidebar-btn';
    a.innerHTML =
      '<i class="fa fa-exclamation-triangle"></i><span class="label">强制归档</span>';
    a.onclick = function () {
      window.archiveComicForce(cid);
    };
    sidebar.appendChild(a);
  };

  window.archiveComic = function archiveComic(cid) {
    var btn = document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive');
    xhr.onload = function () {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('已归档', { type: 'success' });
          btn.innerHTML =
            '<i class="fa fa-undo"></i><span class="label">恢复</span>';
          btn.onclick = function () {
            window.restoreComic(cid);
          };
          btn.id = 'sidebarArchiveBtn';
        } else {
          var msg = window.formatError(resp);
          window.showToast(msg, { type: 'error' });
          if (resp.head.code === -1001) {
            var invalids = (resp.body && resp.body.invalid_images) || [];
            if (
              !invalids.length &&
              window._gallery &&
              window._gallery.images &&
              Array.isArray(window._gallery.images.pages)
            ) {
              window._gallery.images.pages.forEach(function (p, i) {
                if (p && p.status === false) invalids.push({ index: i + 1 });
              });
            }
            window.highlightInvalidPages(invalids);
            window.ensureForceArchiveButton(cid);
            window.showToast(
              '检测到异常图片，建议先"修复漫画状态"，或使用"强制归档"',
              { type: 'info' },
            );
          }
        }
      } else {
        window.LoadingManager.error(btn);
        var msg = xhr.responseText || '请求失败: ' + xhr.status;
        window.showToast(msg, { type: 'error' });
      }
    };
    xhr.onerror = function () {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send();
  };

  window.archiveComicForce = function archiveComicForce(cid) {
    var btn =
      document.getElementById('forceArchiveBtn') ||
      document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(
      'POST',
      '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive?force=true',
    );
    xhr.onload = function () {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('已强制归档', { type: 'success' });
          btn.innerHTML =
            '<i class="fa fa-undo"></i><span class="label">恢复</span>';
          btn.onclick = function () {
            window.restoreComic(cid);
          };
        } else {
          var msg = window.formatError(resp);
          window.showToast(msg, { type: 'error' });
        }
      } else {
        window.LoadingManager.error(btn);
        var msg = xhr.responseText || '请求失败: ' + xhr.status;
        window.showToast(msg, { type: 'error' });
      }
    };
    xhr.onerror = function () {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send();
  };

  window.restoreComic = function restoreComic(cid) {
    var btn = document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/restore');
    xhr.onload = function () {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('已恢复', { type: 'success' });
          btn.innerHTML =
            '<i class="fa fa-archive"></i><span class="label">归档</span>';
          btn.onclick = function () {
            window.archiveComic(cid);
          };
          btn.id = 'sidebarArchiveBtn';
        } else {
          var msg = window.formatError(resp);
          window.showToast(msg, { type: 'error' });
        }
      } else {
        window.LoadingManager.error(btn);
        var msg = xhr.responseText || '请求失败: ' + xhr.status;
        window.showToast(msg, { type: 'error' });
      }
    };
    xhr.onerror = function () {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send();
  };

  window.verifyComic = function verifyComic(cid) {
    var btn = document.getElementById('sidebarFixBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/verify');
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function () {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('修复任务已启动', { type: 'success' });
        } else {
          var msg = window.formatError(resp);
          window.showToast(msg, { type: 'error' });
        }
      } else {
        window.LoadingManager.error(btn);
        var msg = xhr.responseText || '请求失败: ' + xhr.status;
        window.showToast(msg, { type: 'error' });
      }
    };
    xhr.onerror = function () {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    var body = { id: String(cid), autoFix: true, maxWorkers: 1 };
    xhr.send(JSON.stringify(body));
  };
})();
