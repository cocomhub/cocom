/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Related tags loader and relation modal.
 *
 * Note: openTagRelationManager and buildRelationModal are also declared in
 * tag_relation.js. This module provides loadRelatedTags. The relation
 * management functions are kept in tag_relation.js to preserve backward compat.
 */
(function () {
  'use strict';

  window.loadRelatedTags = function loadRelatedTags(type, name) {
    var container = document.getElementById('related-tags-content');
    if (!container) return;

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(
      'GET',
      '/api/comic/tags/related?type=' +
        encodeURIComponent(type) +
        '&name=' +
        encodeURIComponent(name) +
        '&limit=30',
    );
    xhr.onload = function () {
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

        // Group by type
        var groups = {};
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

        var esc = function (s) {
          return String(s).replace(/[&<>"']/g, function (c) {
            return {
              '&': '&amp;',
              '<': '&lt;',
              '>': '&gt;',
              '"': '&quot;',
              "'": '&#39;',
            }[c];
          });
        };
        var html = '';
        Object.keys(groups)
          .sort()
          .forEach(function (type) {
            var groupTags = groups[type];
            html +=
              '<div class="tag-container field-name"><strong>' +
              esc(typeLabels[type] || type) +
              ':</strong> <span class="tags">';
            groupTags.forEach(function (t) {
              var likeClass = t.like ? ' tag-like' : '';
              html +=
                '<a href="/tag' +
                esc(t.url) +
                '" class="tag tag-' +
                (t.id || 0) +
                likeClass +
                '">' +
                '<span class="name">' +
                esc(t.name) +
                '</span>' +
                '<span class="count">' +
                esc(t.count) +
                '</span></a>';
            });
            html += '</span></div>';
          });
        container.innerHTML = html;
      } catch (e) {
        container.innerHTML = '<p>解析失败</p>';
      }
    };
    xhr.onerror = function () {
      container.innerHTML = '<p>网络错误</p>';
    };
    xhr.send();
  };
})();
