/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Search autocomplete dropdown — mixed comic titles + tags.
 */
(function() {
  'use strict';

  function esc(s) {
    return String(s).replace(/[&<>"']/g, function(c) {
      return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];
    });
  }

  window.initSearchAutocomplete = function initSearchAutocomplete() {
    var input = document.querySelector('input[type="search"]');
    if (!input) return;

    var dropdown = document.createElement('div');
    dropdown.className = 'search-autocomplete-dropdown';
    dropdown.style.display = 'none';
    input.parentNode.appendChild(dropdown);

    var timeout = null;

    function hideDropdown() {
      dropdown.style.display = 'none';
      dropdown.innerHTML = '';
    }

    function handleResponse(comics, tags) {
      dropdown.innerHTML = '';

      var html = '';

      // Comics section
      if (comics && comics.length > 0) {
        html += '<div class="autocomplete-section"><div class="autocomplete-section-title">漫画</div>';
        comics.forEach(function(c) {
          html += '<div class="autocomplete-item autocomplete-comic" data-cid="' + c.cid + '">' +
            '<i class="fa fa-book" style="margin-right:6px;color:#888;"></i>' +
            '<span class="autocomplete-title">' + esc(c.title) + '</span>' +
            '<span class="autocomplete-cid">#' + c.cid + '</span></div>';
        });
        html += '</div>';
      }

      // Tags section
      if (tags && tags.length > 0) {
        html += '<div class="autocomplete-section"><div class="autocomplete-section-title">标签</div>';
        tags.forEach(function(t) {
          html += '<div class="autocomplete-item autocomplete-tag" data-type="' + esc(t.type) + '" data-name="' + esc(t.name) + '" data-url="' + esc(t.url) + '">' +
            '<i class="fa fa-tag" style="margin-right:6px;color:#888;"></i>' +
            '<span class="autocomplete-tag-type">[' + esc(t.type) + ']</span> ' +
            '<span class="autocomplete-tag-name">' + esc(t.name) + '</span>' +
            '<span class="autocomplete-tag-count">' + t.count + '</span></div>';
        });
        html += '</div>';
      }

      if (!html) {
        hideDropdown();
        return;
      }

      dropdown.innerHTML = html;
      dropdown.style.display = 'block';

      // Bind item click
      dropdown.querySelectorAll('.autocomplete-item').forEach(function(item) {
        item.addEventListener('click', function() {
          if (item.classList.contains('autocomplete-comic')) {
            var cid = item.getAttribute('data-cid');
            window.location.href = '/g/' + cid + '/';
          } else if (item.classList.contains('autocomplete-tag')) {
            var url = item.getAttribute('data-url');
            window.location.href = '/tag' + url;
          }
        });
      });
    }

    input.addEventListener('input', function() {
      var q = this.value.trim();
      if (timeout) clearTimeout(timeout);
      if (q.length < 2) {
        hideDropdown();
        return;
      }
      timeout = setTimeout(function() {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', '/api/search/autocomplete?q=' + encodeURIComponent(q) + '&limit=5');
        xhr.onload = function() {
          if (xhr.status !== 200) return;
          try {
            var resp = JSON.parse(xhr.responseText);
            handleResponse(resp.body.comics, resp.body.tags);
          } catch (e) { /* ignore parse errors */ }
        };
        xhr.send();
      }, 200);
    });

    // Keyboard navigation (reuse bindAutocompleteKeys)
    if (typeof window.bindAutocompleteKeys === 'function') {
      window.bindAutocompleteKeys(input, dropdown, function() {
        // Default Enter: submit form if no selection
        input.closest('form').submit();
      });
    }

    // Close on blur (with delay for click)
    input.addEventListener('blur', function() {
      setTimeout(hideDropdown, 200);
    });

    // Close on Escape
    input.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') {
        hideDropdown();
      }
    });
  };
})();