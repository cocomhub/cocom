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
    xhr.open('GET', '/api/comic/tags/relation?type=' + encodeURIComponent(type) + '&name=' + encodeURIComponent(name) + '&id=' + encodeURIComponent(id));
    xhr.onload = function() {
        if (xhr.status !== 200) { showToast('获取关系列表失败', { type: 'error' }); return; }
        try {
            var resp = JSON.parse(xhr.responseText);
            var groups = (resp.body && resp.body.groups) || [];
            buildRelationModal(type, name, id, groups);
        } catch(e) { showToast('解析失败', { type: 'error' }); }
    };
    xhr.onerror = function() { showToast('网络错误', { type: 'error' }); };
    xhr.send();
}

function buildRelationModal(srcType, srcName, srcId, groups) {
    var tagTypes = [
        {value:'parody', label:'Parodies'}, {value:'character', label:'Characters'},
        {value:'tag', label:'Tags'}, {value:'artist', label:'Artists'},
        {value:'group', label:'Groups'}, {value:'language', label:'Languages'},
        {value:'category', label:'Categories'}, {value:'custom', label:'Customs'}
    ];

    var wrapper;
    var content = document.createElement('div');

    // Add new relation section
    var addSection = document.createElement('div');
    addSection.style.cssText = 'margin-bottom: 15px; padding-bottom: 15px; border-bottom: 1px solid #444;';
    var addTitle = document.createElement('h4');
    addTitle.style.cssText = 'margin: 0 0 8px 0;';
    addTitle.textContent = 'Add New Relation Group';
    addSection.appendChild(addTitle);

    var groupChips = document.createElement('div');
    groupChips.style.cssText = 'display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 8px; min-height: 30px;';
    var hint = document.createElement('span');
    hint.style.cssText = 'color: #666; font-size: 13px;';
    hint.textContent = 'Select at least 2 tags to form a relation group.';
    groupChips.appendChild(hint);
    addSection.appendChild(groupChips);

    var selectorRow = document.createElement('div');
    selectorRow.style.cssText = 'display: flex; gap: 5px; align-items: center; flex-wrap: wrap;';
    var typeSelect = document.createElement('select');
    typeSelect.style.cssText = 'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px;';
    tagTypes.forEach(function(tt) {
        var opt = document.createElement('option');
        opt.value = tt.value; opt.textContent = tt.label;
        typeSelect.appendChild(opt);
    });
    selectorRow.appendChild(typeSelect);
    var nameInput = document.createElement('input');
    nameInput.type = 'text';
    nameInput.placeholder = 'Tag name';
    nameInput.style.cssText = 'padding: 4px; background: #333; color: #fff; border: 1px solid #555; border-radius: 3px; flex: 1; min-width: 120px;';
    selectorRow.appendChild(nameInput);
    var addBtn = document.createElement('a');
    addBtn.href = 'javascript:;';
    addBtn.className = 'btn btn-secondary';
    addBtn.textContent = 'Add to Group';
    addBtn.style.cssText = 'padding: 4px 10px;';
    addBtn.onclick = function() {
        var tName = nameInput.value.trim();
        var tType = typeSelect.value;
        if (!tName) { showToast('请输入 tag 名称', { type: 'error' }); return; }
        if (tType === srcType && tName === srcName) { showToast('不能将当前 tag 添加到组中', { type: 'error' }); return; }

        var existing = groupChips.querySelectorAll('.relation-chips');
        for (var i = 0; i < existing.length; i++) {
            if (existing[i].getAttribute('data-type') === tType && existing[i].getAttribute('data-name') === tName) {
                showToast('该 tag 已在组中', { type: 'info' }); return;
            }
        }

        var chip = document.createElement('span');
        chip.className = 'tag relation-chips';
        chip.setAttribute('data-type', tType);
        chip.setAttribute('data-name', tName);
        var urlName = encodeURIComponent(tName.toLowerCase().replace(/\s+/g, '-'));
        chip.setAttribute('data-url', '/' + tType + '/' + urlName + '/');
        chip.style.cssText = 'display: inline-flex; align-items: center; margin: 2px; padding: 2px 8px; border-radius: 3px; background: #2a2a2a;';
        chip.innerHTML = '<span class="name" style="margin-right:4px">[' + tType + '] ' + tName + '</span>';
        var delBtn = document.createElement('a');
        delBtn.href = 'javascript:;';
        delBtn.textContent = 'x';
        delBtn.style.cssText = 'color: #e74c3c; text-decoration: none; font-weight: bold;';
        delBtn.onclick = function() { chip.remove(); updateBtnState(); };
        chip.appendChild(delBtn);
        groupChips.appendChild(chip);
        var hintEl = groupChips.querySelector('span[style]');
        if (hintEl && hintEl.style.color === 'rgb(102, 102, 102)') hintEl.remove();
        nameInput.value = '';
        updateBtnState();
    };
    selectorRow.appendChild(addBtn);
    addSection.appendChild(selectorRow);

    var groupSaveBtn = document.createElement('a');
    groupSaveBtn.href = 'javascript:;';
    groupSaveBtn.className = 'btn btn-primary';
    groupSaveBtn.textContent = 'Save Relation Group';
    groupSaveBtn.style.cssText = 'margin-top: 8px; display: inline-block; opacity: 0.4; pointer-events: none;';
    groupSaveBtn.onclick = function() {
        var chips = groupChips.querySelectorAll('.relation-chips');
        if (chips.length < 2) { showToast('至少需要 2 个 tag', { type: 'error' }); return; }
        var tags = [];
        chips.forEach(function(c) {
            tags.push({
                type: c.getAttribute('data-type'),
                name: c.getAttribute('data-name'),
                url: c.getAttribute('data-url'),
                id: 0
            });
        });
        var payload = JSON.stringify({tags: tags});
        var xhr = new XMLHttpRequest();
        xhr.withCredentials = true;
        xhr.open('POST', '/api/comic/tags/relation');
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.onload = function() {
            if (xhr.status >= 200 && xhr.status < 300) {
                showToast('关系组已创建', { type: 'success' });
                if (wrapper) closeModal(wrapper);
                setTimeout(function() { location.reload(); }, 500);
            } else {
                try { var r = JSON.parse(xhr.responseText); showToast(r.head && r.head.msg || '创建失败', { type: 'error' }); }
                catch(e) { showToast('创建失败', { type: 'error' }); }
            }
        };
        xhr.onerror = function() { showToast('网络错误', { type: 'error' }); };
        xhr.send(payload);
    };
    addSection.appendChild(groupSaveBtn);

    function updateBtnState() {
        var chips = groupChips.querySelectorAll('.relation-chips');
        groupSaveBtn.style.opacity = chips.length >= 2 ? '1' : '0.4';
        groupSaveBtn.style.pointerEvents = chips.length >= 2 ? 'auto' : 'none';
    }

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
        groups.forEach(function(group) {
            var groupEl = document.createElement('div');
            groupEl.style.cssText = 'margin-bottom: 8px; padding: 8px; border: 1px solid #444; border-radius: 4px;';
            var tagsHtml = '';
            group.tags.forEach(function(t) {
                tagsHtml += '<a href="/tag/' + encodeURIComponent(t.type) + '/' + encodeURIComponent(t.name) + '/" class="tag tag-' + (t.id || 0) + '">' +
                    '<span class="name">[' + t.type + '] ' + t.name + '</span></a>';
            });
            groupEl.innerHTML = '<div style="margin-bottom:4px;">' + tagsHtml + '</div>' +
                '<div style="display:flex; justify-content:space-between; align-items:center;">' +
                '<span style="color:#666;font-size:12px;">' + (group.created_at || '') + '</span>' +
                '<a href="javascript:;" class="btn btn-secondary" style="padding:2px 8px;font-size:12px;" onclick="deleteRelationGroup(\'' + group.id + '\', true)"><i class="fa fa-trash"></i> Delete</a></div>';
            content.appendChild(groupEl);
        });
    }

    wrapper = showCustomModal('Manage Relations', content,
        '<a href="javascript:;" class="btn btn-secondary" onclick="closeModal(this.closest(\'.modal-wrapper\'))">Close</a>'
    );
}

function deleteRelationGroup(groupId, doReload) {
    if (!confirm('确定删除该关系组？')) return;
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('DELETE', '/api/comic/tags/relation');
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function() {
        if (xhr.status >= 200 && xhr.status < 300) {
            showToast('关系组已删除', { type: 'success' });
            if (doReload) { setTimeout(function() { location.reload(); }, 300); }
        } else {
            try { var r = JSON.parse(xhr.responseText); showToast(r.head && r.head.msg || '删除失败', { type: 'error' }); }
            catch(e) { showToast('删除失败', { type: 'error' }); }
        }
    };
    xhr.onerror = function() { showToast('网络错误', { type: 'error' }); };
    xhr.send(JSON.stringify({id: groupId}));
}