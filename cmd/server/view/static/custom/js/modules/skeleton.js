/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Skeleton screen and empty state utilities.
 */
(function() {
  'use strict';

  /**
   * Create a single skeleton card element.
   * @returns {HTMLElement}
   */
  function createSkeletonCard() {
    var card = document.createElement('div');
    card.className = 'skeleton-card';
    card.innerHTML = '<div class="skeleton-thumb"></div><div class="skeleton-line"></div>';
    return card;
  }

  /**
   * Show skeleton screen placeholders inside a container.
   * Replaces existing content with skeleton cards.
   * @param {string|HTMLElement} container - Element ID or DOM element.
   * @param {number} [count=12] - Number of skeleton cards to show.
   */
  window.showSkeleton = function showSkeleton(container, count) {
    var el = (typeof container === 'string') ? document.getElementById(container) : container;
    if (!el) return;
    count = count || 12;
    el.innerHTML = '';
    for (var i = 0; i < count; i++) {
      el.appendChild(createSkeletonCard());
    }
  };

  /**
   * Hide skeleton screen placeholders by emptying the container.
   * @param {string|HTMLElement} container - Element ID or DOM element.
   */
  window.hideSkeleton = function hideSkeleton(container) {
    var el = (typeof container === 'string') ? document.getElementById(container) : container;
    if (!el) return;
    el.innerHTML = '';
  };

  /**
   * Show an empty state message inside a container.
   * @param {HTMLElement} container - DOM element to populate.
   * @param {string} [message='暂无数据'] - Message to display.
   */
  window.showEmptyState = function showEmptyState(container, message) {
    if (!container) return;
    container.innerHTML = '<div class="empty-state"><i class="fa fa-inbox"></i><p>' + (message || '暂无数据') + '</p></div>';
  };

})();
