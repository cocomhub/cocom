// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

(function () {
  'use strict';

  // 页面加载后自动加载所有推荐
  document.addEventListener('DOMContentLoaded', function () {
    var container = document.getElementById('recommend-container');
    if (!container) return;
    var cid = container.getAttribute('data-cid');
    if (!cid) return;
    container.style.display = ''; // 显示容器
    var sections = container.querySelectorAll('[data-recommend-type]');
    sections.forEach(function (section) {
      var type = section.getAttribute('data-recommend-type');
      loadRecommendations(cid, type, section);
    });
  });

  /**
   * 刷新指定维度的推荐
   * @param {HTMLElement} btn 点击的刷新按钮
   * @param {string} tagType 推荐维度类型
   */
  window.refreshRecommend = function (btn, tagType) {
    var section = btn.closest('[data-recommend-type]');
    if (!section) return;
    var container = document.getElementById('recommend-container');
    if (!container) return;
    var cid = container.getAttribute('data-cid');
    loadRecommendations(cid, tagType, section);
  };

  /**
   * 从 API 加载推荐数据
   * @param {number|string} cid 当前漫画 ID
   * @param {string} tagType 推荐维度类型
   * @param {HTMLElement} section 推荐区域 DOM 元素
   */
  function loadRecommendations(cid, tagType, section) {
    var grid = section.querySelector('.recommend-grid');
    if (!grid) return;

    // 显示骨架屏
    grid.innerHTML =
      '<div class="skeleton-grid">' +
      '<div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>'.repeat(
        5,
      ) +
      '</div>';

    fetch(
      '/api/comic/recommendations?cid=' +
        encodeURIComponent(cid) +
        '&type=' +
        encodeURIComponent(tagType),
    )
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function (data) {
        renderRecommendGrid(grid, data.results || []);
      })
      .catch(function () {
        grid.innerHTML =
          '<div class="empty-state"><i class="fa fa-exclamation-circle"></i><p>加载失败，点击刷新重试</p></div>';
      });
  }

  /**
   * 渲染推荐网格
   * @param {HTMLElement} grid 网格容器
   * @param {Array} comics 漫画列表
   */
  function renderRecommendGrid(grid, comics) {
    if (!comics || comics.length === 0) {
      grid.innerHTML =
        '<div class="empty-state"><i class="fa fa-inbox"></i><p>暂无推荐</p></div>';
      return;
    }
    var html = '';
    comics.forEach(function (c) {
      html +=
        '<div class="gallery" data-tags="' +
        (c.tags_id_string || '') +
        '">' +
        '<a href="/g/' +
        c.cid +
        '/" class="cover" style="padding:0 0 141.6% 0">' +
        '<img class="lazyload" width="250" height="354" ' +
        'data-src="/galleries/' +
        c.media_id +
        '/' +
        c.cover_name +
        '" ' +
        'src="/galleries/' +
        c.media_id +
        '/' +
        c.cover_name +
        '" />' +
        '<div class="caption">' +
        escapeHtml(c.title_english || '') +
        '</div>' +
        '</a>' +
        '</div>';
    });
    grid.innerHTML = html;

    // 触发 lazyload（如果页面有 lazyload 机制）
    if (window.lazySizes && lazySizes.init) {
      lazySizes.init();
    }
  }

  /**
   * HTML 转义
   */
  function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }
})();
