/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Tag aligner modal for search results page.
 */
(function () {
  'use strict';

  window.openTagAligner = function openTagAligner(query) {
    window.showToast('正在获取去重标签列表...', { type: 'info' });

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(
      'GET',
      '/api/comic/tags/search-unique?q=' +
        encodeURIComponent(query) +
        '&limit=500',
    );
    xhr.onload = function () {
      if (xhr.status !== 200) {
        window.showToast('获取标签列表失败', { type: 'error' });
        return;
      }
      try {
        var resp = JSON.parse(xhr.responseText);
        var data = resp.body;
        var tags = data.tags || [];
        var cidList = data.cidList || [];
        var total = data.total || 0;

        if (tags.length === 0) {
          window.showToast('搜索结果中没有标签', { type: 'info' });
          return;
        }

        buildTagAlignerModal(cidList, tags, query);
      } catch (e) {
        window.showToast('解析响应失败', { type: 'error' });
      }
    };
    xhr.onerror = function () {
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send();
  };

  function buildTagAlignerModal(cidList, tags, query) {
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
    var selectedTag = null;

    // Group by type
    var tagGroups = {};
    tags.forEach(function (t) {
      if (!tagGroups[t.type]) tagGroups[t.type] = [];
      tagGroups[t.type].push(t);
    });

    var typeLabels = {
      parody: 'Parodies',
      character: 'Characters',
      tag: 'Tags',
      artist: 'Artists',
      group: 'Groups',
      language: 'Languages',
      category: 'Categories',
      custom: 'Customs',
    };

    var content = document.createElement('div');
    var infoPara = document.createElement('p');
    infoPara.style.cssText = 'margin-bottom:10px';
    infoPara.textContent =
      '搜索 "' +
      query +
      '" 匹配 ' +
      cidList.length +
      ' 本漫画，共 ' +
      tags.length +
      ' 个去重标签。选择要批量添加的标签：';
    content.appendChild(infoPara);

    Object.keys(tagGroups)
      .sort()
      .forEach(function (type) {
        var groupTags = tagGroups[type];
        var groupDiv = document.createElement('div');
        groupDiv.style.cssText = 'margin-bottom: 8px;';

        var header = document.createElement('h4');
        header.style.cssText =
          'margin: 0 0 4px 0; color: #888; font-size: 13px;';
        header.textContent = typeLabels[type] || type;
        groupDiv.appendChild(header);

        groupTags.forEach(function (t) {
          var tagEl = document.createElement('a');
          tagEl.href = 'javascript:;';
          tagEl.className = 'tag tag-' + (t.id || 0);
          tagEl.style.cssText =
            'display: inline-block; margin: 2px; padding: 2px 8px; cursor: pointer;';
          tagEl.innerHTML =
            '<span class="name">' +
            esc(t.name) +
            '</span><span class="count">' +
            esc(t.count) +
            '</span>';

          tagEl.onclick = function () {
            content.querySelectorAll('.tag.selected').forEach(function (el) {
              el.classList.remove('selected');
              el.style.outline = '';
            });
            tagEl.classList.add('selected');
            tagEl.style.outline = '2px solid #4CAF50';
            selectedTag = {
              id: t.id,
              name: t.name,
              type: t.type,
              url: t.url,
              count: 1,
            };

            var applyBtn = content.querySelector('#applyTagBtn');
            if (applyBtn) {
              applyBtn.style.opacity = '1';
              applyBtn.style.pointerEvents = 'auto';
            }
          };
          groupDiv.appendChild(tagEl);
        });
        content.appendChild(groupDiv);
      });

    var btnContent = document.createElement('div');
    btnContent.style.cssText =
      'display: flex; gap: 8px; align-items: center; margin-top: 10px;';

    var applyBtn = document.createElement('a');
    applyBtn.id = 'applyTagBtn';
    applyBtn.href = 'javascript:;';
    applyBtn.className = 'btn btn-primary';
    applyBtn.textContent = 'Apply to All (' + cidList.length + ')';
    applyBtn.style.cssText = 'opacity: 0.4; pointer-events: none;';

    applyBtn.onclick = function () {
      if (!selectedTag) {
        window.showToast('请先选择一个标签', { type: 'error' });
        return;
      }
      window.LoadingManager.start(applyBtn);

      var payload = JSON.stringify({ cidList: cidList, tag: selectedTag });
      var xhr = new XMLHttpRequest();
      xhr.withCredentials = true;
      xhr.open('POST', '/api/comic/tags/batch-add');
      xhr.setRequestHeader('Content-Type', 'application/json');
      xhr.onload = function () {
        window.LoadingManager.done(applyBtn);
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            var resp = JSON.parse(xhr.responseText);
            var data = resp.body;
            var msg =
              '标签 "' +
              selectedTag.name +
              '" 已添加到 ' +
              data.updated +
              '/' +
              cidList.length +
              ' 本漫画';
            if (data.errors && data.errors.length > 0) {
              msg += '，' + data.errors.length + ' 本失败';
            }
            window.showToast(msg, { type: 'success' });
            window.closeModal(content.closest('.modal-wrapper'));
          } catch (e) {
            window.showToast('处理完成', { type: 'success' });
            window.closeModal(content.closest('.modal-wrapper'));
          }
        } else {
          window.LoadingManager.error(applyBtn);
          try {
            var r = JSON.parse(xhr.responseText);
            window.showToast((r.head && r.head.msg) || '批量添加失败', {
              type: 'error',
            });
          } catch (e) {
            window.showToast('批量添加失败: ' + xhr.status, { type: 'error' });
          }
        }
      };
      xhr.onerror = function () {
        window.LoadingManager.error(applyBtn);
        window.showToast('网络错误', { type: 'error' });
      };
      xhr.send(payload);
    };
    btnContent.appendChild(applyBtn);

    var cancelBtn = document.createElement('a');
    cancelBtn.href = 'javascript:;';
    cancelBtn.className = 'btn btn-secondary';
    cancelBtn.textContent = 'Cancel';
    cancelBtn.onclick = function () {
      window.closeModal(content.closest('.modal-wrapper'));
    };
    btnContent.appendChild(cancelBtn);

    content.appendChild(btnContent);
    window.showCustomModal('Align Tags', content, '');
  }
})();
