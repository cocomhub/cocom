/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Entry point — loads all module files and initializes page-specific logic.
 * Module files are loaded separately via <script> tags in head.tpl.
 */

// 全局搜索快捷键：按 / 聚焦搜索框
document.addEventListener('keydown', function (e) {
  var tag = e.target.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target.isContentEditable)
    return;
  if (e.key === '/') {
    e.preventDefault();
    var searchInput = document.querySelector('input[type="search"]');
    if (searchInput) {
      searchInput.focus();
      searchInput.select();
    }
  }
});

// 搜索框按 Esc 失焦
document.addEventListener('keydown', function (e) {
  if (e.key === 'Escape' && e.target && e.target.type === 'search') {
    e.target.blur();
  }
});

// 首页自动聚焦搜索框
(function () {
  var path = window.location.pathname;
  if (path === '/' || path === '/search/') {
    var searchInput = document.querySelector('input[type="search"]');
    if (searchInput && !searchInput.value) {
      setTimeout(function () {
        searchInput.focus();
      }, 300);
    }
  }
})();

// 页面初始化
function initGalleryPage() {
  if (typeof initThumbnailZoom === 'function') initThumbnailZoom();
  if (typeof initLargeModeToggle === 'function') initLargeModeToggle();
  if (typeof initKeyboardShortcuts === 'function') initKeyboardShortcuts();
  if (typeof initSearchAutocomplete === 'function') initSearchAutocomplete();
}
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initGalleryPage);
} else {
  initGalleryPage();
}
