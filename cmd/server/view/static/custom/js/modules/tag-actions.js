/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Tag page action functions: toggleLikeTag, rebuildTagsSection.
 */
(function () {
  'use strict';

  window.toggleLikeTag = function toggleLikeTag(type, name, id) {
    const btn = document.getElementById('toggleLikeTag');
    if (!btn) return;
    const liked = btn.classList.contains('btn-primary');
    const xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/likeTag');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function () {
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
    xhr.onerror = function () {
      console.error('likeTag request network error');
    };
    var params = 'type=' + encodeURIComponent(type);
    if (id && id > 0) {
      params += '&id=' + encodeURIComponent(id);
    } else if (name) {
      params += '&name=' + encodeURIComponent(name);
    }
    xhr.send(params);
  };

  /**
   * Rebuild the tag list section (no refresh)
   */
  function esc(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;',
      }[c];
    });
  }

  window.rebuildTagsSection = function rebuildTagsSection(tags) {
    var container = document.querySelector('#tags');
    if (!container) return;
    var groups = {};
    var typeOrder = [
      'parody',
      'character',
      'tag',
      'artist',
      'group',
      'language',
      'category',
      'custom',
    ];
    var typeLabels = {
      parody: 'Parodies',
      character: 'Characters',
      tag: 'Tags',
      artist: 'Artists',
      group: 'Groups',
      language: 'Languages',
      category: 'Categories',
      custom: 'Customs',
    };
    tags.forEach(function (t) {
      if (!groups[t.type]) groups[t.type] = [];
      groups[t.type].push(t);
    });
    var html = '';
    typeOrder.forEach(function (type) {
      var list = groups[type];
      if (!list || list.length === 0) return;
      html +=
        '<div class="tag-container field-name">' +
        esc(typeLabels[type]) +
        ': <span class="tags">';
      list.forEach(function (t) {
        html +=
          '<a href="/tag/' +
          encodeURIComponent(t.type) +
          '/' +
          encodeURIComponent(t.name.toLowerCase().replace(/\s+/g, '-')) +
          '/" class="tag tag-' +
          (t.id || 0) +
          '">' +
          '<span class="name">' +
          esc(t.name) +
          '</span><span class="count">' +
          esc(t.count || 1) +
          '</span></a>';
      });
      html += '</span></div>';
    });
    container.innerHTML = html;
  };
})();
