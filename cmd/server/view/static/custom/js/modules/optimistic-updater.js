/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function () {
  'use strict';

  /**
   * OptimisticUpdater — optimistic update + partial refresh utility
   */
  window.OptimisticUpdater = {
    // Optimistic Toggle: immediately switch class, rollback on failure
    optimisticToggle: function (btn, activeClass, inactiveClass) {
      var wasActive = btn.classList.contains(activeClass);
      var rollbackState = {
        activeClass: activeClass,
        inactiveClass: inactiveClass,
        wasActive: wasActive,
      };
      btn.classList.remove(activeClass, inactiveClass);
      btn.classList.add(wasActive ? inactiveClass : activeClass);
      return {
        rollback: function () {
          btn.classList.remove(activeClass, inactiveClass);
          btn.classList.add(
            rollbackState.wasActive ? activeClass : inactiveClass,
          );
        },
        wasActive: wasActive,
      };
    },

    // Partial refresh: fetch data and execute render function to replace container content
    refreshContainer: function (url, containerSelector, renderFn) {
      var container = document.querySelector(containerSelector);
      if (!container)
        return Promise.reject('Container not found: ' + containerSelector);
      return fetch(url, { credentials: 'include' })
        .then(function (r) {
          if (!r.ok) throw new Error('HTTP ' + r.status);
          return r.json();
        })
        .then(function (data) {
          if (renderFn && typeof renderFn === 'function') {
            renderFn(container, data);
          }
          return data;
        })
        .catch(function (err) {
          window.showToast('刷新失败: ' + err.message, { type: 'error' });
        });
    },
  };
})();
