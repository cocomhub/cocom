/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Navigation helpers and keyboard shortcuts.
 */
(function () {
  'use strict';

  window.jumpToPage = function jumpToPage(input, baseUrl, query) {
    var page = parseInt(input.value, 10);
    var max = parseInt(input.max, 10);
    if (isNaN(page) || page < 1 || page > max) {
      window.showToast('页码应在 1 ~ ' + max + ' 之间', { type: 'warning' });
      return;
    }
    var url = baseUrl + '?page=' + page;
    if (query) url += '&q=' + encodeURIComponent(query);
    window.location.href = url;
  };

  window.jumpToTagPage = function jumpToTagPage(input, baseUrl, sortType) {
    var page = parseInt(input.value, 10);
    var max = parseInt(input.max, 10);
    if (isNaN(page) || page < 1 || page > max) {
      window.showToast('页码应在 1 ~ ' + max + ' 之间', { type: 'warning' });
      return;
    }
    var url = baseUrl + '?page=' + page;
    if (sortType === 1) url += '&sortType=popular';
    window.location.href = url;
  };

  window.initKeyboardShortcuts = function initKeyboardShortcuts() {
    // Gallery page: left/right arrow for previous/next page
    var galleryNav = document.querySelector('.gallery-nav');
    if (galleryNav) {
      var prevLink = galleryNav.querySelector('a[rel="prev"]');
      var nextLink = galleryNav.querySelector('a[rel="next"]');
      if (prevLink || nextLink) {
        document.addEventListener('keydown', function navHandler(e) {
          if (
            e.target.tagName === 'INPUT' ||
            e.target.tagName === 'TEXTAREA' ||
            e.target.isContentEditable
          )
            return;
          if (e.key === 'ArrowLeft' && prevLink) {
            e.preventDefault();
            window.location.href = prevLink.href;
          } else if (e.key === 'ArrowRight' && nextLink) {
            e.preventDefault();
            window.location.href = nextLink.href;
          }
        });
      }
    }
  };
})();
