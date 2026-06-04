/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Thumbnail zoom control and large mode toggle for gallery detail page.
 */
(function() {
  'use strict';

  window.initThumbnailZoom = function initThumbnailZoom() {
    var slider = document.getElementById('thumbZoomSlider');
    var zoomValue = document.getElementById('zoomValue');
    var zoomInBtn = document.getElementById('zoomInBtn');
    var zoomOutBtn = document.getElementById('zoomOutBtn');
    var zoomResetBtn = document.getElementById('zoomResetBtn');
    var container = document.getElementById('thumbnail-container');
    if (!slider || !container) return;

    // Restore from localStorage
    var saved = localStorage.getItem('thumbZoom');
    if (saved) {
      var v = parseInt(saved, 10);
      if (!isNaN(v) && v >= 60 && v <= 1200) {
        slider.value = v;
      }
    }

    function applyZoom(val) {
      container.style.setProperty('--thumb-w', val + 'px');
      if (zoomValue) zoomValue.textContent = val;
      localStorage.setItem('thumbZoom', String(val));
    }

    // Initial apply
    applyZoom(parseInt(slider.value, 10));

    slider.addEventListener('input', function() {
      applyZoom(parseInt(this.value, 10));
    });

    if (zoomInBtn) {
      zoomInBtn.addEventListener('click', function() {
        var v = Math.min(1200, parseInt(slider.value, 10) + 20);
        slider.value = v;
        applyZoom(v);
      });
    }

    if (zoomOutBtn) {
      zoomOutBtn.addEventListener('click', function() {
        var v = Math.max(60, parseInt(slider.value, 10) - 20);
        slider.value = v;
        applyZoom(v);
      });
    }

    // Reset button
    if (zoomResetBtn) {
      zoomResetBtn.addEventListener('click', function() {
        slider.value = 1200;
        applyZoom(1200);
      });
    }

    // Preset shortcut values
    var presetBtns = document.querySelectorAll('.preset-btn');
    presetBtns.forEach(function(btn) {
      btn.addEventListener('click', function() {
        var val = parseInt(this.getAttribute('data-zoom'), 10);
        if (!isNaN(val) && val >= 60 && val <= 1200) {
          slider.value = val;
          applyZoom(val);
        }
      });
    });
  };

  /**
   * Large mode / thumbnail mode client-side toggle
   * No server request, pure DOM class switching
   */
  window.toggleLargeMode = function toggleLargeMode() {
    var container = document.getElementById('thumbnail-container');
    var zoomSidebar = document.getElementById('zoomSidebar');
    var btn = document.getElementById('sidebarLargeToggle');
    if (!container || !btn) return;

    var isLarge = container.classList.toggle('large-mode');

    if (isLarge) {
      // Enter large mode
      if (zoomSidebar) zoomSidebar.style.display = '';
      container.querySelectorAll('.thumb-container').forEach(function(el) {
        el.classList.remove('thumb-container');
        el.classList.add('thumb-container-large');
      });
      container.querySelectorAll('.thumb-container-large img').forEach(function(img) {
        img.removeAttribute('width');
        img.removeAttribute('height');
        // Switch to original (full-size) image
        var origSrc = img.getAttribute('data-original-src');
        if (origSrc) {
          var thumbSrc = img.getAttribute('data-src');
          img.setAttribute('data-thumb-src', thumbSrc); // save thumbnail src for exit
          img.setAttribute('data-src', origSrc);
          // If already loaded (src is not the placeholder), update src too
          if (img.src && img.src.indexOf('data:image/gif') === -1) {
            img.src = origSrc;
          }
        }
      });
      btn.innerHTML = '<i class="fa fa-compress"></i><span class="label">退出大图</span>';
      localStorage.setItem('largeMode', 'true');
    } else {
      // Exit large mode
      if (zoomSidebar) zoomSidebar.style.display = 'none';
      container.querySelectorAll('.thumb-container-large').forEach(function(el) {
        el.classList.remove('thumb-container-large');
        el.classList.add('thumb-container');
      });
      container.querySelectorAll('.thumb-container img').forEach(function(img) {
        img.setAttribute('width', '200');
        img.setAttribute('height', '282');
        // Restore thumbnail (w=200) src
        var thumbSrc = img.getAttribute('data-thumb-src');
        if (thumbSrc) {
          img.setAttribute('data-src', thumbSrc);
          if (img.src && img.src.indexOf('data:image/gif') === -1) {
            img.src = thumbSrc;
          }
        }
      });
      btn.innerHTML = '<i class="fa fa-expand"></i><span class="label">大图模式</span>';
      localStorage.setItem('largeMode', 'false');
    }
  };

  /**
   * Initialize large mode toggle button state
   */
  window.initLargeModeToggle = function initLargeModeToggle() {
    var btn = document.getElementById('sidebarLargeToggle');
    if (!btn) return;
    var container = document.getElementById('thumbnail-container');
    var zoomSidebar = document.getElementById('zoomSidebar');
    if (!container) return;

    // Sync with server-side render state (?large=true SSR)
    var hasLarge = container.querySelectorAll('.thumb-container-large').length > 0;
    if (hasLarge) {
      container.classList.add('large-mode');
      btn.innerHTML = '<i class="fa fa-compress"></i><span class="label">退出大图</span>';
      if (zoomSidebar) zoomSidebar.style.display = '';
    }

    // Restore preference from localStorage
    var saved = localStorage.getItem('largeMode');
    if (saved === 'false' && hasLarge) {
      window.toggleLargeMode();
    } else if (saved === 'true' && !hasLarge) {
      window.toggleLargeMode();
    }
  };

  /**
   * Mobile toggle zoom sidebar
   */
  window.toggleMobileZoom = function toggleMobileZoom() {
    var sidebar = document.getElementById('zoomSidebar');
    if (sidebar) sidebar.classList.toggle('mobile-open');
  };

})();