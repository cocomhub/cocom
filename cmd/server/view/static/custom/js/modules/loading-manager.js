/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function () {
  'use strict';

  /**
   * LoadingManager — button-level loading state management
   * Usage: window.LoadingManager.start(btnEl); window.LoadingManager.done(btnEl);
   */
  window.LoadingManager = {
    start: function (btn) {
      if (!btn || btn.dataset.loading) return;
      btn.dataset.loading = 'true';
      btn.dataset.origHTML = btn.innerHTML;
      btn.classList.add('btn-loading');
      btn.disabled = true;
    },
    done: function (btn) {
      if (!btn) return;
      delete btn.dataset.loading;
      btn.classList.remove('btn-loading', 'btn-error');
      btn.disabled = false;
    },
    error: function (btn) {
      if (!btn) return;
      delete btn.dataset.loading;
      btn.classList.remove('btn-loading');
      btn.classList.add('btn-error');
      btn.disabled = false;
      setTimeout(function () {
        if (btn) btn.classList.remove('btn-error');
      }, 400);
    },
  };
})();
