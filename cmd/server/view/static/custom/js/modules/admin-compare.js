// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

(function () {
  'use strict';

  var currentCID1 = 0;
  var currentCID2 = 0;
  var compareData = null;

  // 多漫画对比：标题缓存
  window._comicTitles = window._comicTitles || {};
  window._pendingMultiCIDs = window._pendingMultiCIDs || [];

  /* ===== 对比操作 ===== */
  window.compareComics = function () {
    var cid1 = parseInt(document.getElementById('cid-main').value, 10);
    var cid2 = parseInt(document.getElementById('cid-target').value, 10);
    if (!cid1 || !cid2 || cid1 === cid2) {
      showAdminToast('请输入两个不同的有效 CID');
      return;
    }
    // 如果预览 overlay 仍打开，自动关闭
    closePreview();
    currentCID1 = cid1;
    currentCID2 = cid2;

    fetch('/api/admin/comic/compare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cid1: cid1, cid2: cid2 }),
    })
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function (data) {
        compareData = data;
        // 缓存标题
        if (data.body && data.body.cid1 && data.body.cid1.info) {
          window._comicTitles[data.body.cid1.info.cid] =
            (data.body.cid1.info.title && data.body.cid1.info.title.english) ||
            '';
        }
        if (data.body && data.body.cid2 && data.body.cid2.info) {
          window._comicTitles[data.body.cid2.info.cid] =
            (data.body.cid2.info.title && data.body.cid2.info.title.english) ||
            '';
        }
        renderCompareResult(data.body);
        loadLinks(currentCID1, currentCID2);
      })
      .catch(function (err) {
        showAdminToast('对比失败: ' + err.message);
      });
  };

  window.swapCids = function () {
    var a = document.getElementById('cid-main');
    var b = document.getElementById('cid-target');
    var t = a.value;
    a.value = b.value;
    b.value = t;
  };

  /* ===== 渲染对比结果 ===== */
  function renderCompareResult(body) {
    var container = document.getElementById('compare-result');
    container.style.display = 'block';

    var info1 = body.cid1.info;
    var info2 = body.cid2.info;

    document.getElementById('comic-info-pair').innerHTML =
      '<div style="flex:1;background:#1a1a1a;border:1px solid #333;border-radius:4px;padding:12px;border-left:3px solid #ed2553;">' +
      '<strong style="color:#ed2553;">CID ' +
      info1.cid +
      '</strong>' +
      '<div style="color:#aaa;font-size:13px;margin-top:4px;">' +
      escapeHtml((info1.title && info1.title.english) || '') +
      '</div>' +
      '<div style="font-size:11px;color:#666;margin-top:4px;">' +
      (body.cid1.pages.length || 0) +
      ' 页</div>' +
      '</div>' +
      '<div style="flex:1;background:#1a1a1a;border:1px solid #333;border-radius:4px;padding:12px;border-left:3px solid #f39c12;">' +
      '<strong style="color:#f39c12;">CID ' +
      info2.cid +
      '</strong>' +
      '<div style="color:#aaa;font-size:13px;margin-top:4px;">' +
      escapeHtml((info2.title && info2.title.english) || '') +
      '</div>' +
      '<div style="font-size:11px;color:#666;margin-top:4px;">' +
      (body.cid2.pages.length || 0) +
      ' 页</div>' +
      '</div>';

    var stats = body.stats;
    document.getElementById('stats-bar').innerHTML =
      '<div style="display:flex;gap:16px;flex-wrap:wrap;font-size:13px;padding:8px 0;">' +
      '<span><strong>对齐页数：</strong>' +
      stats.total +
      '</span>' +
      '<span><strong>匹配度：</strong><span style="color:#4caf50;font-weight:bold;">' +
      (stats.match_ratio * 100).toFixed(1) +
      '%</span></span>' +
      '<span><strong style="color:#4caf50;">✅ ' +
      stats.matched +
      ' 匹配</strong></span>' +
      '<span><strong style="color:#f44336;">❌ ' +
      stats.mismatched +
      ' 不匹配</strong></span>' +
      '</div>';

    // 渲染 tag 差异
    renderTagDiff(body);

    var html =
      '<table style="width:100%;border-collapse:collapse;font-size:13px;">' +
      '<thead><tr style="background:#2a2a2a;">' +
      '<th style="padding:6px 10px;text-align:left;">文件名</th>' +
      '<th style="padding:6px 10px;text-align:left;">CID1 MD5</th>' +
      '<th style="padding:6px 10px;text-align:left;">CID2 MD5</th>' +
      '<th style="padding:6px 10px;text-align:center;">状态</th>' +
      '<th style="padding:6px 10px;">操作</th>' +
      '</tr></thead><tbody>';
    (body.comparison || []).forEach(function (row) {
      var ok = row.md5_match;
      var cls = ok ? '' : ' style="background:rgba(244,67,54,0.08);"';
      var cid1md5 = row.cid1_md5
        ? row.cid1_md5.substring(0, 12) + '...'
        : '<span style="color:#ffc107;">无</span>';
      var cid2md5 = row.cid2_md5
        ? row.cid2_md5.substring(0, 12) + '...'
        : '<span style="color:#ffc107;">无</span>';
      var cid2Color = ok ? 'style="color:#666;"' : 'style="color:#f44336;"';
      html +=
        '<tr' +
        cls +
        '>' +
        '<td style="padding:4px 10px;">' +
        escapeHtml(row.name) +
        '</td>' +
        '<td style="padding:4px 10px;font-family:monospace;font-size:11px;color:#666;">' +
        cid1md5 +
        '</td>' +
        '<td style="padding:4px 10px;font-family:monospace;font-size:11px;' +
        cid2Color +
        '">' +
        cid2md5 +
        '</td>' +
        '<td style="padding:4px 10px;text-align:center;">' +
        (ok
          ? '<span style="color:#4caf50;">✅</span>'
          : '<span style="color:#f44336;">❌</span>') +
        '</td>' +
        '<td style="padding:4px 10px;text-align:center;">' +
        (ok
          ? ''
          : '<button class="btn btn-primary btn-sm" onclick="showPreview(\'' +
            escapeHtml(row.name) +
            '\')">‎▶ 并排预览</button>') +
        '</td>' +
        '</tr>';
    });
    html += '</tbody></table>';
    document.getElementById('compare-table-container').innerHTML = html;

    renderLinkAction(body);

    // 更新多漫画选择栏
    if (window._pendingMultiCIDs && window._pendingMultiCIDs.length > 0) {
      var allCIDs = [body.cid1.info.cid].concat(window._pendingMultiCIDs);
      if (allCIDs.indexOf(body.cid2.info.cid) === -1)
        allCIDs.push(body.cid2.info.cid);
      renderMultiComicBar(allCIDs);
    }
  }

  /* ===== Tag 差异渲染 ===== */
  function renderTagDiff(body) {
    var info1 = body.cid1.info;
    var info2 = body.cid2.info;
    var tags1 = info1.tags || [];
    var tags2 = info2.tags || [];

    var set1 = {},
      set2 = {};
    tags1.forEach(function (t) {
      set1[t.type + ':' + t.name] = t;
    });
    tags2.forEach(function (t) {
      set2[t.type + ':' + t.name] = t;
    });

    var onlyIn1 = [],
      onlyIn2 = [],
      inBoth = [];
    tags1.forEach(function (t) {
      var key = t.type + ':' + t.name;
      if (set2[key]) inBoth.push(t);
      else onlyIn1.push(t);
    });
    tags2.forEach(function (t) {
      var key = t.type + ':' + t.name;
      if (!set1[key]) onlyIn2.push(t);
    });

    var html = '<div class="tag-diff-section">';
    html +=
      '<div class="tag-diff-header"><i class="fa fa-tags color-icon"></i> 标签差异</div>';
    html +=
      '<div><span style="color:#ed2553;">主 CID 独有 (' +
      onlyIn1.length +
      ')</span>：';
    if (onlyIn1.length === 0)
      html += '<span style="color:#666;font-size:12px;">(无)</span>';
    onlyIn1.forEach(function (t) {
      html +=
        '<span class="tag-group tag-only-in-main">' +
        escapeHtml(t.name) +
        ' <span style="color:#888;font-size:10px;">[' +
        t.type +
        ']</span></span>';
    });
    html +=
      '</div><div style="margin-top:4px;"><span style="color:#f39c12;">从 CID 独有 (' +
      onlyIn2.length +
      ')</span>：';
    if (onlyIn2.length === 0)
      html += '<span style="color:#666;font-size:12px;">(无)</span>';
    onlyIn2.forEach(function (t) {
      html +=
        '<span class="tag-group tag-only-in-sub">' +
        escapeHtml(t.name) +
        ' <span style="color:#888;font-size:10px;">[' +
        t.type +
        ']</span></span>';
    });
    html += '</div>';
    if (inBoth.length > 0) {
      html +=
        '<div style="margin-top:4px;"><span style="color:#4caf50;">共有 (' +
        inBoth.length +
        ')</span>：';
      inBoth.forEach(function (t) {
        html +=
          '<span class="tag-group tag-in-both">' +
          escapeHtml(t.name) +
          '</span>';
      });
      html += '</div>';
    }
    html += '</div>';

    var statsBar = document.getElementById('stats-bar');
    var existingDiff = document.getElementById('tag-diff-area');
    if (existingDiff) existingDiff.remove();
    var div = document.createElement('div');
    div.id = 'tag-diff-area';
    div.innerHTML = html;
    statsBar.parentNode.insertBefore(div, statsBar.nextSibling);
  }

  /* ===== 多漫画选择栏 ===== */
  function renderMultiComicBar(cids) {
    var bar = document.getElementById('multi-comic-bar');
    if (!bar) return;
    if (!cids || cids.length === 0) {
      bar.style.display = 'none';
      return;
    }
    bar.style.display = 'flex';

    var html =
      '<div style="font-size:13px;color:#888;margin-right:8px;line-height:32px;">所有漫画：</div>';
    cids.forEach(function (cid) {
      var isActive = cid === currentCID1 || cid === currentCID2;
      var cls = isActive ? 'multi-comic-card active' : 'multi-comic-card';
      html +=
        '<div class="' +
        cls +
        '" onclick="selectMainComic(' +
        cid +
        ')">' +
        '<div class="cid">CID ' +
        cid +
        '</div>' +
        '<div class="title">' +
        (window._comicTitles[cid]
          ? escapeHtml(window._comicTitles[cid])
          : '...') +
        '</div>' +
        '</div>';
    });
    bar.innerHTML = html;
  }

  window.selectMainComic = function (cid) {
    if (cid === currentCID1) return;
    // 构建所有漫画列表
    var allCIDs = [currentCID1, currentCID2].concat(
      window._pendingMultiCIDs || [],
    );
    allCIDs = allCIDs.filter(function (v, i, a) {
      return a.indexOf(v) === i;
    });
    var others = allCIDs.filter(function (c) {
      return c !== cid;
    });
    if (others.length === 0) return;
    document.getElementById('cid-main').value = cid;
    document.getElementById('cid-target').value = others[0];
    compareComics();
  };

  function renderLinkAction(body) {
    var cid1 = body.cid1.info.cid;
    var cid2 = body.cid2.info.cid;
    document.getElementById('link-action').innerHTML =
      '<div style="border-top:1px solid #333;padding-top:12px;">' +
      '<h3 style="font-size:14px;margin-bottom:8px;"><i class="fa fa-link color-icon"></i> 建立从属关系</h3>' +
      '<div style="display:flex;gap:12px;align-items:center;flex-wrap:wrap;">' +
      '<label>主：<input id="link-main" type="text" value="' +
      cid1 +
      '" style="width:80px;text-align:center;" /></label>' +
      '<span style="color:#555;">← 从属于 ←</span>' +
      '<label>从：<input id="link-sub" type="text" value="' +
      cid2 +
      '" style="width:80px;text-align:center;" /></label>' +
      '<button class="btn btn-primary" onclick="confirmLink()"><i class="fa fa-link"></i> 建立链接</button>' +
      '</div>' +
      '<div style="font-size:12px;color:#888;margin-top:6px;">链接后从属 comic 重定向到主 comic，tags 自动合并，不在搜索结果展示。</div>' +
      '</div>';
  }

  /* ===== 并排预览（大图半屏覆盖层） ===== */
  window.showPreview = function (fileName) {
    var overlay = document.createElement('div');
    overlay.className = 'preview-overlay';
    overlay.id = 'preview-overlay';

    window._previewFiles = [];
    if (compareData && compareData.body) {
      (compareData.body.comparison || []).forEach(function (row) {
        if (!row.md5_match) window._previewFiles.push(row.name);
      });
    }
    window._previewIndex = window._previewFiles.indexOf(fileName);

    overlay.innerHTML =
      '<div class="preview-header">' +
      '<div>' +
      '<button class="btn btn-secondary btn-sm" onclick="previewNav(-1)"><i class="fa fa-chevron-left"></i> 上一张</button>' +
      ' <span id="preview-counter" style="margin:0 12px;color:#fff;">' +
      (window._previewIndex + 1) +
      '/' +
      window._previewFiles.length +
      '</span>' +
      '<button class="btn btn-secondary btn-sm" onclick="previewNav(1)">下一张 <i class="fa fa-chevron-right"></i></button>' +
      '</div>' +
      '<div>' +
      '<span style="color:#888;margin-right:12px;">← → 方向键翻页 | Esc 关闭</span>' +
      '<button class="btn btn-primary btn-sm" onclick="closePreview()">关闭 ✕</button>' +
      '</div>' +
      '</div>' +
      '<div class="preview-images">' +
      '<div class="preview-col">' +
      '<div style="color:#ed2553;font-weight:bold;font-size:13px;margin-bottom:4px;">CID ' +
      currentCID1 +
      '</div>' +
      '<img src="/galleries/' +
      currentCID1 +
      '/' +
      encodeURIComponent(fileName) +
      '" />' +
      '</div>' +
      '<div class="preview-col">' +
      '<div style="color:#f39c12;font-weight:bold;font-size:13px;margin-bottom:4px;">CID ' +
      currentCID2 +
      '</div>' +
      '<img src="/galleries/' +
      currentCID2 +
      '/' +
      encodeURIComponent(fileName) +
      '" />' +
      '</div>' +
      '</div>';
    document.body.appendChild(overlay);
    document.body.style.overflow = 'hidden';
  };

  window.previewNav = function (delta) {
    var files = window._previewFiles || [];
    if (files.length === 0) return;
    window._previewIndex =
      (window._previewIndex + delta + files.length) % files.length;
    var fileName = files[window._previewIndex];
    var overlay = document.getElementById('preview-overlay');
    if (!overlay) return;

    var cols = overlay.querySelectorAll('.preview-col');
    cols[0].querySelector('img').src =
      '/galleries/' + currentCID1 + '/' + encodeURIComponent(fileName);
    cols[1].querySelector('img').src =
      '/galleries/' + currentCID2 + '/' + encodeURIComponent(fileName);

    document.getElementById('preview-counter').textContent =
      window._previewIndex + 1 + '/' + files.length;
  };

  window.closePreview = function () {
    var overlay = document.getElementById('preview-overlay');
    if (overlay) {
      overlay.remove();
      document.body.style.overflow = '';
    }
  };

  /* ===== 建立链接 ===== */
  window.confirmLink = function () {
    var mainCID = parseInt(document.getElementById('link-main').value, 10);
    var subCID = parseInt(document.getElementById('link-sub').value, 10);
    if (!mainCID || !subCID || mainCID === subCID) {
      showAdminToast('请输入有效的主/从 CID');
      return;
    }
    if (
      !confirm(
        '确认将从属 CID ' +
          subCID +
          ' 链接到主 CID ' +
          mainCID +
          ' ？\n操作可撤销。',
      )
    )
      return;

    fetch('/api/admin/comic/link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ main_cid: mainCID, sub_cid: subCID }),
    })
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function () {
        showAdminToast('链接建立成功！CID ' + subCID + ' → ' + mainCID);
        loadLinks(mainCID, subCID);
      })
      .catch(function (err) {
        showAdminToast('建立链接失败: ' + err.message);
      });
  };

  /* ===== 取消链接 ===== */
  window.unlinkComic = function (subCID) {
    if (
      !confirm('确认取消 CID ' + subCID + ' 的从属关系？已合并的 tags 将保留。')
    )
      return;
    fetch('/api/admin/comic/unlink', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ main_cid: 0, sub_cid: subCID }),
    })
      .then(function (resp) {
        if (!resp.ok) throw new Error('HTTP ' + resp.status);
        return resp.json();
      })
      .then(function () {
        showAdminToast('取消合并成功！CID ' + subCID + ' 已恢复独立。');
        loadLinks(currentCID1, currentCID2);
      })
      .catch(function (err) {
        showAdminToast('取消合并失败: ' + err.message);
      });
  };

  /* ===== 加载链接列表 ===== */
  function loadLinks(mainCID, subCID) {
    fetch('/api/admin/comic/links?all=true')
      .then(function (resp) {
        return resp.json();
      })
      .then(function (data) {
        renderLinksTable(data.body.links || [], mainCID, subCID);
      })
      .catch(function () {
        showAdminToast('加载链接列表失败');
      });
  }

  function renderLinksTable(links, mainCID, subCID) {
    var allHtml = '';
    var currentHtml = '';
    links.forEach(function (link) {
      var isCurrent = link.main_cid === mainCID && link.sub_cid === subCID;
      var row =
        '<tr>' +
        '<td style="padding:4px 10px;"><strong style="color:#ed2553;">' +
        link.main_cid +
        '</strong></td>' +
        '<td style="padding:4px 10px;">' +
        escapeHtml(link.sub_title || '') +
        '</td>' +
        '<td style="padding:4px 10px;"><span style="color:#f39c12;">' +
        link.sub_cid +
        '</span></td>' +
        '<td style="padding:4px 10px;"><button class="btn btn-warning btn-sm" onclick="unlinkComic(' +
        link.sub_cid +
        ')"><i class="fa fa-unlink"></i> 取消合并</button></td>' +
        '</tr>';
      allHtml += row;
      if (isCurrent) currentHtml += row;
    });

    if (!currentHtml)
      currentHtml =
        '<tr><td colspan="4" style="padding:10px;text-align:center;color:#888;">无当前比较相关的链接</td></tr>';
    if (!allHtml)
      allHtml =
        '<tr><td colspan="4" style="padding:10px;text-align:center;color:#888;">暂无链接</td></tr>';

    document.getElementById('linked-table-container').innerHTML =
      '<table id="linked-table-current" class="link-table" style="width:100%;border-collapse:collapse;font-size:13px;">' +
      '<thead><tr style="background:#2a2a2a;"><th style="padding:6px 10px;">主 CID</th><th style="padding:6px 10px;">从属标题</th><th style="padding:6px 10px;">从属 CID</th><th style="padding:6px 10px;">操作</th></tr></thead>' +
      '<tbody>' +
      currentHtml +
      '</tbody>' +
      '</table>' +
      '<table id="linked-table-all" class="link-table" style="width:100%;border-collapse:collapse;font-size:13px;display:none;">' +
      '<thead><tr style="background:#2a2a2a;"><th style="padding:6px 10px;">主 CID</th><th style="padding:6px 10px;">从属标题</th><th style="padding:6px 10px;">从属 CID</th><th style="padding:6px 10px;">操作</th></tr></thead>' +
      '<tbody>' +
      allHtml +
      '</tbody>' +
      '</table>';
  }

  /* ===== 切换链接视图 ===== */
  window.switchLinksView = function (mode) {
    var cTable = document.getElementById('linked-table-current');
    var aTable = document.getElementById('linked-table-all');
    var cBtn = document.getElementById('btn-show-current');
    var aBtn = document.getElementById('btn-show-all');
    if (mode === 'current') {
      cTable.style.display = '';
      aTable.style.display = 'none';
      cBtn.className = 'btn btn-primary btn-sm';
      aBtn.className = 'btn btn-secondary btn-sm';
    } else {
      cTable.style.display = 'none';
      aTable.style.display = '';
      cBtn.className = 'btn btn-secondary btn-sm';
      aBtn.className = 'btn btn-primary btn-sm';
    }
  };

  /* ===== 键盘快捷键（全局） ===== */
  document.addEventListener('keydown', function (e) {
    var overlay = document.getElementById('preview-overlay');
    if (!overlay) return;
    if (e.key === 'Escape') {
      closePreview();
      e.preventDefault();
    }
    if (e.key === 'ArrowLeft') {
      previewNav(-1);
      e.preventDefault();
    }
    if (e.key === 'ArrowRight') {
      previewNav(1);
      e.preventDefault();
    }
  });

  /* ===== 工具函数 ===== */
  function showAdminToast(msg) {
    var el = document.createElement('div');
    el.className = 'alert';
    el.textContent = msg;
    var msgContainer = document.getElementById('messages');
    if (msgContainer) {
      msgContainer.appendChild(el);
      setTimeout(function () {
        el.remove();
      }, 3000);
    } else {
      alert(msg);
    }
  }

  function escapeHtml(str) {
    if (!str) return '';
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }
})();
