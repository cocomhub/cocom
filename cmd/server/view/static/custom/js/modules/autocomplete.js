/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function () {
  'use strict';

  /**
   * Autocomplete dropdown keyboard navigation
   * @param {HTMLElement} dropdown - dropdown container
   * @param {Function} onSelect - selection callback, receives current highlight index
   * @returns {Function} destroy function
   */
  window.enableAutocompleteKeyboardNav = function enableAutocompleteKeyboardNav(
    dropdown,
    onSelect,
  ) {
    var selectedIdx = -1;

    function getItems() {
      return dropdown.querySelectorAll('div');
    }

    function highlight(idx) {
      var items = getItems();
      items.forEach(function (el, i) {
        el.classList.remove('keyboard-selected');
        el.style.background = i === idx ? '#444' : 'transparent';
      });
    }

    function handler(e) {
      var items = getItems();
      if (items.length === 0) return;
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        selectedIdx = (selectedIdx + 1) % items.length;
        highlight(selectedIdx);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        selectedIdx = (selectedIdx - 1 + items.length) % items.length;
        highlight(selectedIdx);
      } else if (e.key === 'Enter' && selectedIdx >= 0 && items[selectedIdx]) {
        e.preventDefault();
        items[selectedIdx].click();
      } else if (e.key === 'Escape') {
        e.preventDefault();
        dropdown.style.display = 'none';
        selectedIdx = -1;
      }
    }

    dropdown.addEventListener('keydown', handler);
    return function destroy() {
      dropdown.removeEventListener('keydown', handler);
    };
  };

  // Bind keyboard navigation to input (delegates input keydown event to dropdown)
  window.bindAutocompleteKeys = function bindAutocompleteKeys(
    input,
    dropdown,
    onEnter,
  ) {
    input.addEventListener('keydown', function (e) {
      if (dropdown.style.display === 'none' || !dropdown.children.length) {
        if (e.key === 'Enter' && onEnter) {
          e.preventDefault();
          onEnter();
        }
        return;
      }
      var items = dropdown.querySelectorAll('div');
      var selected = dropdown.querySelector('.keyboard-selected');
      var idx = -1;
      if (selected) {
        for (var i = 0; i < items.length; i++) {
          if (items[i] === selected) {
            idx = i;
            break;
          }
        }
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        idx = (idx + 1) % items.length;
        items.forEach(function (el, i) {
          el.classList.toggle('keyboard-selected', i === idx);
          el.style.background = i === idx ? '#444' : 'transparent';
        });
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        idx = (idx - 1 + items.length) % items.length;
        items.forEach(function (el, i) {
          el.classList.toggle('keyboard-selected', i === idx);
          el.style.background = i === idx ? '#444' : 'transparent';
        });
      } else if (e.key === 'Enter' && idx >= 0 && items[idx]) {
        e.preventDefault();
        items[idx].click();
      } else if (e.key === 'Escape') {
        e.preventDefault();
        dropdown.style.display = 'none';
      }
    });
  };
})();
