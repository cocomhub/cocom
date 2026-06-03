/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Skeleton screen and empty state utilities (stubs for future use).
 */
(function() {
  'use strict';

  window.showSkeleton = function showSkeleton(container) {
    // TODO: implement skeleton screen
  };

  window.hideSkeleton = function hideSkeleton(container) {
    // TODO: implement skeleton screen removal
  };

  window.showEmptyState = function showEmptyState(container, message) {
    if (!container) return;
    container.innerHTML = '<div class="empty-state"><p>' + (message || '暂无数据') + '</p></div>';
  };

})();
