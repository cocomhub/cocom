/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

/**
 * Tag 关系管理弹窗 — 打开关系管理器
 */
function openTagRelationManager(type, name, id) {
  var xhr = new XMLHttpRequest();
  xhr.withCredentials = true;
  xhr.open(
    'GET',
    '/api/comic/tags/relation?type=' +
      encodeURIComponent(type) +
      '&name=' +
      encodeURIComponent(name) +
      '&id=' +
      encodeURIComponent(id),
  );
  xhr.onload = function () {
    if (xhr.status !== 200) {
      window.showToast('获取关系列表失败', { type: 'error' });
      return;
    }
    try {
      var resp = JSON.parse(xhr.responseText);
      var groups = (resp.body && resp.body.groups) || [];
      buildRelationModal(type, name, id, groups);
    } catch (e) {
      window.showToast('解析失败', { type: 'error' });
    }
  };
  xhr.onerror = function () {
    window.showToast('网络错误', { type: 'error' });
  };
  xhr.send();
}

function buildRelationModal(srcType, srcName, srcId, groups) {
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

  var wrapper;
  var content = document.createElement('div');

  // Add new relation section
  var addSection = document.createElement('div');
  addSection.style.cssText =
    'margin-bottom: 15px; padding-bottom: 15px; border-bottom: 1px solid #444;';
  var addTitle = document.createElement('h4');
  addTitle.style.cssText = 'margin: 0 0 8px 0;';
  addTitle.textContent = 'Add New Relation Group';
  addSection.appendChild(addTitle);

  var groupChips = document.createElement('div');
  groupChips.style.cssText =
    'display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 8px; min-height: 30px;';
  var hint = document.createElement('span');
  hint.style.cssText = 'color: #666; font-size: 13px;';
  hint.textContent = 'Select at least 2 tags to form a relation group.';
  groupChips.appendChild(hint);
  addSection.appendChild(groupChips);

  // ===== Tab switcher: Existing / New =====
  var tabBar = document.createElement('div');
  tabBar.style.cssText =
    'display: flex; gap: 0; margin-bottom: 8px; border-bottom: 2px solid #444;';
  addSection.appendChild(tabBar);

  var activeMode = 'existing';
  var existingPanel = document.createElement('div');
  var newPanel = document.createElement('div');

  function relSwitchMode(mode) {
    activeMode = mode;
    tabBar.querySelectorAll('.rel-mode-tab').forEach(function (el) {
      el.style.background =
        el.getAttribute('data-mode') === mode ? '#444' : 'transparent';
      el.style.color = el.getAttribute('data-mode') === mode ? '#fff' : '#888';
    });
    existingPanel.style.display = mode === 'existing' ? 'flex' : 'none';
    newPanel.style.display = mode === 'new' ? 'flex' : 'none';
  }

  function relCreateTab(label, mode, active) {
    var tab = document.createElement('a');
    tab.href = 'javascript:;';
    tab.className = 'rel-mode-tab';
    tab.setAttribute('data-mode', mode);
    tab.textContent = label;
    tab.style.cssText =
      'padding: 6px 16px; cursor: pointer; font-size: 13px; text-decoration: none;' +
      (active
        ? 'background:#444;color:#fff;'
        : 'background:transparent;color:#888;');
    tab.onclick = function () {
      relSwitchMode(mode);
    };
    tabBar.appendChild(tab);
  }
  relCreateTab('Existing', 'existing', true);
  relCreateTab('New', 'new', false);

  // ----- Existing mode -----
  existingPanel.style.cssText =
    'display: flex; gap: 5px; align-items: center; flex-wrap: wrap;';
  addSection.appendChild(existingPanel);

  var relExTypeSelect = document.createElement('select');
  relExTypeSelect.style.cssText =
    'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px;';
  tagTypes.forEach(function (tt) {
    var opt = document.createElement('option');
    opt.value = tt.value;
    opt.textContent = tt.label;
    relExTypeSelect.appendChild(opt);
  });
  existingPanel.appendChild(relExTypeSelect);

  var relSearchWrap = document.createElement('div');
  relSearchWrap.style.cssText =
    'position: relative; flex: 1; min-width: 120px;';
  existingPanel.appendChild(relSearchWrap);

  var relSearchInput = document.createElement('input');
  relSearchInput.type = 'text';
  relSearchInput.placeholder = 'Search existing tag...';
  relSearchInput.style.cssText =
    'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px; width: 100%; box-sizing: border-box;';
  relSearchWrap.appendChild(relSearchInput);

  var relDropdown = document.createElement('div');
  relDropdown.style.cssText =
    'display: none; position: absolute; top: 100%; left: 0; right: 0; ' +
    'background: #333; border: 1px solid #555; border-radius: 3px; ' +
    'max-height: 200px; overflow-y: auto; z-index: 1000;';
  relSearchWrap.appendChild(relDropdown);

  function relHideDropdown() {
    relDropdown.style.display = 'none';
    relDropdown.innerHTML = '';
  }

  function relRenderDropdown(tags) {
    relDropdown.innerHTML = '';
    if (tags.length === 0) {
      relDropdown.style.display = 'none';
      return;
    }
    relDropdown.style.display = 'block';
    tags.forEach(function (t) {
      var item = document.createElement('div');
      item.style.cssText =
        'padding: 6px 10px; cursor: pointer; border-bottom: 1px solid #444; display: flex; justify-content: space-between;';
      item.innerHTML =
        '<span>[' +
        t.type +
        '] ' +
        t.name +
        '</span><span style="color:#888;font-size:12px;">' +
        t.count +
        '</span>';
      item.onclick = function () {
        var chips = groupChips.querySelectorAll('.relation-chips');
        var dup = false;
        for (var i = 0; i < chips.length; i++) {
          if (
            chips[i].getAttribute('data-type') === t.type &&
            (chips[i].getAttribute('data-id') == t.id ||
              chips[i].getAttribute('data-name') === t.name)
          ) {
            dup = true;
            break;
          }
        }
        if (t.type === srcType && t.name === srcName) {
          window.showToast('不能将当前 tag 添加到组中', { type: 'error' });
          return;
        }
        if (dup) {
          window.showToast('该 tag 已在组中', { type: 'info' });
          relHideDropdown();
          return;
        }

        var chip = document.createElement('span');
        chip.className = 'tag relation-chips';
        chip.setAttribute('data-type', t.type);
        chip.setAttribute('data-name', t.name);
        chip.setAttribute('data-id', t.id || 0);
        chip.setAttribute(
          'data-url',
          t.url ||
            '/' +
              t.type +
              '/' +
              encodeURIComponent(t.name.toLowerCase().replace(/\s+/g, '-')) +
              '/',
        );
        chip.style.cssText =
          'display: inline-flex; align-items: center; margin: 2px; padding: 2px 8px; border-radius: 3px; background: #2a2a2a;';
        chip.innerHTML =
          '<span class="name" style="margin-right:4px">[' +
          t.type +
          '] ' +
          t.name +
          '</span>';
        var delBtn = document.createElement('a');
        delBtn.href = 'javascript:;';
        delBtn.textContent = 'x';
        delBtn.style.cssText =
          'color: #e74c3c; text-decoration: none; font-weight: bold;';
        delBtn.onclick = function () {
          chip.remove();
          relUpdateBtn();
        };
        chip.appendChild(delBtn);
        groupChips.appendChild(chip);
        var h = groupChips.querySelector('span[style]');
        if (h && h.style.color === 'rgb(102, 102, 102)') h.remove();
        relSearchInput.value = '';
        relHideDropdown();
        relUpdateBtn();
        window.showToast('已添加: [' + t.type + '] ' + t.name, {
          type: 'success',
        });
      };
      item.onmouseenter = function () {
        item.style.background = '#444';
      };
      item.onmouseleave = function () {
        item.style.background = 'transparent';
      };
      relDropdown.appendChild(item);
    });
  }

  var relSearchTimer = null;
  relSearchInput.addEventListener('input', function () {
    if (relSearchTimer) clearTimeout(relSearchTimer);
    var q = this.value.trim();
    if (!q) {
      relHideDropdown();
      return;
    }
    var type = relExTypeSelect.value;
    relSearchTimer = setTimeout(function () {
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
          relRenderDropdown((resp.body && resp.body.tags) || []);
        } catch (e) {}
      };
      xhr.send();
    }, 300);
  });
  relSearchInput.addEventListener('blur', function () {
    setTimeout(relHideDropdown, 150);
  });

  // 绑定键盘导航
  window.bindAutocompleteKeys(relSearchInput, relDropdown, function () {
    var firstItem = relDropdown.querySelector('div');
    if (firstItem) firstItem.click();
  });

  // ----- New mode -----
  newPanel.style.cssText =
    'display: none; gap: 5px; align-items: center; flex-wrap: wrap;';
  addSection.appendChild(newPanel);

  var relNewTypeSelect = document.createElement('select');
  relNewTypeSelect.style.cssText =
    'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px;';
  tagTypes.forEach(function (tt) {
    var opt = document.createElement('option');
    opt.value = tt.value;
    opt.textContent = tt.label;
    relNewTypeSelect.appendChild(opt);
  });
  newPanel.appendChild(relNewTypeSelect);

  var relNewNameInput = document.createElement('input');
  relNewNameInput.type = 'text';
  relNewNameInput.placeholder = 'New tag name';
  relNewNameInput.style.cssText =
    'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px; flex: 1; min-width: 120px;';
  newPanel.appendChild(relNewNameInput);

  relNewNameInput.addEventListener('keydown', function (e) {
    if (e.key === 'Enter') {
      e.preventDefault();
      relAddNewBtn.click();
    }
  });

  var relAddNewBtn = document.createElement('a');
  relAddNewBtn.href = 'javascript:;';
  relAddNewBtn.className = 'btn btn-secondary';
  relAddNewBtn.textContent = 'Add to Group';
  relAddNewBtn.style.cssText = 'padding: 4px 10px;';
  relAddNewBtn.onclick = function () {
    var tName = relNewNameInput.value.trim();
    var tType = relNewTypeSelect.value;
    if (!tName) {
      window.showToast('请输入 tag 名称', { type: 'error' });
      return;
    }
    if (tType === srcType && tName === srcName) {
      window.showToast('不能将当前 tag 添加到组中', { type: 'error' });
      return;
    }
    var existing = groupChips.querySelectorAll('.relation-chips');
    for (var i = 0; i < existing.length; i++) {
      if (
        existing[i].getAttribute('data-type') === tType &&
        existing[i].getAttribute('data-name') === tName
      ) {
        window.showToast('该 tag 已在组中', { type: 'info' });
        return;
      }
    }
    var chip = document.createElement('span');
    chip.className = 'tag relation-chips';
    chip.setAttribute('data-type', tType);
    chip.setAttribute('data-name', tName);
    chip.setAttribute('data-id', '0');
    var urlName = encodeURIComponent(tName.toLowerCase().replace(/\s+/g, '-'));
    chip.setAttribute('data-url', '/' + tType + '/' + urlName + '/');
    chip.style.cssText =
      'display: inline-flex; align-items: center; margin: 2px; padding: 2px 8px; border-radius: 3px; background: #2a2a2a;';
    chip.innerHTML =
      '<span class="name" style="margin-right:4px">[' +
      tType +
      '] ' +
      tName +
      '</span>';
    var delBtn = document.createElement('a');
    delBtn.href = 'javascript:;';
    delBtn.textContent = 'x';
    delBtn.style.cssText =
      'color: #e74c3c; text-decoration: none; font-weight: bold;';
    delBtn.onclick = function () {
      chip.remove();
      relUpdateBtn();
    };
    chip.appendChild(delBtn);
    groupChips.appendChild(chip);
    var h = groupChips.querySelector('span[style]');
    if (h && h.style.color === 'rgb(102, 102, 102)') h.remove();
    relNewNameInput.value = '';
    relUpdateBtn();
  };
  newPanel.appendChild(relAddNewBtn);

  // Save button
  function relCollectTags() {
    var chips = groupChips.querySelectorAll('.relation-chips');
    var tags = [];
    chips.forEach(function (c) {
      tags.push({
        type: c.getAttribute('data-type'),
        name: c.getAttribute('data-name'),
        url: c.getAttribute('data-url'),
        id: parseInt(c.getAttribute('data-id')) || 0,
      });
    });
    return tags;
  }

  function relUpdateBtn() {
    var chips = groupChips.querySelectorAll('.relation-chips');
    groupSaveBtn.style.opacity = chips.length >= 2 ? '1' : '0.4';
    groupSaveBtn.style.pointerEvents = chips.length >= 2 ? 'auto' : 'none';
  }

  var groupSaveBtn = document.createElement('a');
  groupSaveBtn.href = 'javascript:;';
  groupSaveBtn.className = 'btn btn-primary';
  groupSaveBtn.textContent = 'Save Relation Group';
  groupSaveBtn.style.cssText =
    'margin-top: 8px; display: inline-block; opacity: 0.4; pointer-events: none;';
  groupSaveBtn.onclick = function () {
    var tags = relCollectTags();
    if (tags.length < 2) {
      window.showToast('至少需要 2 个 tag', { type: 'error' });
      return;
    }
    var payload = JSON.stringify({ tags: tags });
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/api/comic/tags/relation');
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function () {
      if (xhr.status >= 200 && xhr.status < 300) {
        window.showToast('关系组已创建', { type: 'success' });
        if (wrapper) window.closeModal(wrapper);
        setTimeout(function () {
          location.reload();
        }, 500);
      } else {
        try {
          var r = JSON.parse(xhr.responseText);
          window.showToast((r.head && r.head.msg) || '创建失败', {
            type: 'error',
          });
        } catch (e) {
          window.showToast('创建失败', { type: 'error' });
        }
      }
    };
    xhr.onerror = function () {
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send(payload);
  };
  addSection.appendChild(groupSaveBtn);

  content.appendChild(addSection);

  // Existing relation groups
  var existTitle = document.createElement('h4');
  existTitle.style.cssText = 'margin: 0 0 8px 0;';
  existTitle.textContent = 'Existing Relation Groups';
  content.appendChild(existTitle);

  if (groups.length === 0) {
    var noGroups = document.createElement('p');
    noGroups.style.cssText = 'color: #888;';
    noGroups.textContent = 'No relation groups yet.';
    content.appendChild(noGroups);
  } else {
    groups.forEach(function (group) {
      var groupEl = document.createElement('div');
      groupEl.style.cssText =
        'margin-bottom: 8px; padding: 8px; border: 1px solid #444; border-radius: 4px;';
      var tagsHtml = '';
      group.tags.forEach(function (t) {
        tagsHtml +=
          '<a href="/tag/' +
          encodeURIComponent(t.type) +
          '/' +
          encodeURIComponent(t.name) +
          '/" class="tag tag-' +
          (t.id || 0) +
          '">' +
          '<span class="name">[' +
          t.type +
          '] ' +
          t.name +
          '</span></a>';
      });
      groupEl.innerHTML =
        '<div style="margin-bottom:4px;">' +
        tagsHtml +
        '</div>' +
        '<div style="display:flex; justify-content:space-between; align-items:center;">' +
        '<span style="color:#666;font-size:12px;">' +
        (group.created_at || '') +
        '</span>' +
        '<a href="javascript:;" class="btn btn-secondary" style="padding:2px 8px;font-size:12px;" onclick="deleteRelationGroup(\'' +
        group.id +
        '\', true)"><i class="fa fa-trash"></i> Delete</a></div>';
      content.appendChild(groupEl);
    });
  }

  wrapper = window.showCustomModal(
    'Manage Relations',
    content,
    '<a href="javascript:;" class="btn btn-secondary" onclick="closeModal(this.closest(\'.modal-wrapper\'))">Close</a>',
  );
}

function deleteRelationGroup(groupId, doReload) {
  if (!confirm('确定删除该关系组？')) return;
  var xhr = new XMLHttpRequest();
  xhr.withCredentials = true;
  xhr.open('DELETE', '/api/comic/tags/relation');
  xhr.setRequestHeader('Content-Type', 'application/json');
  xhr.onload = function () {
    if (xhr.status >= 200 && xhr.status < 300) {
      window.showToast('关系组已删除', { type: 'success' });
      if (doReload) {
        setTimeout(function () {
          location.reload();
        }, 300);
      }
    } else {
      try {
        var r = JSON.parse(xhr.responseText);
        window.showToast((r.head && r.head.msg) || '删除失败', {
          type: 'error',
        });
      } catch (e) {
        window.showToast('删除失败', { type: 'error' });
      }
    }
  };
  xhr.onerror = function () {
    window.showToast('网络错误', { type: 'error' });
  };
  xhr.send(JSON.stringify({ id: groupId }));
}
