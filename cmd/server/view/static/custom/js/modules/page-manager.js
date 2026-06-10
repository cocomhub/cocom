/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

// page-manager.js — Detail 页页管理功能
(function() {
  'use strict';

  var state = {
    active: false,
    mode: null,           // null | 'delete' | 'insert' | 'replace' | 'reorder'
    changes: [],          // { type, data, timestamp }
    selectedPages: [],
    insertedPages: [],
  };

  var bar = document.getElementById('page-manager-bar');
  var statusEl = document.getElementById('pm-status');
  var container = document.getElementById('thumbnail-container');
  var insertForm = document.getElementById('insert-form');

  // ---- 模式切换 ----
  window.togglePageManager = function() {
    if (state.active) {
      pmExit();
    } else {
      pmEnter();
    }
  };

  function pmEnter() {
    state.active = true;
    if (bar) bar.style.display = 'flex';
    if (container) container.classList.add('page-manager-mode');
    // 绑定缩略图点击事件
    document.querySelectorAll('.gallery-thumb, .thumb-container').forEach(function(el) {
      el.style.cursor = 'pointer';
      el.addEventListener('click', onThumbClick);
    });
    updateStatus();
  }

  function pmExit() {
    state.active = false;
    state.mode = null;
    if (bar) bar.style.display = 'none';
    if (insertForm) insertForm.style.display = 'none';
    if (container) container.classList.remove('page-manager-mode');
    // 移除缩略图点击事件
    document.querySelectorAll('.gallery-thumb, .thumb-container').forEach(function(el) {
      el.style.cursor = '';
      el.removeEventListener('click', onThumbClick);
      el.classList.remove('page-deleted', 'page-selected');
    });
  }
  window.pmExit = pmExit;

  // ---- 缩略图点击 ----
  function onThumbClick(e) {
    if (!state.mode || state.mode === 'insert') return;
    var el = e.currentTarget;
    var page = parseInt(el.getAttribute('data-page') || el.getAttribute('data-index'), 10);
    if (isNaN(page)) return;

    if (state.mode === 'delete') {
      togglePageDelete(page, el);
    } else if (state.mode === 'replace') {
      triggerPageReplace(page);
    } else if (state.mode === 'reorder') {
      // 简单交换
      var prev = state.changes.filter(function(c) { return c.type === 'reorder'; });
      if (prev.length > 0 && prev[prev.length-1].data.from === page) {
        // 已有此页的重排标记，忽略
        return;
      }
    }
    updateStatus();
  }

  function togglePageDelete(page, el) {
    var idx = state.selectedPages.indexOf(page);
    if (idx >= 0) {
      state.selectedPages.splice(idx, 1);
      el.classList.remove('page-deleted');
      // 从变更中移除
      state.changes = state.changes.filter(function(c) { return !(c.type === 'delete' && c.data === page); });
    } else {
      state.selectedPages.push(page);
      el.classList.add('page-deleted');
      state.changes.push({ type: 'delete', data: page, timestamp: Date.now() });
    }
  }

  // ---- 删除模式 ----
  window.pmDeleteMode = function() {
    state.mode = 'delete';
    state.selectedPages = [];
  };

  // ---- 插入模式 ----
  window.pmInsertMode = function() {
    state.mode = 'insert';
    if (insertForm) insertForm.style.display = 'block';
  };

  window.pmFetchPreview = function() {
    var sourceCID = parseInt(document.getElementById('insert-source-cid').value, 10);
    if (!sourceCID) { showToast('请输入源 CID', 'error'); return; }

    fetch('/api/comic/getComicPages', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cid: sourceCID }),
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.head && data.head.code === 0 && data.body && data.body.pages) {
        renderInsertPreview(data.body.pages, sourceCID);
      } else {
        showToast('获取页面失败: ' + (data.head ? data.head.msg : '未知错误'), 'error');
      }
    })
    .catch(function(err) { showToast('请求失败: ' + err.message, 'error'); });
  };

  function renderInsertPreview(pages, sourceCID) {
    var previewContainer = document.getElementById('insert-preview');
    if (!previewContainer) return;
    previewContainer.style.display = 'flex';
    previewContainer.innerHTML = '';
    state.insertedPages = [];

    pages.forEach(function(p) {
      var item = document.createElement('div');
      item.className = 'preview-item';
      item.setAttribute('data-page', p.page);
      item.innerHTML = '<img src="' + escapeAttr(p.thumb_url) + '" alt="Page ' + p.page + '" loading="lazy">' +
        '<div class="page-num">' + p.page + '</div>';
      item.addEventListener('click', function() {
        this.classList.toggle('selected');
        var pg = parseInt(this.getAttribute('data-page'), 10);
        var idx = state.insertedPages.indexOf(pg);
        if (idx >= 0) { state.insertedPages.splice(idx, 1); }
        else { state.insertedPages.push(pg); }
      });
      previewContainer.appendChild(item);
    });
    previewContainer.setAttribute('data-source-cid', sourceCID);
    // 默认不选中任何页面，用户点击选择
  }

  window.pmConfirmInsert = function() {
    var previewContainer = document.getElementById('insert-preview');
    if (!previewContainer || state.insertedPages.length === 0) {
      showToast('请先选择要插入的页面', 'error');
      return;
    }
    var sourceCID = parseInt(previewContainer.getAttribute('data-source-cid'), 10);
    var afterPage = parseInt(document.getElementById('insert-after-page').value, 10) || 0;

    state.changes.push({ type: 'insert', data: { source_cid: sourceCID, pages: state.insertedPages.slice(), after_page: afterPage }, timestamp: Date.now() });
    showToast('已标记插入 ' + state.insertedPages.length + ' 页，点击保存生效', 'info');
    pmCancelInsert();
    updateStatus();
  };

  window.pmCancelInsert = function() {
    if (insertForm) insertForm.style.display = 'none';
    var preview = document.getElementById('insert-preview');
    if (preview) { preview.style.display = 'none'; preview.innerHTML = ''; }
    state.insertedPages = [];
    state.mode = null;
  };

  // ---- 替换模式 ----
  window.pmReplaceMode = function() { state.mode = 'replace'; };

  // ---- 重排模式 ----
  window.pmReorderMode = function() { state.mode = 'reorder'; };

  // ---- 撤销 ----
  window.pmUndo = function() {
    if (state.changes.length === 0) { showToast('没有可撤销的变更', 'info'); return; }
    state.changes.pop();
    updateStatus();
    showToast('已撤销上一步操作', 'info');
  };

  // ---- 保存 ----
  window.pmSave = function() {
    if (state.changes.length === 0) { showToast('没有变更需要保存', 'info'); return; }

    var payload = {
      cid: window._gallery ? window._gallery.CID : null,
      pages: state.changes.map(function(c) {
        if (c.type === 'delete') return { page: c.data, action: 'delete' };
        if (c.type === 'insert') return { page: c.data.after_page, action: 'insert', source_cid: c.data.source_cid, source_pages: c.data.pages };
        if (c.type === 'replace') return { page: c.data, action: 'replace' };
        if (c.type === 'reorder') return { page: c.data, action: 'reorder' };
        return null;
      }).filter(Boolean),
    };

    fetch('/api/comic/savePages', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.head && data.head.code === 0) {
        showToast('保存成功！归档已标记为过期', 'success');
        state.changes = [];
        pmExit();
        if (confirm('页面已变更，是否立即重新归档？')) {
          if (window.reArchive) window.reArchive();
        }
      } else {
        showToast('保存失败: ' + (data.head ? data.head.msg : '未知错误'), 'error');
      }
    })
    .catch(function(err) { showToast('请求失败: ' + err.message, 'error'); });
  };

  // ---- 删除确认 ----
  window.openDeleteConfirm = function() {
    var title = window._gallery && window._gallery.Title ? (window._gallery.Title.english || '') : '';
    var input = prompt('输入 comic 标题以确认删除:\n"一旦删除无法恢复"\n\n标题: ' + title);
    if (input && input.trim() === title.trim()) {
      var cid = window._gallery ? window._gallery.CID : null;
      fetch('/api/admin/comic/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cid: cid }),
      })
      .then(function(r) { return r.json(); })
      .then(function(data) {
        if (data.head && data.head.code === 0) {
          showToast('删除成功', 'success');
          window.location.href = '/';
        } else {
          showToast('删除失败: ' + (data.head ? data.head.msg : '未知错误'), 'error');
        }
      })
      .catch(function(err) { showToast('请求失败: ' + err.message, 'error'); });
    } else if (input !== null) {
      showToast('标题不匹配，删除取消', 'error');
    }
  };

  // ---- 状态更新 ----
  function updateStatus() {
    if (statusEl) statusEl.textContent = '未保存变更: ' + state.changes.length;
  }

  // ---- 工具 ----
  function escapeAttr(text) {
    return String(text).replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function showToast(msg, type) {
    if (window.showAdminToast) { window.showAdminToast(msg, type); }
    else if (window.showToast) { window.showToast(msg, type); }
    else { alert(msg); }
  }
})();
