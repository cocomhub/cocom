/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

// quick-link.js — Index 页快速链接与对比功能
(function () {
  'use strict';

  // ---- 状态 ----
  var state = {
    mode: null, // null | 'link' | 'compare'
    mainCID: null, // 主 comic 的 CID（链接模式）
    selectedCIDs: [], // 已选中的 CID 列表
  };

  // ---- 配置 ----
  var STORAGE_KEY = 'comic_link_target';

  // ---- DOM 引用（惰性初始化，避免 <head> 加载时 DOM 未就绪） ----
  var sidebar, statusEl, statusInfo, statusActions;

  function ensureDOM() {
    if (statusEl) return true;
    sidebar = document.getElementById('quick-action-sidebar');
    statusEl = document.getElementById('sidebar-status');
    statusInfo = statusEl ? statusEl.querySelector('.status-info') : null;
    statusActions = statusEl ? statusEl.querySelector('.status-actions') : null;
    return !!statusEl;
  }

  // ---- 初始化 ----
  function init() {
    // 恢复新标签打开设置
    var checkbox = document.getElementById('comic-link-target');
    if (checkbox) {
      var saved = localStorage.getItem(STORAGE_KEY);
      checkbox.checked = saved !== '_self'; // 默认新标签打开
      checkbox.addEventListener('change', function () {
        localStorage.setItem(
          STORAGE_KEY,
          checkbox.checked ? '_blank' : '_self',
        );
      });
    }
  }

  // ---- 工具 ----
  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
  }

  function showToast(msg, type) {
    if (window.showAdminToast) {
      window.showAdminToast(msg, type);
    } else if (window.showToast) {
      window.showToast(msg, type);
    } else {
      alert(msg);
    }
  }

  // ---- 模式切换 ----
  window.toggleLinkMode = function () {
    if (state.mode === 'link') {
      // 已有选中时触发确认，否则退出
      if (state.mainCID || state.selectedCIDs.length > 0) {
        confirmLinkAction();
      } else {
        exitMode();
      }
      return;
    }
    enterMode('link');
  };

  window.toggleCompareMode = function () {
    if (state.mode === 'compare') {
      // 已有选中时触发确认，否则退出
      if (state.selectedCIDs.length > 0) {
        confirmCompareAction();
      } else {
        exitMode();
      }
      return;
    }
    enterMode('compare');
  };

  function enterMode(mode) {
    state.mode = mode;
    ensureDOM(); // 确保 DOM 引用就绪
    state.mainCID = null;
    state.selectedCIDs = [];
    updateUI();

    // 给所有 comic 卡片添加可交互样式
    document.querySelectorAll('.gallery').forEach(function (card) {
      card.classList.add('link-selectable');
      card.addEventListener('click', onCardClick);
    });

    // 禁用默认点击跳转
    document.querySelectorAll('.gallery a').forEach(function (a) {
      a.addEventListener('click', preventDefaultClick);
    });

    var btn = document.getElementById(
      state.mode === 'link' ? 'btn-link-mode' : 'btn-compare-mode',
    );
    if (btn) btn.classList.add('active');
  }

  function exitMode() {
    state.mode = null;
    state.mainCID = null;
    state.selectedCIDs = [];
    if (statusEl) {
      statusEl.style.display = 'none';
    }

    document.querySelectorAll('.gallery').forEach(function (card) {
      card.classList.remove('link-selectable', 'selected-main', 'selected-sub');
      card.removeEventListener('click', onCardClick);
    });
    document.querySelectorAll('.gallery a').forEach(function (a) {
      a.removeEventListener('click', preventDefaultClick);
    });

    var linkBtn = document.getElementById('btn-link-mode');
    var cmpBtn = document.getElementById('btn-compare-mode');
    if (linkBtn) linkBtn.classList.remove('active');
    if (cmpBtn) cmpBtn.classList.remove('active');
  }

  function preventDefaultClick(e) {
    e.preventDefault();
    // 不调 stopPropagation：让事件冒泡到 .gallery 上的 onCardClick
  }

  // ---- 卡片点击 ----
  function onCardClick(e) {
    if (!state.mode) return;
    e.preventDefault();

    var card = e.currentTarget;
    var cid = parseInt(card.getAttribute('data-cid'), 10);
    // 如果 data-cid 没有，尝试从 a 标签的 href 提取
    if (!cid) {
      var link = card.querySelector('a');
      if (link) {
        var m = link.getAttribute('href').match(/\/g\/(\d+)\//);
        if (m) cid = parseInt(m[1], 10);
      }
    }
    if (!cid) return;

    if (state.mode === 'link') {
      handleLinkClick(cid, card);
    } else if (state.mode === 'compare') {
      handleCompareClick(cid, card);
    }
    updateUI();
  }

  // ---- 链接模式点击 ----
  function handleLinkClick(cid, card) {
    if (cid === state.mainCID) {
      state.mainCID = null;
      card.classList.remove('selected-main');
      return;
    }

    var idx = state.selectedCIDs.indexOf(cid);
    if (idx >= 0) {
      state.selectedCIDs.splice(idx, 1);
      card.classList.remove('selected-sub');
      card.removeAttribute('data-order');
      return;
    }

    if (!state.mainCID) {
      state.mainCID = cid;
      card.classList.add('selected-main');
    } else {
      state.selectedCIDs.push(cid);
      card.classList.add('selected-sub');
      card.setAttribute('data-order', state.selectedCIDs.length);
    }
  }

  // ---- 对比模式点击 ----
  function handleCompareClick(cid, card) {
    var idx = state.selectedCIDs.indexOf(cid);
    if (idx >= 0) {
      state.selectedCIDs.splice(idx, 1);
      card.classList.remove('selected-sub');
      card.removeAttribute('data-order');
    } else {
      state.selectedCIDs.push(cid);
      card.classList.add('selected-sub');
      card.setAttribute('data-order', state.selectedCIDs.length);
    }
  }

  // ---- 更新界面 ----
  function updateUI() {
    if (!statusEl || !statusInfo) return;

    if (!state.mode) {
      statusEl.style.display = 'none';
      return;
    }

    statusEl.style.display = 'block';
    var modeLabel = state.mode === 'link' ? '链接模式' : '对比模式';
    var html = '<strong>' + escapeHtml(modeLabel) + '</strong><br>';

    if (state.mode === 'link') {
      if (state.mainCID) {
        html +=
          '主 comic: <strong>' + escapeHtml(state.mainCID) + '</strong> | ';
        html +=
          '备 comic: <strong>' +
          escapeHtml(state.selectedCIDs.length) +
          '</strong> 个';
        if (statusActions) statusActions.style.display = 'flex';
      } else {
        html += '请点击选择主 comic（⭐）';
        if (statusActions) statusActions.style.display = 'none';
      }
    } else {
      html +=
        '已选择: <strong>' +
        escapeHtml(state.selectedCIDs.length) +
        '</strong> 个 (最少 2 个)';
      if (statusActions) {
        statusActions.style.display =
          state.selectedCIDs.length >= 2 ? 'flex' : 'none';
      }
    }

    statusInfo.innerHTML = html;
  }

  // ---- 确认操作 ----
  window.confirmAction = function () {
    if (state.mode === 'link') {
      confirmLinkAction();
    } else if (state.mode === 'compare') {
      confirmCompareAction();
    }
  };

  window.cancelAction = function () {
    exitMode();
  };

  function confirmLinkAction() {
    if (!state.mainCID || state.selectedCIDs.length === 0) {
      showToast('请选择主 comic 和至少一个备 comic', 'error');
      return;
    }

    if (
      !confirm(
        '确认将 ' +
          state.selectedCIDs.length +
          ' 个备 comic 链接到 ' +
          state.mainCID +
          '？',
      )
    )
      return;

    fetch('/api/admin/comic/link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        main_cid: state.mainCID,
        sub_cids: state.selectedCIDs,
      }),
    })
      .then(function (r) {
        return r.json();
      })
      .then(function (data) {
        if (data.head && data.head.code === 0) {
          showToast('链接成功！', 'success');
          exitMode();
          location.reload();
        } else {
          showToast(
            '链接失败: ' + (data.head ? data.head.msg : '未知错误'),
            'error',
          );
        }
      })
      .catch(function (err) {
        showToast('请求失败: ' + err.message, 'error');
      });
  }

  function confirmCompareAction() {
    if (state.selectedCIDs.length < 2) {
      showToast('请至少选择 2 个漫画进行对比', 'error');
      return;
    }
    window.location.href = '/admin?cids=' + state.selectedCIDs.join(',');
  }

  // ---- 侧边栏折叠 ----
  window.toggleSidebar = function () {
    if (!sidebar) return;
    sidebar.classList.toggle('collapsed');
  };

  // ---- 键盘快捷键 ----
  document.addEventListener('keydown', function (e) {
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;

    if (e.key === 'l' || e.key === 'L') {
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        window.toggleLinkMode();
      }
    } else if (e.key === 'c' || e.key === 'C') {
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        window.toggleCompareMode();
      }
    } else if (e.key === 'Escape' || e.key === 'Esc') {
      if (state.mode) {
        e.preventDefault();
        exitMode();
      }
    }
  });

  // ---- 初始化 ----
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
