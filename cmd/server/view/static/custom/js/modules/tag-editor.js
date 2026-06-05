/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Tag editor modal for gallery detail page.
 */
(function () {
  'use strict';

  window.openTagEditor = function openTagEditor(cid) {
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/api/comic/getComicInfo');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function () {
      if (xhr.status !== 200) {
        window.showToast('获取漫画信息失败', { type: 'error' });
        return;
      }
      try {
        var resp = JSON.parse(xhr.responseText);
        var info = resp.body;
        var currentTags = info.tags || [];
        buildTagEditorModal(cid, currentTags);
      } catch (e) {
        window.showToast('解析响应失败', { type: 'error' });
      }
    };
    xhr.onerror = function () {
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send('cid=' + encodeURIComponent(cid));
  };

  function esc(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;',
      }[c];
    });
  }

  function buildTagEditorModal(cid, currentTags) {
    var added = [];
    var removed = [];
    var tagTypes = [
      { value: 'parody', label: 'Parodies' },
      { value: 'character', label: 'Characters' },
      { value: 'tag', label: 'Tags' },
      { value: 'artist', label: 'Artists' },
      { value: 'group', label: 'Groups' },
      { value: 'language', label: 'Languages' },
      { value: 'category', label: 'Categories' },
      { value: 'custom', label: 'Customs' },
    ];

    // Dedup key: matches server tagKey logic (id > 0 uses type:id, else type:name)
    function dedupKey(t) {
      return t.type + ':' + (t.id || t.name);
    }

    // Build tag display area
    var tagsContainer = document.createElement('div');
    tagsContainer.style.cssText =
      'margin-bottom: 10px; display: flex; flex-wrap: wrap; gap: 5px;';

    function renderTags() {
      tagsContainer.innerHTML = '';
      var displayTags = [];

      // Existing tags (exclude removed ones)
      currentTags.forEach(function (t) {
        var key = dedupKey(t);
        if (
          !removed.some(function (r) {
            return dedupKey(r) === key;
          })
        ) {
          displayTags.push(t);
        }
      });

      // Newly added tags
      added.forEach(function (t) {
        displayTags.push(t);
      });

      displayTags.forEach(function (t) {
        var chip = document.createElement('span');
        chip.className = 'tag tag-' + (t.id || 0);
        chip.style.cssText =
          'display: inline-flex; align-items: center; margin: 2px; padding: 2px 8px; border-radius: 3px; background: #2a2a2a;';
        chip.innerHTML =
          '<span class="name" style="margin-right:4px">[' +
          esc(t.type) +
          '] ' +
          esc(t.name) +
          '</span>';

        var delBtn = document.createElement('a');
        delBtn.href = 'javascript:;';
        delBtn.textContent = 'x';
        delBtn.style.cssText =
          'color: #e74c3c; text-decoration: none; font-weight: bold;';
        delBtn.onclick = function () {
          var key = dedupKey(t);
          var addedIdx = -1;
          for (var i = 0; i < added.length; i++) {
            if (dedupKey(added[i]) === key) {
              addedIdx = i;
              break;
            }
          }
          if (addedIdx >= 0) {
            added.splice(addedIdx, 1);
          } else {
            removed.push(t);
          }
          renderTags();
        };
        chip.appendChild(delBtn);
        tagsContainer.appendChild(chip);
      });
    }
    renderTags();

    // Add tag form (dual mode: Existing / New)
    var formContainer = document.createElement('div');

    // Tab switcher
    var tabBar = document.createElement('div');
    tabBar.style.cssText =
      'display: flex; gap: 0; margin-bottom: 10px; border-bottom: 2px solid #444;';
    formContainer.appendChild(tabBar);

    var activeMode = 'existing';
    var existingPanel = document.createElement('div');
    var newPanel = document.createElement('div');

    function switchMode(mode) {
      activeMode = mode;
      tabBar.querySelectorAll('.mode-tab').forEach(function (el) {
        el.style.background =
          el.getAttribute('data-mode') === mode ? '#444' : 'transparent';
        el.style.color =
          el.getAttribute('data-mode') === mode ? '#fff' : '#888';
      });
      existingPanel.style.display = mode === 'existing' ? 'flex' : 'none';
      newPanel.style.display = mode === 'new' ? 'flex' : 'none';
    }

    function createTab(label, mode, active) {
      var tab = document.createElement('a');
      tab.href = 'javascript:;';
      tab.className = 'mode-tab';
      tab.setAttribute('data-mode', mode);
      tab.textContent = label;
      tab.style.cssText =
        'padding: 6px 16px; cursor: pointer; font-size: 13px; text-decoration: none;' +
        (active
          ? 'background:#444;color:#fff;'
          : 'background:transparent;color:#888;');
      tab.onclick = function () {
        switchMode(mode);
      };
      tabBar.appendChild(tab);
      return tab;
    }
    createTab('Existing', 'existing', true);
    createTab('New', 'new', false);

    // ----- Existing mode -----
    existingPanel.style.cssText =
      'display: flex; gap: 5px; align-items: center; flex-wrap: wrap;';

    var exTypeSelect = document.createElement('select');
    exTypeSelect.style.cssText =
      'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px;';
    tagTypes.forEach(function (tt) {
      var opt = document.createElement('option');
      opt.value = tt.value;
      opt.textContent = tt.label;
      exTypeSelect.appendChild(opt);
    });
    existingPanel.appendChild(exTypeSelect);

    var searchWrapper = document.createElement('div');
    searchWrapper.style.cssText =
      'position: relative; flex: 1; min-width: 150px;';
    existingPanel.appendChild(searchWrapper);

    var searchInput = document.createElement('input');
    searchInput.type = 'text';
    searchInput.placeholder = 'Search existing tag...';
    searchInput.style.cssText =
      'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px; width: 100%; box-sizing: border-box;';
    searchWrapper.appendChild(searchInput);

    var autocompleteDropdown = document.createElement('div');
    autocompleteDropdown.className = 'tag-autocomplete-dropdown';
    autocompleteDropdown.style.cssText =
      'display: none; position: absolute; top: 100%; left: 0; right: 0; ' +
      'background: #333; border: 1px solid #555; border-radius: 3px; ' +
      'max-height: 200px; overflow-y: auto; z-index: 1000;';
    searchWrapper.appendChild(autocompleteDropdown);

    function hideAutocomplete() {
      autocompleteDropdown.style.display = 'none';
      autocompleteDropdown.innerHTML = '';
    }

    function renderAutocomplete(tags) {
      autocompleteDropdown.innerHTML = '';
      if (tags.length === 0) {
        autocompleteDropdown.style.display = 'none';
        return;
      }
      autocompleteDropdown.style.display = 'block';

      tags.forEach(function (t) {
        var item = document.createElement('div');
        item.style.cssText =
          'padding: 6px 10px; cursor: pointer; border-bottom: 1px solid #444; ' +
          'display: flex; justify-content: space-between;';
        item.innerHTML =
          '<span>[' +
          esc(t.type) +
          '] ' +
          esc(t.name) +
          '</span>' +
          '<span style="color:#888;font-size:12px;">' +
          esc(t.count) +
          '</span>';
        item.onclick = function () {
          var key = dedupKey(t);
          var exists =
            added.some(function (a) {
              return dedupKey(a) === key;
            }) ||
            currentTags.some(function (ct) {
              return (
                dedupKey(ct) === key &&
                !removed.some(function (r) {
                  return dedupKey(r) === dedupKey(ct);
                })
              );
            });
          if (exists) {
            window.showToast('该 tag 已存在', { type: 'info' });
          } else {
            added.push(t);
            renderTags();
            window.showToast('已添加: [' + t.type + '] ' + t.name, {
              type: 'success',
            });
          }
          searchInput.value = '';
          hideAutocomplete();
        };
        item.onmouseenter = function () {
          item.style.background = '#444';
        };
        item.onmouseleave = function () {
          item.style.background = 'transparent';
        };
        autocompleteDropdown.appendChild(item);
      });
    }

    var searchTimeout = null;
    searchInput.addEventListener('input', function () {
      if (searchTimeout) clearTimeout(searchTimeout);
      var q = this.value.trim();
      if (!q) {
        hideAutocomplete();
        return;
      }
      var type = exTypeSelect.value;
      searchTimeout = setTimeout(function () {
        var xhr = new XMLHttpRequest();
        xhr.withCredentials = true;
        xhr.open(
          'GET',
          '/api/comic/tags/search?type=' +
            encodeURIComponent(type) +
            '&q=' +
            encodeURIComponent(q) +
            '&limit=20',
        );
        xhr.onload = function () {
          if (xhr.status !== 200) return;
          try {
            var resp = JSON.parse(xhr.responseText);
            renderAutocomplete((resp.body && resp.body.tags) || []);
          } catch (e) {}
        };
        xhr.send();
      }, 300);
    });

    searchInput.addEventListener('blur', function () {
      setTimeout(hideAutocomplete, 150);
    });

    searchInput.addEventListener('focus', function () {
      if (this.value.trim()) {
        var event = new Event('input');
        this.dispatchEvent(event);
      }
    });

    // Bind keyboard navigation
    window.bindAutocompleteKeys(searchInput, autocompleteDropdown, function () {
      var firstItem = autocompleteDropdown.querySelector('div');
      if (firstItem) firstItem.click();
    });

    // Click outside to close
    document.addEventListener('click', function (e) {
      if (
        !e.target.closest('.tag-autocomplete-dropdown') &&
        !e.target.closest('#editTagsSearchWrapper')
      ) {
        // no-op, keep existing logic
      }
    });

    formContainer.appendChild(existingPanel);

    // ----- New mode -----
    newPanel.style.cssText =
      'display: none; gap: 5px; align-items: center; flex-wrap: wrap;';

    var newTypeSelect = document.createElement('select');
    newTypeSelect.style.cssText =
      'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px;';
    tagTypes.forEach(function (tt) {
      var opt = document.createElement('option');
      opt.value = tt.value;
      opt.textContent = tt.label;
      newTypeSelect.appendChild(opt);
    });
    newPanel.appendChild(newTypeSelect);

    var nameInput = document.createElement('input');
    nameInput.type = 'text';
    nameInput.placeholder = 'New tag name';
    nameInput.style.cssText =
      'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px; flex: 1; min-width: 150px;';
    newPanel.appendChild(nameInput);

    nameInput.addEventListener('keydown', function (e) {
      if (e.key === 'Enter') {
        e.preventDefault();
        addBtn.click();
      }
    });

    var addBtn = document.createElement('a');
    addBtn.href = 'javascript:;';
    addBtn.className = 'btn btn-secondary';
    addBtn.textContent = 'Add';
    addBtn.style.cssText = 'padding: 4px 12px;';
    addBtn.onclick = function () {
      var name = nameInput.value.trim();
      if (!name) {
        window.showToast('请输入 tag 名称', { type: 'error' });
        return;
      }
      var type = newTypeSelect.value;
      var urlName = encodeURIComponent(name.toLowerCase().replace(/\s+/g, '-'));
      var url = '/' + type + '/' + urlName + '/';
      var newTag = { id: 0, name: name, type: type, url: url, count: 1 };
      var key = dedupKey(newTag);

      if (
        added.some(function (a) {
          return dedupKey(a) === key;
        })
      ) {
        window.showToast('该 tag 已在添加列表中', { type: 'info' });
        return;
      }
      if (
        currentTags.some(function (ct) {
          return (
            dedupKey(ct) === key &&
            !removed.some(function (r) {
              return dedupKey(r) === dedupKey(ct);
            })
          );
        })
      ) {
        window.showToast('该 tag 已存在', { type: 'info' });
        return;
      }

      added.push(newTag);
      nameInput.value = '';
      renderTags();
      window.showToast('已添加: ' + name, { type: 'success' });
    };
    newPanel.appendChild(addBtn);

    formContainer.appendChild(newPanel);

    // Modal content
    var modalContent = document.createElement('div');
    modalContent.appendChild(tagsContainer);
    modalContent.appendChild(formContainer);

    var wrapper = window.showCustomModal(
      'Edit Tags',
      modalContent,
      '<a href="javascript:;" class="btn btn-secondary" onclick="window.closeModal(this.closest(\'.modal-wrapper\'))">Cancel</a>' +
        '<a href="javascript:;" class="btn btn-primary" id="saveTagsBtn">Save</a>',
    );

    // Save button logic
    var saveBtn = wrapper.querySelector('#saveTagsBtn');
    if (saveBtn) {
      saveBtn.onclick = function () {
        if (added.length === 0 && removed.length === 0) {
          window.showToast('没有变更', { type: 'info' });
          return;
        }
        window.LoadingManager.start(saveBtn);
        var payload = JSON.stringify({
          cid: cid,
          added: added,
          removed: removed,
        });
        var saveXhr = new XMLHttpRequest();
        saveXhr.withCredentials = true;
        saveXhr.open('POST', '/api/comic/tags/update');
        saveXhr.setRequestHeader('Content-Type', 'application/json');
        saveXhr.onload = function () {
          window.LoadingManager.done(saveBtn);
          if (saveXhr.status >= 200 && saveXhr.status < 300) {
            window.showToast('Tags 已更新', { type: 'success' });
            window.closeModal(wrapper);
            // No refresh: use getComicInfo to refresh tag area
            window.OptimisticUpdater.refreshContainer(
              '/api/comic/getComicInfo',
              '#tags',
              function (container, data) {
                if (data.body && data.body.tags) {
                  window.rebuildTagsSection(data.body.tags);
                }
              },
            );
          } else {
            try {
              var r = JSON.parse(saveXhr.responseText);
              window.showToast((r.head && r.head.msg) || '保存失败', {
                type: 'error',
              });
            } catch (e) {
              window.showToast('保存失败: ' + saveXhr.status, {
                type: 'error',
              });
            }
          }
        };
        saveXhr.onerror = function () {
          window.LoadingManager.done(saveBtn);
          window.showToast('网络错误', { type: 'error' });
        };
        saveXhr.send(payload);
      };
    }
  }
})();
