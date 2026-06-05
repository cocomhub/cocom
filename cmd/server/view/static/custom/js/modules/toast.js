/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function () {
  'use strict';

  window.showToast = function showToast(message, opts) {
    var options = opts || {};
    var type = options.type || 'info';
    var duration =
      typeof options.duration === 'number' ? options.duration : 5000;
    var dismissible = options.dismissible !== false;
    if (typeof message === 'object' && message !== null) {
      try {
        message = JSON.stringify(message);
      } catch (e) {}
    }
    var icons = { success: '✅', error: '❌', info: 'ℹ️', warning: '⚠️' };
    var icon = icons[type] || '';
    var typeClass = 'alert-info';
    if (type === 'success') typeClass = 'alert-success';
    else if (type === 'error') typeClass = 'alert-danger';
    else if (type === 'warning') typeClass = 'alert-warning';
    var container = document.getElementById('messages');
    if (!container) return;
    var alert = document.createElement('div');
    alert.className = 'alert ' + typeClass + ' fade-slide-in open';
    alert.textContent = icon + ' ' + message;
    if (dismissible) {
      alert.style.cursor = 'pointer';
      alert.addEventListener('click', function () {
        if (alert && alert.parentNode) {
          alert.parentNode.removeChild(alert);
        }
      });
    }
    container.appendChild(alert);
    if (duration > 0) {
      setTimeout(function () {
        if (alert && alert.parentNode) {
          alert.parentNode.removeChild(alert);
        }
      }, duration);
    }
  };

  window.showProgressToast = function showProgressToast(message, percent) {
    var container = document.getElementById('messages');
    if (!container) return;
    var existing = document.getElementById('progress-toast');
    if (existing) {
      var bar = existing.querySelector('.progress-bar');
      if (bar) bar.style.width = Math.min(100, Math.max(0, percent || 0)) + '%';
      var msg = existing.querySelector('.progress-msg');
      if (msg) msg.textContent = message;
      return;
    }
    var toast = document.createElement('div');
    toast.id = 'progress-toast';
    toast.className = 'alert alert-info fade-slide-in open';
    toast.style.cssText = 'padding: 8px 12px;';
    toast.innerHTML =
      '<div class="progress-msg">' +
      message +
      '</div>' +
      '<div style="margin-top:4px;height:4px;background:#444;border-radius:2px;overflow:hidden;">' +
      '<div class="progress-bar" style="width:' +
      Math.min(100, Math.max(0, percent || 0)) +
      '%;height:100%;background:#4CAF50;transition:width 0.3s;"></div></div>';
    container.appendChild(toast);
  };
})();
