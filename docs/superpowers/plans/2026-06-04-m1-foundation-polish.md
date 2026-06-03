# Milestone 1: 基础体验打磨 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 把近期新增的 UI 交互（Modal、Toast、LoadingManager、OptimisticUpdater、缩放控制、边栏）打磨到稳定可用状态，补齐缺失的基础体验。移除导航栏上冗余的大图模式入口，由左侧边栏统一管理。

**架构：** 前端三件套（scripts.js、styles.css、tpl）同步修改。JS 从单文件 ~1518 行拆分为多模块；CSS 提炼自定义属性；模板删除导航大图模式链接；后端 API 响应格式统一（小版本）。

**技术栈：** Vanilla JS（无框架）、CSS3 Custom Properties、Go/Gin 后端。

---

## 文件清单

### 创建文件

| 文件 | 职责 |
|------|------|
| `cmd/server/view/static/custom/js/modules/toast.js` | Toast 通知系统（showToast、showProgressToast） |
| `cmd/server/view/static/custom/js/modules/loading-manager.js` | LoadingManager（start/done/error） |
| `cmd/server/view/static/custom/js/modules/optimistic-updater.js` | OptimisticUpdater（optimisticToggle、refreshContainer） |
| `cmd/server/view/static/custom/js/modules/autocomplete.js` | 自动补全键盘导航（bindAutocompleteKeys、enableAutocompleteKeyboardNav） |
| `cmd/server/view/static/custom/js/modules/modal.js` | 通用弹窗（showCustomModal、closeModal） |
| `cmd/server/view/static/custom/js/modules/gallery-actions.js` | 画廊操作（addLikeGroup、archiveComic、restoreComic、archiveComicForce、verifyComic、addLikeTag、removeLikeTag、findCustomsContainer、formatError、highlightInvalidPages、ensureForceArchiveButton） |
| `cmd/server/view/static/custom/js/modules/tag-editor.js` | Tag 编辑器（openTagEditor、buildTagEditorModal） |
| `cmd/server/view/static/custom/js/modules/tag-actions.js` | Tag 辅助函数（rebuildTagsSection、toggleLikeTag） |
| `cmd/server/view/static/custom/js/modules/thumbnail-zoom.js` | 缩略图缩放（initThumbnailZoom、toggleLargeMode、initLargeModeToggle、toggleMobileZoom） |
| `cmd/server/view/static/custom/js/modules/tag-aligner.js` | Tag 对齐器（openTagAligner、buildTagAlignerModal） |
| `cmd/server/view/static/custom/js/modules/related-tags.js` | 关联标签（loadRelatedTags、openTagRelationManager） |
| `cmd/server/view/static/custom/js/modules/navigation.js` | 键盘快捷键、页面导航（jumpToPage、jumpToTagPage） |
| `cmd/server/view/static/custom/js/modules/skeleton.js` | 骨架屏/空状态占位（showSkeleton、hideSkeleton、showEmptyState） |

### 修改文件

| 文件 | 涉及任务 |
|------|----------|
| `cmd/server/view/static/custom/js/scripts.js` | T3（改为入口文件，import 所有模块） |
| `cmd/server/view/static/custom/css/styles.css` | T4, T5 |
| `cmd/server/view/static/tpl/head.tpl` | T1 |
| `cmd/server/view/static/tpl/gallery_detail.tpl` | T5（骨架屏占位） |
| `cmd/server/view/static/tpl/index.tpl` | T5（骨架屏占位） |

---

### 任务 1：移除导航栏大图模式按钮

**文件：**
- 修改：`cmd/server/view/static/tpl/head.tpl:52,87`

head.tpl 中 `{{define "navigation.tpl"}}` 有两个 `大图模式` 链接：
- 桌面菜单（line 52）：`<li class="desktop "><a href="{{$.URL}}?large=true">大图模式</a></li>`
- 下拉菜单（line 87）：`<li><a href="{{$.URL}}?large=true">大图模式</a></li>`

- [ ] **步骤 1：删除桌面菜单的大图模式链接**

在 `cmd/server/view/static/tpl/head.tpl` 中找到并删除第 51-53 行：

```
                <li class="desktop ">
                    <a href="{{$.URL}}?large=true">大图模式</a>
                </li>
```

- [ ] **步骤 2：删除下拉菜单的大图模式链接**

同一文件中找到并删除第 85-87 行：

```
                        <li>
                            <a href="{{$.URL}}?large=true">大图模式</a>
                        </li>
```

- [ ] **步骤 3：验证模板编译通过**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./cmd/server/view/...
```

预期：无编译错误。

- [ ] **步骤 4：Commit**

```bash
git add cmd/server/view/static/tpl/head.tpl
git commit -m "feat(ui): 移除导航栏大图模式按钮，由左侧边栏统一管理"
```

---

### 任务 2：CSS 变量提炼

**文件：**
- 修改：`cmd/server/view/static/custom/css/styles.css`

- [ ] **步骤 1：在 styles.css 顶部添加 :root 自定义属性**

在文件顶部版权声明之后添加：

```css
/* ===== CSS 自定义属性（Design Tokens） ===== */

:root {
    /* 主题色 */
    --color-primary: #ed2553;
    --color-primary-hover: #ff3b6f;
    --color-bg: #1f1f1f;
    --color-bg-raised: #2a2a2a;
    --color-bg-overlay: rgba(30, 30, 30, 0.85);
    --color-text: #fff;
    --color-text-secondary: #ccc;
    --color-text-muted: #888;
    --color-border: #555;
    --color-danger: #e74c3c;
    --color-success: #4CAF50;
    --color-warning: #f0ad4e;

    /* 间距 */
    --space-xs: 4px;
    --space-sm: 8px;
    --space-md: 12px;
    --space-lg: 16px;

    /* 侧边栏尺寸 */
    --sidebar-btn-min-width: 52px;
    --sidebar-btn-min-width-mobile: 44px;

    /* 圆角 */
    --radius-sm: 3px;
    --radius-md: 8px;

    /* 动画 */
    --transition-fast: 0.15s;
    --transition-normal: 0.3s;

    /* 断点（供 JS 参考，CSS 中仍使用 media query） */
    --breakpoint-mobile: 768px;

    /* 缩略图默认宽度 */
    --thumb-w: 200px;
}
```

- [ ] **步骤 2：将 styles.css 中各处的 magic number 替换为 var() 引用**

具体替换模式（仅在明确对应时替换，不要强制替换所有颜色值）：

1. 替换 `.right-zoom-sidebar` 的背景色 `rgba(30, 30, 30, 0.85)` → `var(--color-bg-overlay)`
2. 替换 `.left-action-sidebar` 的背景色 `rgba(30, 30, 30, 0.85)` → `var(--color-bg-overlay)`
3. 替换 `.thumb-zoom-controls input[type="range"]::-webkit-slider-thumb` 的 `background: #ed2553` → `var(--color-primary)`
4. 替换 `.right-zoom-sidebar input[type="range"]::-webkit-slider-thumb` 的 `background: #ed2553` → `var(--color-primary)`
5. 替换 `.left-action-sidebar .sidebar-btn.btn-primary` 的 `color: #ed2553` → `var(--color-primary)`
6. 替换 `.zoom-reset-btn` 的 `color: #888` → `var(--color-text-muted)`
7. 替换 `.zoom-reset-btn:hover` 的 `border-color: #888` → `var(--color-text-muted)`
8. 替换 `.left-action-sidebar .sidebar-btn` 的 `color: #ccc` → `var(--color-text-secondary)`
9. 替换 `.right-zoom-sidebar .zoom-value` 的 `color: #ccc` → `var(--color-text-secondary)`
10. 替换 `.page-jump .jump-input` 的 `border: 1px solid #555` → `border: 1px solid var(--color-border)`

- [ ] **步骤 3：验证 CSS 无语法错误**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./cmd/server/view/...
```

预期：编译通过（Go embed 会检测 CSS 文件存在）。

- [ ] **步骤 4：Commit**

```bash
git add cmd/server/view/static/custom/css/styles.css
git commit -m "feat(css): 提炼 CSS 自定义属性，替换 magic number"
```

---

### 任务 3：JS 代码模块化拆分

**文件：**
- 创建：`cmd/server/view/static/custom/js/modules/toast.js`
- 创建：`cmd/server/view/static/custom/js/modules/loading-manager.js`
- 创建：`cmd/server/view/static/custom/js/modules/optimistic-updater.js`
- 创建：`cmd/server/view/static/custom/js/modules/autocomplete.js`
- 创建：`cmd/server/view/static/custom/js/modules/modal.js`
- 创建：`cmd/server/view/static/custom/js/modules/gallery-actions.js`
- 创建：`cmd/server/view/static/custom/js/modules/tag-editor.js`
- 创建：`cmd/server/view/static/custom/js/modules/tag-actions.js`
- 创建：`cmd/server/view/static/custom/js/modules/thumbnail-zoom.js`
- 创建：`cmd/server/view/static/custom/js/modules/tag-aligner.js`
- 创建：`cmd/server/view/static/custom/js/modules/related-tags.js`
- 创建：`cmd/server/view/static/custom/js/modules/navigation.js`
- 修改：`cmd/server/view/static/custom/js/scripts.js`（改为入口文件）

策略：每个模块文件是一个 IIFE（Immediately Invoked Function Expression），挂在 `window` 上。这种方式兼容现有模板中 `onclick` 属性调用全局函数的方式。

**每个模块的职责划分：**

- [ ] **步骤 1：创建 `modules/toast.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.showToast = function showToast(message, opts) {
    var options = opts || {};
    var type = options.type || 'info';
    var duration = typeof options.duration === 'number' ? options.duration : 5000;
    var dismissible = options.dismissible !== false;
    if (typeof message === 'object' && message !== null) {
      try { message = JSON.stringify(message); } catch (e) {}
    }
    var icons = { success: '✅', error: '❌', info: 'ℹ️', warning: '⚠️' };
    var icon = icons[type] || '';
    var typeClass = 'alert-info';
    if (type === 'success') typeClass = 'alert-success';
    else if (type === 'error') typeClass = 'alert-danger';
    else if (type === 'warning') typeClass = 'alert-warning';
    var container = document.getElementById('messages');
    if (!container) return;
    var alert = document.createElement('div');
    alert.className = 'alert ' + typeClass + ' fade-slide-in open';
    alert.textContent = icon + ' ' + message;
    if (dismissible) {
      alert.style.cursor = 'pointer';
      alert.addEventListener('click', function() {
        if (alert && alert.parentNode) alert.parentNode.removeChild(alert);
      });
    }
    container.appendChild(alert);
    if (duration > 0) {
      setTimeout(function() {
        if (alert && alert.parentNode) alert.parentNode.removeChild(alert);
      }, duration);
    }
  };

  window.showProgressToast = function showProgressToast(message, percent) {
    var container = document.getElementById('messages');
    if (!container) return;
    var existing = document.getElementById('progress-toast');
    if (existing) {
      var bar = existing.querySelector('.progress-bar');
      if (bar) bar.style.width = Math.min(100, Math.max(0, percent || 0)) + '%';
      var msg = existing.querySelector('.progress-msg');
      if (msg) msg.textContent = message;
      return;
    }
    var toast = document.createElement('div');
    toast.id = 'progress-toast';
    toast.className = 'alert alert-info fade-slide-in open';
    toast.style.cssText = 'padding: 8px 12px;';
    toast.innerHTML = '<div class="progress-msg">' + message + '</div>' +
      '<div style="margin-top:4px;height:4px;background:#444;border-radius:2px;overflow:hidden;">' +
      '<div class="progress-bar" style="width:' + Math.min(100, Math.max(0, percent || 0)) + '%;height:100%;background:var(--color-success, #4CAF50);transition:width 0.3s;"></div></div>';
    container.appendChild(toast);
  };
})();
```

- [ ] **步骤 2：创建 `modules/loading-manager.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.LoadingManager = {
    start: function(btn) {
      if (!btn || btn.dataset.loading) return;
      btn.dataset.loading = 'true';
      btn.dataset.origHTML = btn.innerHTML;
      btn.classList.add('btn-loading');
      btn.disabled = true;
    },
    done: function(btn) {
      if (!btn) return;
      delete btn.dataset.loading;
      btn.classList.remove('btn-loading', 'btn-error');
      btn.disabled = false;
    },
    error: function(btn) {
      if (!btn) return;
      delete btn.dataset.loading;
      btn.classList.remove('btn-loading');
      btn.classList.add('btn-error');
      btn.disabled = false;
      setTimeout(function() {
        if (btn) btn.classList.remove('btn-error');
      }, 400);
    }
  };
})();
```

- [ ] **步骤 3：创建 `modules/optimistic-updater.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.OptimisticUpdater = {
    optimisticToggle: function(btn, activeClass, inactiveClass) {
      var wasActive = btn.classList.contains(activeClass);
      var rollbackState = { activeClass: activeClass, inactiveClass: inactiveClass, wasActive: wasActive };
      btn.classList.remove(activeClass, inactiveClass);
      btn.classList.add(wasActive ? inactiveClass : activeClass);
      return {
        rollback: function() {
          btn.classList.remove(activeClass, inactiveClass);
          btn.classList.add(rollbackState.wasActive ? activeClass : inactiveClass);
        },
        wasActive: wasActive
      };
    },
    refreshContainer: function(url, containerSelector, renderFn) {
      var container = document.querySelector(containerSelector);
      if (!container) return Promise.reject('Container not found: ' + containerSelector);
      return fetch(url, { credentials: 'include' })
        .then(function(r) {
          if (!r.ok) throw new Error('HTTP ' + r.status);
          return r.json();
        })
        .then(function(data) {
          if (renderFn && typeof renderFn === 'function') renderFn(container, data);
          return data;
        })
        .catch(function(err) {
          window.showToast('刷新失败: ' + err.message, { type: 'error' });
        });
    }
  };
})();
```

- [ ] **步骤 4：创建 `modules/autocomplete.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.enableAutocompleteKeyboardNav = function enableAutocompleteKeyboardNav(dropdown, onSelect) {
    var selectedIdx = -1;
    function getItems() { return dropdown.querySelectorAll('div'); }
    function highlight(idx) {
      getItems().forEach(function(el, i) {
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
    return function destroy() { dropdown.removeEventListener('keydown', handler); };
  };

  window.bindAutocompleteKeys = function bindAutocompleteKeys(input, dropdown, onEnter) {
    input.addEventListener('keydown', function(e) {
      if (dropdown.style.display === 'none' || !dropdown.children.length) {
        if (e.key === 'Enter' && onEnter) { e.preventDefault(); onEnter(); }
        return;
      }
      var items = dropdown.querySelectorAll('div');
      var selected = dropdown.querySelector('.keyboard-selected');
      var idx = -1;
      if (selected) {
        for (var i = 0; i < items.length; i++) {
          if (items[i] === selected) { idx = i; break; }
        }
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        idx = (idx + 1) % items.length;
        items.forEach(function(el, i) {
          el.classList.toggle('keyboard-selected', i === idx);
          el.style.background = i === idx ? '#444' : 'transparent';
        });
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        idx = (idx - 1 + items.length) % items.length;
        items.forEach(function(el, i) {
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
```

- [ ] **步骤 5：创建 `modules/modal.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

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
        if (document.activeElement === first) { e.preventDefault(); last.focus(); }
      } else {
        if (document.activeElement === last) { e.preventDefault(); first.focus(); }
      }
    }
    document.addEventListener('keydown', trapFocus);

    setTimeout(function() {
      var firstFocusable = wrapper.querySelector(focusableSel);
      if (firstFocusable) firstFocusable.focus();
    }, 50);

    wrapper.addEventListener('click', function(e) {
      if (e.target === wrapper) closeModal(wrapper);
    });

    var escHandler = function(e) {
      if (e.key === 'Escape') closeModal(wrapper);
    };
    wrapper._escHandler = escHandler;
    document.addEventListener('keydown', escHandler);

    wrapper._trapFocus = trapFocus;
    wrapper._prevFocus = prevFocus;
    return wrapper;
  };

  window.closeModal = function closeModal(wrapper) {
    if (wrapper && wrapper.parentNode) {
      if (wrapper._escHandler) document.removeEventListener('keydown', wrapper._escHandler);
      if (wrapper._trapFocus) document.removeEventListener('keydown', wrapper._trapFocus);
      if (wrapper._prevFocus) wrapper._prevFocus.focus();
      wrapper.classList.remove('open');
      wrapper.classList.add('fade-slide-out');
      setTimeout(function() {
        if (wrapper && wrapper.parentNode) wrapper.parentNode.removeChild(wrapper);
      }, 150);
    }
    if (!document.querySelector('.modal-wrapper.open')) {
      document.body.classList.remove('modal-open');
      document.body.style.removeProperty('--scrollbar-width');
    }
  };
})();
```

- [ ] **步骤 6：创建 `modules/gallery-actions.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.findCustomsContainer = function findCustomsContainer() {
    var containers = document.querySelectorAll('.tag-container.field-name');
    for (var c of containers) {
      var text = (c.textContent || '').trim();
      if (text.startsWith('Customs')) return c;
    }
    return null;
  };

  window.addLikeTag = function addLikeTag() {
    var container = findCustomsContainer();
    if (!container) return;
    container.classList.remove('hidden');
    var span = container.querySelector('span.tags');
    if (!span) return;
    if (span.querySelector('.tag-99999')) return;
    var a = document.createElement('a');
    a.href = '/tag/custom/like/';
    a.className = 'tag tag-99999';
    var name = document.createElement('span');
    name.className = 'name';
    name.textContent = 'like';
    var count = document.createElement('span');
    count.className = 'count';
    count.textContent = '1';
    a.appendChild(name);
    a.appendChild(count);
    span.appendChild(a);
  };

  window.removeLikeTag = function removeLikeTag() {
    var container = findCustomsContainer();
    if (!container) return;
    var span = container.querySelector('span.tags');
    if (!span) return;
    var a = span.querySelector('.tag-99999');
    if (a) a.remove();
    if (!span.querySelector('a.tag')) container.classList.add('hidden');
  };

  window.formatError = function formatError(resp) {
    var code = resp && resp.head ? resp.head.code : -1;
    var msg = resp && resp.head ? (resp.head.msg || resp.head.message || '') : '';
    return '[' + code + '] ' + (msg || '请求失败');
  };

  window.highlightInvalidPages = function highlightInvalidPages(indexes) {
    if (!Array.isArray(indexes) || indexes.length === 0) return;
    var container = document.getElementById('thumbnail-container');
    if (!container) return;
    indexes.forEach(function(it) {
      var idx = it.index || it;
      var cid = window._gallery && window._gallery.cid || '';
      var link = container.querySelector('a.gallerythumb[href="/g/' + String(cid) + '/' + String(idx) + '/"]');
      if (link && link.parentElement) link.parentElement.style.outline = '3px solid #e74c3c';
    });
  };

  window.ensureForceArchiveButton = function ensureForceArchiveButton(cid) {
    if (document.getElementById('forceArchiveBtn')) return;
    var sidebar = document.querySelector('.left-action-sidebar');
    if (!sidebar) return;
    var a = document.createElement('a');
    a.id = 'forceArchiveBtn';
    a.href = 'javascript:;';
    a.className = 'sidebar-btn';
    a.innerHTML = '<i class="fa fa-exclamation-triangle"></i><span class="label">强制归档</span>';
    a.onclick = function() { archiveComicForce(cid); };
    sidebar.appendChild(a);
  };

  window.addLikeGroup = function addLikeGroup(cid) {
    var btn = document.getElementById('sidebarLikeBtn');
    if (!btn) btn = document.getElementById('addLikeGroup');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var liked = btn.classList.contains('btn-primary');
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/like');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');

    var toggle = window.OptimisticUpdater.optimisticToggle(btn, 'btn-primary', 'btn-secondary');
    var label = btn.querySelector('.label');
    if (label) label.textContent = liked ? 'Like' : 'Liked';

    xhr.onload = function() {
      window.LoadingManager.done(btn);
      if (xhr.status >= 200 && xhr.status < 300) {
        var detailLikeTag = document.querySelector('.tag-99999');
        if (liked && detailLikeTag) { detailLikeTag.remove(); }
        else if (!liked) { addLikeTag(); }
        window.showToast(liked ? '已取消 Like' : '已添加 Like', { type: 'success' });
      } else {
        toggle.rollback();
        if (label) label.textContent = liked ? 'Liked' : 'Like';
        window.showToast('操作失败', { type: 'error' });
      }
    };
    xhr.onerror = function() {
      window.LoadingManager.done(btn);
      toggle.rollback();
      if (label) label.textContent = liked ? 'Liked' : 'Like';
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send('cid=' + encodeURIComponent(cid));
  };

  window.archiveComic = function archiveComic(cid) {
    var btn = document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive');
    xhr.onload = function() {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('已归档', { type: 'success' });
          btn.innerHTML = '<i class="fa fa-undo"></i><span class="label">恢复</span>';
          btn.onclick = function() { restoreComic(cid); };
          btn.id = 'sidebarArchiveBtn';
        } else {
          window.showToast(window.formatError(resp), { type: 'error' });
          if (resp.head.code === -1001) {
            var invalids = (resp.body && resp.body.invalid_images) || [];
            if (!invalids.length && window._gallery && window._gallery.images && Array.isArray(window._gallery.images.pages)) {
              window._gallery.images.pages.forEach(function(p, i) {
                if (p && p.status === false) invalids.push({ index: i + 1 });
              });
            }
            window.highlightInvalidPages(invalids);
            window.ensureForceArchiveButton(cid);
            window.showToast('检测到异常图片，建议先“修复漫画状态”，或使用“强制归档”', { type: 'info' });
          }
        }
      } else {
        window.LoadingManager.error(btn);
        window.showToast(xhr.responseText || ('请求失败: ' + xhr.status), { type: 'error' });
      }
    };
    xhr.onerror = function() {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send();
  };

  window.archiveComicForce = function archiveComicForce(cid) {
    var btn = document.getElementById('forceArchiveBtn') || document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive?force=true');
    xhr.onload = function() {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('已强制归档', { type: 'success' });
          btn.innerHTML = '<i class="fa fa-undo"></i><span class="label">恢复</span>';
          btn.onclick = function() { restoreComic(cid); };
        } else {
          window.showToast(window.formatError(resp), { type: 'error' });
        }
      } else {
        window.LoadingManager.error(btn);
        window.showToast(xhr.responseText || ('请求失败: ' + xhr.status), { type: 'error' });
      }
    };
    xhr.onerror = function() {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send();
  };

  window.restoreComic = function restoreComic(cid) {
    var btn = document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/restore');
    xhr.onload = function() {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('已恢复', { type: 'success' });
          btn.innerHTML = '<i class="fa fa-archive"></i><span class="label">归档</span>';
          btn.onclick = function() { archiveComic(cid); };
          btn.id = 'sidebarArchiveBtn';
        } else {
          window.showToast(window.formatError(resp), { type: 'error' });
        }
      } else {
        window.LoadingManager.error(btn);
        window.showToast(xhr.responseText || ('请求失败: ' + xhr.status), { type: 'error' });
      }
    };
    xhr.onerror = function() {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    xhr.send();
  };

  window.verifyComic = function verifyComic(cid) {
    var btn = document.getElementById('sidebarFixBtn');
    if (!btn || btn.dataset.loading) return;
    window.LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/verify');
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function() {
      window.LoadingManager.done(btn);
      if (xhr.status == 200) {
        var resp = JSON.parse(xhr.responseText);
        if (resp.head.code === 0) {
          window.showToast('修复任务已启动', { type: 'success' });
        } else {
          window.showToast(window.formatError(resp), { type: 'error' });
        }
      } else {
        window.LoadingManager.error(btn);
        window.showToast(xhr.responseText || ('请求失败: ' + xhr.status), { type: 'error' });
      }
    };
    xhr.onerror = function() {
      window.LoadingManager.error(btn);
      window.showToast('网络错误', { type: 'error' });
    };
    var body = { id: String(cid), autoFix: true, maxWorkers: 1 };
    xhr.send(JSON.stringify(body));
  };
})();
```

- [ ] **步骤 7：创建 `modules/tag-editor.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.openTagEditor = function openTagEditor(cid) {
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/api/comic/getComicInfo');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function() {
      if (xhr.status !== 200) { window.showToast('获取漫画信息失败', { type: 'error' }); return; }
      try {
        var resp = JSON.parse(xhr.responseText);
        buildTagEditorModal(cid, resp.body.tags || []);
      } catch (e) { window.showToast('解析响应失败', { type: 'error' }); }
    };
    xhr.onerror = function() { window.showToast('网络错误', { type: 'error' }); };
    xhr.send('cid=' + encodeURIComponent(cid));
  };

  function dedupKey(t) {
    return t.type + ':' + (t.id || t.name);
  }

  function buildTagEditorModal(cid, currentTags) {
    var added = [];
    var removed = [];
    var tagTypes = [
      {value: 'parody', label: 'Parodies'}, {value: 'character', label: 'Characters'},
      {value: 'tag', label: 'Tags'}, {value: 'artist', label: 'Artists'},
      {value: 'group', label: 'Groups'}, {value: 'language', label: 'Languages'},
      {value: 'category', label: 'Categories'}, {value: 'custom', label: 'Customs'}
    ];

    var tagsContainer = document.createElement('div');
    tagsContainer.style.cssText = 'margin-bottom: 10px; display: flex; flex-wrap: wrap; gap: 5px;';

    function renderTags() {
      tagsContainer.innerHTML = '';
      var displayTags = [];
      currentTags.forEach(function(t) {
        var key = dedupKey(t);
        if (!removed.some(function(r) { return dedupKey(r) === key; })) displayTags.push(t);
      });
      added.forEach(function(t) { displayTags.push(t); });
      displayTags.forEach(function(t) {
        var chip = document.createElement('span');
        chip.className = 'tag tag-' + (t.id || 0);
        chip.style.cssText = 'display: inline-flex; align-items: center; margin: 2px; padding: 2px 8px; border-radius: 3px; background: #2a2a2a;';
        chip.innerHTML = '<span class="name" style="margin-right:4px">[' + t.type + '] ' + t.name + '</span>';
        var delBtn = document.createElement('a');
        delBtn.href = 'javascript:;';
        delBtn.textContent = 'x';
        delBtn.style.cssText = 'color: #e74c3c; text-decoration: none; font-weight: bold;';
        delBtn.onclick = function() {
          var key = dedupKey(t);
          var addedIdx = -1;
          for (var i = 0; i < added.length; i++) { if (dedupKey(added[i]) === key) { addedIdx = i; break; } }
          if (addedIdx >= 0) added.splice(addedIdx, 1);
          else removed.push(t);
          renderTags();
        };
        chip.appendChild(delBtn);
        tagsContainer.appendChild(chip);
      });
    }
    renderTags();

    // ...（为简洁省略表单构建部分，完整代码从原 scripts.js 原样复制）
    // 实际的 tag-editor.js 应包含 buildTagEditorModal 的全部表单逻辑
    // 包括 Existing/New 双模式、自动补全、Save 按钮等

    // 占位提示：实际实现需要包含原 scripts.js 中 buildTagEditorModal 的完整代码
    console.log('tag-editor module loaded');
  }
})();
```

> **注意：** 步骤 7-11 的模块文件内容较多，实际实现时直接从原 `scripts.js` 中提取对应函数代码，封装到各自 IIFE 中。每个模块文件保持 < 200 行。

- [ ] **步骤 8：创建 `modules/thumbnail-zoom.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
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

    var saved = localStorage.getItem('thumbZoom');
    if (saved) {
      var v = parseInt(saved, 10);
      if (!isNaN(v) && v >= 60 && v <= 1200) slider.value = v;
    }

    function applyZoom(val) {
      container.style.setProperty('--thumb-w', val + 'px');
      if (zoomValue) zoomValue.textContent = val;
      localStorage.setItem('thumbZoom', String(val));
    }
    applyZoom(parseInt(slider.value, 10));

    slider.addEventListener('input', function() { applyZoom(parseInt(this.value, 10)); });
    if (zoomInBtn) zoomInBtn.addEventListener('click', function() {
      var v = Math.min(1200, parseInt(slider.value, 10) + 20);
      slider.value = v; applyZoom(v);
    });
    if (zoomOutBtn) zoomOutBtn.addEventListener('click', function() {
      var v = Math.max(60, parseInt(slider.value, 10) - 20);
      slider.value = v; applyZoom(v);
    });
    if (zoomResetBtn) zoomResetBtn.addEventListener('click', function() {
      slider.value = 1200; applyZoom(1200);
    });
    document.querySelectorAll('.preset-btn').forEach(function(btn) {
      btn.addEventListener('click', function() {
        var val = parseInt(this.getAttribute('data-zoom'), 10);
        if (!isNaN(val) && val >= 60 && val <= 1200) { slider.value = val; applyZoom(val); }
      });
    });
  };

  window.toggleLargeMode = function toggleLargeMode() {
    var container = document.getElementById('thumbnail-container');
    var zoomSidebar = document.getElementById('zoomSidebar');
    var btn = document.getElementById('sidebarLargeToggle');
    if (!container || !btn) return;

    var isLarge = container.classList.toggle('large-mode');
    if (isLarge) {
      if (zoomSidebar) zoomSidebar.style.display = '';
      container.querySelectorAll('.thumb-container').forEach(function(el) {
        el.classList.remove('thumb-container');
        el.classList.add('thumb-container-large');
      });
      container.querySelectorAll('.thumb-container-large img').forEach(function(img) {
        img.removeAttribute('width');
        img.removeAttribute('height');
      });
      btn.innerHTML = '<i class="fa fa-compress"></i><span class="label">退出大图</span>';
      localStorage.setItem('largeMode', 'true');
    } else {
      if (zoomSidebar) zoomSidebar.style.display = 'none';
      container.querySelectorAll('.thumb-container-large').forEach(function(el) {
        el.classList.remove('thumb-container-large');
        el.classList.add('thumb-container');
      });
      container.querySelectorAll('.thumb-container img').forEach(function(img) {
        img.setAttribute('width', '200');
        img.setAttribute('height', '282');
      });
      btn.innerHTML = '<i class="fa fa-expand"></i><span class="label">大图模式</span>';
      localStorage.setItem('largeMode', 'false');
    }
  };

  window.initLargeModeToggle = function initLargeModeToggle() {
    var btn = document.getElementById('sidebarLargeToggle');
    if (!btn) return;
    var container = document.getElementById('thumbnail-container');
    if (!container) return;
    var hasLarge = container.querySelectorAll('.thumb-container-large').length > 0;
    if (hasLarge) {
      container.classList.add('large-mode');
      btn.innerHTML = '<i class="fa fa-compress"></i><span class="label">退出大图</span>';
    }
    var saved = localStorage.getItem('largeMode');
    if (saved === 'false' && hasLarge) {
      toggleLargeMode();
    } else if (saved === 'true' && !hasLarge && document.getElementById('zoomSidebar')) {
      toggleLargeMode();
    }
  };

  window.toggleMobileZoom = function toggleMobileZoom() {
    var sidebar = document.getElementById('zoomSidebar');
    if (sidebar) sidebar.classList.toggle('mobile-open');
  };
})();
```

- [ ] **步骤 9：创建 `modules/navigation.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.jumpToPage = function jumpToPage(input, baseUrl, query) {
    var page = parseInt(input.value, 10);
    var max = parseInt(input.max, 10);
    if (isNaN(page) || page < 1 || page > max) {
      window.showToast('页码应在 1 ~ ' + max + ' 之间', { type: 'warning' });
      return;
    }
    var url = baseUrl + '?page=' + page;
    if (query) url += '&q=' + encodeURIComponent(query);
    window.location.href = url;
  };

  window.jumpToTagPage = function jumpToTagPage(input, baseUrl, sortType) {
    var page = parseInt(input.value, 10);
    var max = parseInt(input.max, 10);
    if (isNaN(page) || page < 1 || page > max) {
      window.showToast('页码应在 1 ~ ' + max + ' 之间', { type: 'warning' });
      return;
    }
    var url = baseUrl + '?page=' + page;
    if (sortType === 1) url += '&sortType=popular';
    window.location.href = url;
  };

  window.initKeyboardShortcuts = function initKeyboardShortcuts() {
    // / 聚焦搜索（已存在，保持兼容）
    // Esc 搜索框失焦（已存在，保持兼容）
    // ← → 翻页（仅在单页查看模式 gallery picture page 有效）
    var imgViewer = document.querySelector('.gallery-picture-viewer');
    if (imgViewer) {
      document.addEventListener('keydown', function pageNav(e) {
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.isContentEditable) return;
        if (e.key === 'ArrowLeft') {
          var prevLink = document.querySelector('a.previous-page');
          if (prevLink) { e.preventDefault(); window.location.href = prevLink.href; }
        } else if (e.key === 'ArrowRight') {
          var nextLink = document.querySelector('a.next-page');
          if (nextLink) { e.preventDefault(); window.location.href = nextLink.href; }
        }
      });
    }

    // L 点赞（仅在详情页有 sidebarLikeBtn 时生效）
    var likeBtn = document.getElementById('sidebarLikeBtn') || document.getElementById('addLikeGroup');
    if (likeBtn) {
      document.addEventListener('keydown', function likeShortcut(e) {
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA' || e.target.isContentEditable) return;
        if (e.key === 'l' || e.key === 'L') {
          e.preventDefault();
          likeBtn.click();
        }
      });
    }
  };
})();
```

- [ ] **步骤 10：创建 `modules/tag-actions.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.toggleLikeTag = function toggleLikeTag(type, name, id) {
    var btn = document.getElementById('toggleLikeTag');
    if (!btn) return;
    var liked = btn.classList.contains('btn-primary');
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/likeTag');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function() {
      if (xhr.status >= 200 && xhr.status < 300) {
        if (liked) {
          btn.classList.remove('btn-primary');
          btn.classList.add('btn-secondary');
          var link = document.getElementById('currentTagLink');
          if (link) link.classList.remove('tag-like');
        } else {
          btn.classList.remove('btn-secondary');
          btn.classList.add('btn-primary');
          var link = document.getElementById('currentTagLink');
          if (link) link.classList.add('tag-like');
        }
      } else {
        console.error('likeTag request failed:', xhr.status, xhr.responseText);
      }
    };
    xhr.onerror = function() { console.error('likeTag request network error'); };
    var params = 'type=' + encodeURIComponent(type);
    if (id && id > 0) params += '&id=' + encodeURIComponent(id);
    else if (name) params += '&name=' + encodeURIComponent(name);
    xhr.send(params);
  };

  window.rebuildTagsSection = function rebuildTagsSection(tags) {
    var container = document.querySelector('#tags');
    if (!container) return;
    var groups = {};
    var typeOrder = ['parody','character','tag','artist','group','language','category','custom'];
    var typeLabels = {
      'parody': 'Parodies', 'character': 'Characters', 'tag': 'Tags',
      'artist': 'Artists', 'group': 'Groups', 'language': 'Languages',
      'category': 'Categories', 'custom': 'Customs'
    };
    tags.forEach(function(t) {
      if (!groups[t.type]) groups[t.type] = [];
      groups[t.type].push(t);
    });
    var html = '';
    typeOrder.forEach(function(type) {
      var list = groups[type];
      if (!list || list.length === 0) return;
      html += '<div class="tag-container field-name">' + typeLabels[type] + ': <span class="tags">';
      list.forEach(function(t) {
        html += '<a href="/tag/' + encodeURIComponent(t.type) + '/' + encodeURIComponent(t.name.toLowerCase().replace(/\s+/g, '-')) + '/" class="tag tag-' + (t.id || 0) + '">' +
          '<span class="name">' + t.name + '</span><span class="count">' + (t.count || 1) + '</span></a>';
      });
      html += '</span></div>';
    });
    container.innerHTML = html;
  };
})();
```

- [ ] **步骤 11：创建 `modules/tag-aligner.js`**（从原 scripts.js 提取 openTagAligner、buildTagAlignerModal 到 IIFE）
- [ ] **步骤 12：创建 `modules/related-tags.js`**（从原 scripts.js 提取 loadRelatedTags、openTagRelationManager 到 IIFE）
- [ ] **步骤 13：创建 `modules/skeleton.js`**

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */
(function() {
  'use strict';

  window.showSkeleton = function showSkeleton(containerId) {
    var container = document.getElementById(containerId);
    if (!container) return;
    container.innerHTML = '';
    for (var i = 0; i < 6; i++) {
      var skeleton = document.createElement('div');
      skeleton.className = 'skeleton-card';
      skeleton.style.cssText = 'width: 200px; height: 300px; background: linear-gradient(90deg, #2a2a2a 25%, #333 50%, #2a2a2a 75%); background-size: 200% 100%; animation: skeleton-shimmer 1.5s infinite; border-radius: 4px;';
      container.appendChild(skeleton);
    }
  };

  window.hideSkeleton = function hideSkeleton(containerId) {
    var container = document.getElementById(containerId);
    if (!container) return;
    container.querySelectorAll('.skeleton-card').forEach(function(el) { el.remove(); });
  };

  window.showEmptyState = function showEmptyState(containerId, message) {
    var container = document.getElementById(containerId);
    if (!container) return;
    container.innerHTML = '<div class="empty-state" style="text-align:center;padding:40px 20px;color:var(--color-text-muted, #888);">' +
      '<i class="fa fa-inbox" style="font-size:48px;display:block;margin-bottom:12px;"></i>' +
      '<p>' + (message || '暂无数据') + '</p></div>';
  };
})();
```

- [ ] **步骤 14：重写 `scripts.js` 为入口文件**

将原有的 ~1518 行代码替换为简洁的入口文件：

```js
/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0 *
 * Entry point — loads all module files and initializes page-specific logic.
 * Module files are loaded separately via <script> tags in head.tpl.
 */

// 全局搜索快捷键：按 / 聚焦搜索框
document.addEventListener('keydown', function(e) {
    var tag = e.target.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target.isContentEditable) return;
    if (e.key === '/') {
        e.preventDefault();
        var searchInput = document.querySelector('input[type="search"]');
        if (searchInput) { searchInput.focus(); searchInput.select(); }
    }
});

// 搜索框按 Esc 失焦
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && e.target && e.target.type === 'search') {
        e.target.blur();
    }
});

// 首页自动聚焦搜索框
(function() {
    var path = window.location.pathname;
    if (path === '/' || path === '/search/') {
        var searchInput = document.querySelector('input[type="search"]');
        if (searchInput && !searchInput.value) {
            setTimeout(function() { searchInput.focus(); }, 300);
        }
    }
})();

// 页面初始化
function initGalleryPage() {
    if (typeof initThumbnailZoom === 'function') initThumbnailZoom();
    if (typeof initLargeModeToggle === 'function') initLargeModeToggle();
    if (typeof initKeyboardShortcuts === 'function') initKeyboardShortcuts();
}
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initGalleryPage);
} else {
    initGalleryPage();
}
```

- [ ] **步骤 15：更新 `head.tpl` 添加模块 script 标签**

在 `cmd/server/view/static/tpl/head.tpl` 的 `{{define "head.common.tpl"}}` 中，将单个 `scripts.js` 引用替换为模块加载顺序：

```html
    <script src="/static/custom/js/modules/toast.js"></script>
    <script src="/static/custom/js/modules/loading-manager.js"></script>
    <script src="/static/custom/js/modules/optimistic-updater.js"></script>
    <script src="/static/custom/js/modules/autocomplete.js"></script>
    <script src="/static/custom/js/modules/modal.js"></script>
    <script src="/static/custom/js/modules/skeleton.js"></script>
    <script src="/static/custom/js/modules/gallery-actions.js"></script>
    <script src="/static/custom/js/modules/tag-actions.js"></script>
    <script src="/static/custom/js/modules/tag-editor.js"></script>
    <script src="/static/custom/js/modules/tag-aligner.js"></script>
    <script src="/static/custom/js/modules/related-tags.js"></script>
    <script src="/static/custom/js/modules/thumbnail-zoom.js"></script>
    <script src="/static/custom/js/modules/navigation.js"></script>
    <script src="/static/custom/js/scripts.js"></script>
```

同时保留 tag_relation.js 引用（若存在）。

- [ ] **步骤 16：验证构建通过**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./cmd/server/view/...
```

预期：无编译错误。注意 Go embed `//go:embed static/*` 会自动包含新创建的模块文件。

- [ ] **步骤 17：Commit**

```bash
git add cmd/server/view/static/custom/js/
git add cmd/server/view/static/tpl/head.tpl
git commit -m "refactor(js): 将 scripts.js 拆分为 13 个功能模块"
```

---

### 任务 4：键盘快捷键统一

**文件：**
- 创建：已包含在 T3 的 `modules/navigation.js` 中

- [ ] **步骤 1：在详情页模板中添加图片查看器的翻页链接 class**

单页查看模板（`gallery_picture.go` 对应的 `gallery_picture.tpl`）中，确保上一页/下一页链接有 class `previous-page` / `next-page`：

```html
<!-- 在图片查看器模板中添加 class -->
<a class="previous-page" href="/g/{{.CID}}/{{.PrevPage}}/">‹ 上一页</a>
<a class="next-page" href="/g/{{.CID}}/{{.NextPage}}/">下一页 ›</a>
```

（如果已有 `previous`/`next` class，则导航.js 中的查询选择器需要对应调整）

- [ ] **步骤 2：提交键盘快捷键相关变更**

```bash
git add cmd/server/view/static/custom/js/modules/navigation.js
git commit -m "feat(ui): 添加 ← → 翻页和 L 点赞键盘快捷键"
```

---

### 任务 5：空状态/骨架屏 + 移动端适配

**文件：**
- 修改：`cmd/server/view/static/custom/css/styles.css`
- 修改：`cmd/server/view/static/tpl/gallery_detail.tpl`
- 修改：`cmd/server/view/static/tpl/index.tpl`

- [ ] **步骤 1：在 styles.css 末尾添加骨架屏动画和空状态样式**

```css
/* ===== 骨架屏 Skeleton ===== */

@keyframes skeleton-shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
}

.skeleton-card {
    width: 200px;
    height: 300px;
    background: linear-gradient(90deg, var(--color-bg-raised) 25%, #333 50%, var(--color-bg-raised) 75%);
    background-size: 200% 100%;
    animation: skeleton-shimmer 1.5s infinite;
    border-radius: var(--radius-sm);
}

.skeleton-text {
    height: 14px;
    background: linear-gradient(90deg, var(--color-bg-raised) 25%, #333 50%, var(--color-bg-raised) 75%);
    background-size: 200% 100%;
    animation: skeleton-shimmer 1.5s infinite;
    border-radius: 2px;
    margin-bottom: 8px;
}

/* ===== 空状态 ===== */

.empty-state {
    text-align: center;
    padding: 60px 20px;
    color: var(--color-text-muted);
}

.empty-state i {
    font-size: 48px;
    display: block;
    margin-bottom: 16px;
    opacity: 0.5;
}

.empty-state p {
    font-size: 15px;
    margin: 0;
}
```

- [ ] **步骤 2：为 index.tpl 的 Popular Now 和 New Uploads 容器添加骨架屏占位**

在 `index.tpl` 中，在 `index-popular` 和 `index-container` 内部添加骨架屏占位元素，在 JS 加载完成后通过 `hideSkeleton` 移除：

```html
    <div class="container index-container index-popular">
        <h2><i class="fa fa-fire color-icon"></i> Popular Now</h2>
        <div class="skeleton-wrapper" id="skeleton-popular">
            <!-- Service-side rendered skeletons: hidden when content loads -->
        </div>
{{range $index, $detail := .PopularNow}}
        <!-- 原有 gallery divs... -->
{{end}}
    </div>
```

- [ ] **步骤 3：在 gallery_detail.tpl 的 thumbnail-container 添加骨架屏**

在 `gallery_detail.tpl` 第 144 行的 `#thumbnail-container` 内，在 `.thumbs` 上方或内部添加骨架屏：

```html
        <div class="container{{if .EnableLarge}} with-sidebars{{end}}" id="thumbnail-container">
            <div class="skeleton-wrapper" id="skeleton-thumbnails" style="display:flex;flex-wrap:wrap;gap:4px;padding:10px;">
                <!-- 6 个骨架卡片由 JS 在页面加载时填充 -->
            </div>
            <div class="thumbs" style="display:none;">
            <!-- 原有 thumbs content... -->
            </div>
        </div>
```

- [ ] **步骤 4：验证构建通过**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./cmd/server/view/...
```

预期：无编译错误。

- [ ] **步骤 5：Commit**

```bash
git add cmd/server/view/static/custom/css/styles.css
git add cmd/server/view/static/tpl/index.tpl
git add cmd/server/view/static/tpl/gallery_detail.tpl
git commit -m "feat(ui): 添加骨架屏和空状态占位"
```

---

### 任务 6：移动端完美适配

**文件：**
- 修改：`cmd/server/view/static/custom/css/styles.css`

- [ ] **步骤 1：优化移动端底部操作栏触屏交互**

在 `styles.css` 的 `@media (max-width: 768px)` 段中，增强 `.left-action-sidebar` 触屏体验：

```css
@media (max-width: 768px) {
    /* ... 已有样式 ... */

    /* 增强触屏交互 */
    .left-action-sidebar .sidebar-btn {
        min-width: var(--sidebar-btn-min-width-mobile);
        padding: 8px 4px;
        -webkit-tap-highlight-color: rgba(237, 37, 83, 0.3);
        touch-action: manipulation;
    }

    /* 缩略图点击区域放大 */
    .thumb-container a.gallerythumb {
        display: block;
        padding: 2px;
        min-height: 44px; /* Apple HIG 最小触控目标 */
    }

    /* 移动端 Toast 堆叠位置调整（避免被底部栏遮挡） */
    #messages {
        top: auto;
        bottom: 70px;
        right: 8px;
        left: 8px;
        max-width: none;
    }

    /* 移动端缩略图容器边距 */
    #thumbnail-container {
        padding-bottom: 60px; /* 为底部操作栏留空间 */
    }
}
```

- [ ] **步骤 2：验证构建通过**

```bash
cd D:\workdir\leon\cocomhub\cocom
go build ./cmd/server/view/...
```

预期：无编译错误。

- [ ] **步骤 3：Commit**

```bash
git add cmd/server/view/static/custom/css/styles.css
git commit -m "fix(ui): 移动端触屏交互优化 — 增大点击区域、Toast 堆叠调整、底部安全间距"
```

---

### 任务 7：小版本 API 对齐（后端）

**文件：**
- 修改：`cmd/server/handler/*.go`（需确认实际电路）

- [ ] **步骤 1：调研当前 API 响应格式不一致之处**

```bash
cd D:\workdir\leon\cocomhub\cocom
grep -rn 'c\.JSON\|c\.AbortWithStatusJSON\|gin\.H{' cmd/server/ | head -30
```

观察哪些 handler 使用了裸 `gin.H{}`（不含 `head/body` 包装），哪些使用了 `pkg/httpwrap` 的统一响应。

- [ ] **步骤 2：为缺少 head/body 包装的端点点添加统一格式**

针对调研发现的不一致端点，添加统一的 JSON 响应结构。例如：

```go
// 统一错误响应
httpwrap.GinErrJSON(c, http.StatusBadRequest, -1, "invalid parameter")
// 统一成功响应
httpwrap.GinSuccessJSON(c, data)
```

- [ ] **步骤 3：运行测试确认不破坏现有行为**

```bash
cd D:\workdir\leon\cocomhub\cocom
go test ./cmd/server/... -tags=memory_storage_integration -count=1
```

预期：所有测试通过。

- [ ] **步骤 4：Commit**

```bash
git add cmd/server/
git commit -m "fix(api): 统一 API 响应格式，前端乐观更新路径对齐"
```

---

## 验收检查清单

实现完成后，对照 M1 验收标准逐项验证：

| 验收标准 | 验证方式 |
|----------|----------|
| 导航栏上不再有大图模式切换入口 | 打开任意详情页，检查导航栏无「大图模式」链接 |
| 所有 Toast 弹窗在 3s 后自动消失，无堆积遮挡 | 触发 Like/归档操作，观察 Toast 显示与消失 |
| 键盘 `← →` 在单页查看时触发翻页 | 打开 `/g/:cid/:no` 页面，按左右方向键 |
| 键盘 `L` 触发点赞 | 打开详情页，按 `L` 键，观察 Like 按钮切换 |
| `scripts.js` 拆分为不超过 300 行/模块的文件 | `wc -l modules/*.js` 检查行数 |
| 375px / 768px / 1440px 三个视口下操作栏和缩放栏均可用 | Chrome DevTools 切换设备仿真 |
| `make test` 通过率 100% | `cd cocom && make test` |