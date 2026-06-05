// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

(function () {
  'use strict';

  var currentCID1 = 0;
  var currentCID2 = 0;
  var compareData = null;

  /* ===== 对比操作 ===== */
  window.compareComics = function () {
    var cid1 = parseInt(document.getElementById('cid-main').value, 10);
    var cid2 = parseInt(document.getElementById('cid-target').value, 10);
    if (!cid1 || !cid2 || cid1 === cid2) {
      showAdminToast('请输入两个不同的有效 CID');
      return;
    }
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
  }

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

  /* ===== 并排预览 ===== */
  window.showPreview = function (fileName) {
    var panel = document.getElementById('preview-panel');
    panel.style.display = 'block';
    panel.innerHTML =
      '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">' +
      '<h3 style="margin:0;font-size:14px;">' +
      escapeHtml(fileName) +
      '</h3>' +
      '<button class="btn btn-secondary btn-sm" onclick="document.getElementById(\'preview-panel\').style.display=\'none\'">关闭</button>' +
      '</div>' +
      '<div style="display:flex;gap:12px;">' +
      '<div style="flex:1;text-align:center;">' +
      '<div style="color:#ed2553;font-weight:bold;font-size:12px;">CID ' +
      currentCID1 +
      '</div>' +
      '<img src="/galleries/' +
      currentCID1 +
      '/' +
      encodeURIComponent(fileName) +
      '" style="max-width:100%;max-height:300px;border-radius:4px;" />' +
      '</div>' +
      '<div style="flex:1;text-align:center;">' +
      '<div style="color:#f39c12;font-weight:bold;font-size:12px;">CID ' +
      currentCID2 +
      '</div>' +
      '<img src="/galleries/' +
      currentCID2 +
      '/' +
      encodeURIComponent(fileName) +
      '" style="max-width:100%;max-height:300px;border-radius:4px;" />' +
      '</div>' +
      '</div>';
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

  /* ===== 页面加载时自动加载链接 ===== */
  // document.addEventListener('DOMContentLoaded', function () {
  //   loadLinks(0, 0);
  // });

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
