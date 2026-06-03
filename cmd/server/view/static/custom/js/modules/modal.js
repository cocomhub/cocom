/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  /**
   * Generic modal utility
   * Creates a .modal-wrapper > .modal-inner structure
   */
  window.showCustomModal = function showCustomModal(title, contentHtml, buttonsHtml) {
    var existing = document.querySelector('.modal-wrapper');
    if (existing) closeModal(existing);

    var wrapper = document.createElement('div');
    wrapper.className = 'modal-wrapper fade-slide-in open';

    var inner = document.createElement('div');
    inner.className = 'modal-inner' + (buttonsHtml ? '' : ' modal-compact');

    var titleEl = document.createElement('h1');
    titleEl.textContent = title;
    inner.appendChild(titleEl);

    var content = document.createElement('div');
    content.className = 'contents';
    if (typeof contentHtml === 'string') {
      content.innerHTML = contentHtml;
    } else if (contentHtml instanceof HTMLElement) {
      content.appendChild(contentHtml);
    }
    inner.appendChild(content);

    if (buttonsHtml) {
      var btns = document.createElement('div');
      btns.className = 'buttons';
      if (typeof buttonsHtml === 'string') {
        btns.innerHTML = buttonsHtml;
      } else if (buttonsHtml instanceof HTMLElement) {
        btns.appendChild(buttonsHtml);
      }
      inner.appendChild(btns);
    }

    wrapper.appendChild(inner);
    document.body.appendChild(wrapper);

    // Save triggering focus
    var prevFocus = document.activeElement;

    // Scroll Lock
    var scrollbarWidth = window.innerWidth - document.documentElement.clientWidth;
    document.body.style.setProperty('--scrollbar-width', scrollbarWidth + 'px');
    document.body.classList.add('modal-open');

    // Focus Trap
    var focusableSel = 'input, button, [href], select, textarea, [tabindex]:not([tabindex="-1"])';
    function trapFocus(e) {
      if (e.key !== 'Tab') return;
      var focusable = wrapper.querySelectorAll(focusableSel);
      if (focusable.length === 0) return;
      var first = focusable[0];
      var last = focusable[focusable.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }
    document.addEventListener('keydown', trapFocus);

    // Focus first focusable element
    setTimeout(function() {
      var firstFocusable = wrapper.querySelector(focusableSel);
      if (firstFocusable) firstFocusable.focus();
    }, 50);

    // Click overlay to close
    wrapper.addEventListener('click', function(e) {
      if (e.target === wrapper) closeModal(wrapper);
    });

    // Esc key to close
    var escHandler = function(e) {
      if (e.key === 'Escape') {
        closeModal(wrapper);
      }
    };
    wrapper._escHandler = escHandler;
    document.addEventListener('keydown', escHandler);

    // Save cleanup data
    wrapper._trapFocus = trapFocus;
    wrapper._prevFocus = prevFocus;

    return wrapper;
  };

  window.closeModal = function closeModal(wrapper) {
    if (wrapper && wrapper.parentNode) {
      if (wrapper._escHandler) {
        document.removeEventListener('keydown', wrapper._escHandler);
      }
      if (wrapper._trapFocus) {
        document.removeEventListener('keydown', wrapper._trapFocus);
      }
      // Restore focus
      if (wrapper._prevFocus) {
        wrapper._prevFocus.focus();
      }
      // Close animation
      wrapper.classList.remove('open');
      wrapper.classList.add('fade-slide-out');
      setTimeout(function() {
        if (wrapper && wrapper.parentNode) {
          wrapper.parentNode.removeChild(wrapper);
        }
      }, 150);
    }
    // Check if there are other open modals
    if (!document.querySelector('.modal-wrapper.open')) {
      document.body.classList.remove('modal-open');
      document.body.style.removeProperty('--scrollbar-width');
    }
  };

})();
