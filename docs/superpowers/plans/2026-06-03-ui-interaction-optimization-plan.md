# UI 交互体验优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 系统性地优化 cocom Web 端交互体验，覆盖异步反馈、无刷新更新、搜索体验、弹窗交互、缩略图缩放五个模块，采用双侧边栏重构操作布局。

**Architecture:** 不修改后端 Go 代码，纯前端优化。基础设施层（LoadingManager、Toast、OptimisticUpdater、键盘导航）在 `custom/js/scripts.js` 中新增，各交互模块在现有函数基础上集成基础设施。CSS 全量新增到 `custom/css/styles.css`。模板层（`.tpl`）做布局结构调整。

**Tech Stack:** 原生 JavaScript（沿用项目现有 XHR/fetch 混用风格，不引入新依赖）、CSS3、Go Gin templates

---

## 文件结构映射

| 文件 | 职责 | 改动量 |
|------|------|--------|
| `custom/css/styles.css` | 所有新增交互样式：loading spinner、shake、侧边栏、竖向 slider、page jump、modal 增强 | 新增 ~150 行 |
| `custom/js/scripts.js` | LoadingManager、Toast 升级、OptimisticUpdater、键盘导航、所有交互函数重写 | 新增/重写 ~400 行 |
| `gallery_detail.tpl` | 操作按钮迁移到左侧边栏、新增右侧缩放侧边栏、缩略图容器边距 | 中等修改 |
| `index.tpl` | 分页区域增加跳页控件、enableLarge 改名、搜索框 placeholder 更新 | 小幅修改 |
| `tag_list_result.tpl` | 分页区域增加跳页控件 | 小幅修改 |
| `head.tpl` | navigation.tpl 中 enableLarge 链接改名 | 小幅修改 |

---

### Task 1: CSS — 交互基础设施样式

**Files:**
- Modify: `cmd/server/view/static/custom/css/styles.css`

- [ ] **Step 1: 新增按钮 loading 与 shake 动画**

在 `styles.css` 末尾追加：

```css
/* ===== Loading Spinner ===== */
.btn-loading {
  position: relative;
  pointer-events: none;
  opacity: 0.7;
}
.btn-loading::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 14px;
  height: 14px;
  margin: -7px 0 0 -7px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: btn-spin 0.6s linear infinite;
}
@keyframes btn-spin {
  to { transform: rotate(360deg); }
}

/* ===== Shake 动画（错误反馈） ===== */
.btn-error {
  animation: shake 0.3s ease-in-out;
}
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-4px); }
  40% { transform: translateX(4px); }
  60% { transform: translateX(-3px); }
  80% { transform: translateX(3px); }
}

/* ===== Toast 堆叠增强 ===== */
#messages {
  position: fixed;
  top: 60px;
  right: 16px;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-width: 360px;
}
#messages .alert {
  margin: 0;
  box-shadow: 0 2px 8px rgba(0,0,0,0.3);
}
```

- [ ] **Step 2: 新增左侧操作侧边栏样式**

```css
/* ===== 左侧操作侧边栏 ===== */
.left-action-sidebar {
  position: fixed;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  z-index: 500;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 4px;
  background: rgba(30,30,30,0.85);
  backdrop-filter: blur(6px);
  border-radius: 0 8px 8px 0;
  box-shadow: 2px 0 8px rgba(0,0,0,0.3);
}
.left-action-sidebar .sidebar-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 8px 6px;
  min-width: 52px;
  font-size: 11px;
  border: none;
  background: transparent;
  color: #ccc;
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.15s, color 0.15s;
  text-decoration: none;
}
.left-action-sidebar .sidebar-btn:hover {
  background: rgba(255,255,255,0.1);
  color: #fff;
}
.left-action-sidebar .sidebar-btn.btn-primary {
  color: #ed2553;
}
.left-action-sidebar .sidebar-btn i {
  font-size: 18px;
}
.left-action-sidebar .sidebar-btn .label {
  font-size: 10px;
  white-space: nowrap;
}
```

- [ ] **Step 3: 新增右侧缩放侧边栏样式（含竖向 Slider）**

```css
/* ===== 右侧缩放侧边栏 ===== */
.right-zoom-sidebar {
  position: fixed;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  z-index: 500;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 12px 8px;
  background: rgba(30,30,30,0.85);
  backdrop-filter: blur(6px);
  border-radius: 8px;
  box-shadow: -2px 0 8px rgba(0,0,0,0.3);
}
.right-zoom-sidebar .zoom-title {
  color: #aaa;
  font-size: 11px;
  writing-mode: vertical-lr;
  letter-spacing: 2px;
}
/* ± 按钮 */
.right-zoom-sidebar .zoom-btn {
  width: 28px;
  height: 28px;
  line-height: 28px;
  text-align: center;
  padding: 0;
  font-size: 16px;
  font-weight: bold;
  border-radius: 50%;
}
/* 竖向 Slider */
.right-zoom-sidebar input[type="range"] {
  width: 120px;
  height: 6px;
  -webkit-appearance: none;
  appearance: none;
  background: #555;
  border-radius: 3px;
  outline: none;
  cursor: pointer;
  writing-mode: vertical-lr;
  direction: rtl;
}
.right-zoom-sidebar input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 16px;
  height: 16px;
  background: #ed2553;
  border-radius: 50%;
  cursor: pointer;
}
.right-zoom-sidebar input[type="range"]::-moz-range-thumb {
  width: 16px;
  height: 16px;
  background: #ed2553;
  border-radius: 50%;
  cursor: pointer;
  border: none;
}
.right-zoom-sidebar .zoom-value {
  color: #ccc;
  font-size: 12px;
  min-width: 40px;
  text-align: center;
}
.zoom-reset-btn {
  font-size: 11px;
  padding: 2px 8px;
  cursor: pointer;
  color: #888;
  background: transparent;
  border: 1px solid #555;
  border-radius: 3px;
  transition: color 0.15s, border-color 0.15s;
}
.zoom-reset-btn:hover {
  color: #fff;
  border-color: #888;
}
.zoom-presets {
  display: flex;
  flex-direction: column;
  gap: 3px;
  align-items: center;
}
.zoom-presets .preset-label {
  color: #666;
  font-size: 10px;
}
.zoom-presets .preset-btn {
  font-size: 11px;
  padding: 1px 6px;
  color: #888;
  cursor: pointer;
  text-decoration: none;
  border-radius: 2px;
  transition: color 0.15s, background 0.15s;
}
.zoom-presets .preset-btn:hover {
  color: #fff;
  background: rgba(255,255,255,0.1);
}
```

- [ ] **Step 4: 新增分页跳页与弹窗增强样式**

```css
/* ===== 分页跳页 ===== */
.page-jump {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 8px;
  color: #888;
  font-size: 13px;
}
.page-jump .jump-input {
  width: 50px;
  padding: 2px 4px;
  background: #333;
  color: #fff;
  border: 1px solid #555;
  border-radius: 3px;
  text-align: center;
  font-size: 13px;
}
.page-jump .jump-go {
  padding: 2px 8px;
  font-size: 12px;
}

/* ===== 弹窗增强 ===== */
body.modal-open {
  overflow: hidden;
  padding-right: var(--scrollbar-width, 0px);
}
```

- [ ] **Step 5: 新增移动端适配样式**

```css
/* ===== 移动端：侧边栏折叠 ===== */
@media (max-width: 768px) {
  .left-action-sidebar {
    position: fixed;
    top: auto;
    bottom: 0;
    left: 0;
    right: 0;
    transform: none;
    flex-direction: row;
    justify-content: space-around;
    border-radius: 8px 8px 0 0;
    padding: 6px 4px;
    background: rgba(20,20,20,0.92);
    z-index: 999;
  }
  .left-action-sidebar .sidebar-btn {
    min-width: 44px;
    padding: 4px 2px;
  }
  .left-action-sidebar .sidebar-btn .label {
    font-size: 9px;
  }
  .right-zoom-sidebar {
    display: none; /* 移动端默认隐藏，通过浮动按钮切换 */
  }
  .right-zoom-sidebar.mobile-open {
    display: flex;
    position: fixed;
    top: auto;
    bottom: 60px;
    right: 10px;
    transform: none;
    flex-direction: row;
    flex-wrap: wrap;
    width: auto;
    border-radius: 8px;
    padding: 8px;
  }
  .right-zoom-sidebar.mobile-open input[type="range"] {
    writing-mode: horizontal-tb;
    direction: ltr;
    width: 100px;
    height: 6px;
  }
  /* 移动端缩放浮动按钮 */
  .zoom-float-btn {
    display: flex;
    position: fixed;
    bottom: 70px;
    right: 10px;
    z-index: 998;
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: rgba(237,37,83,0.9);
    color: #fff;
    align-items: center;
    justify-content: center;
    font-size: 18px;
    cursor: pointer;
    box-shadow: 0 2px 8px rgba(0,0,0,0.4);
    border: none;
  }
}
@media (min-width: 769px) {
  .zoom-float-btn {
    display: none;
  }
}
```

- [ ] **Step 6: 缩略图容器边距适配（防止被侧边栏遮挡）**

缩略图容器当 `EnableLarge=true` 时需要左右留出侧边栏空间，但该逻辑建议通过 JS 动态控制（而非固定的 CSS），因为 `gallery_detail.tpl` 中 `#thumbnail-container` 是固定结构，让 JS 在 `EnableLarge` 时给容器加 class `.with-sidebars`：

```css
.thumb-container.with-sidebars {
  margin-left: 70px;
  margin-right: 60px;
  transition: margin 0.2s;
}
```

---

### Task 2: JS — LoadingManager + Toast 升级

**Files:**
- Modify: `cmd/server/view/static/custom/js/scripts.js`

- [ ] **Step 1: 新增 LoadingManager 对象**

在 `scripts.js` 文件末尾（最后一个函数之后）追加：

```javascript
/**
 * LoadingManager — 按钮级 loading 状态管理
 * 用法: LoadingManager.start(btnEl); LoadingManager.done(btnEl);
 */
const LoadingManager = {
  start(btn) {
    if (!btn || btn.dataset.loading) return;
    btn.dataset.loading = 'true';
    btn.dataset.origHTML = btn.innerHTML;
    btn.classList.add('btn-loading');
    btn.disabled = true;
  },
  done(btn) {
    if (!btn) return;
    delete btn.dataset.loading;
    btn.classList.remove('btn-loading', 'btn-error');
    btn.disabled = false;
  },
  error(btn) {
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
```

- [ ] **Step 2: 升级 showToast（支持类型图标 + 堆叠）**

找到现有的 `showToast` 函数并**替换**为：

```javascript
function showToast(message, opts) {
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
      if (alert && alert.parentNode) {
        alert.parentNode.removeChild(alert);
      }
    });
  }
  container.appendChild(alert);
  if (duration > 0) {
    setTimeout(function() {
      if (alert && alert.parentNode) {
        alert.parentNode.removeChild(alert);
      }
    }, duration);
  }
}
```

- [ ] **Step 3: 新增 showProgressToast（进度 Toast）**

```javascript
function showProgressToast(message, percent) {
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
    '<div class="progress-bar" style="width:' + Math.min(100, Math.max(0, percent || 0)) + '%;height:100%;background:#4CAF50;transition:width 0.3s;"></div></div>';
  container.appendChild(toast);
}
```

---

### Task 3: JS — OptimisticUpdater + 键盘导航工具

**Files:**
- Modify: `cmd/server/view/static/custom/js/scripts.js`

- [ ] **Step 1: 新增 OptimisticUpdater**

在 `LoadingManager` 之后追加：

```javascript
/**
 * OptimisticUpdater — 乐观更新 + 局部刷新工具
 */
const OptimisticUpdater = {
  // 乐观 Toggle：立即切换 class，请求失败回滚
  optimisticToggle(btn, activeClass, inactiveClass) {
    var wasActive = btn.classList.contains(activeClass);
    var rollbackState = { activeClass, inactiveClass, wasActive };
    // 立即切换
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

  // 局部刷新：fetch 数据并执行 render 函数替换容器内容
  refreshContainer(url, containerSelector, renderFn) {
    var container = document.querySelector(containerSelector);
    if (!container) return Promise.reject('Container not found: ' + containerSelector);
    return fetch(url, { credentials: 'include' })
      .then(function(r) { return r.json(); })
      .then(function(data) {
        if (renderFn && typeof renderFn === 'function') {
          renderFn(container, data);
        }
        return data;
      });
  }
};
```

- [ ] **Step 2: 新增 enableAutocompleteKeyboardNav**

```javascript
/**
 * 自动补全下拉键盘导航
 * @param {HTMLElement} dropdown - 下拉容器
 * @param {Function} onSelect - 选中回调，接收当前高亮项索引
 * @returns {Function} destroy 函数
 */
function enableAutocompleteKeyboardNav(dropdown, onSelect) {
  var selectedIdx = -1;

  function getItems() {
    return dropdown.querySelectorAll('div');
  }

  function highlight(idx) {
    var items = getItems();
    items.forEach(function(el, i) {
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
  // dropdown 本身不可聚焦，监听挂载在 document 上但只在下拉可见时生效的变体
  // 实际场景：键盘事件由父级输入框捕获
  return function destroy() {
    dropdown.removeEventListener('keydown', handler);
  };
}

// 在输入框上绑定键盘导航（输入框的 keydown 事件委托给 dropdown）
function bindAutocompleteKeys(input, dropdown, onEnter) {
  input.addEventListener('keydown', function(e) {
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
}
```

---

### Task 4: 模板 — 双侧边栏布局（gallery_detail.tpl）

**Files:**
- Modify: `cmd/server/view/static/tpl/gallery_detail.tpl`

- [ ] **Step 1: 从 #info-block .buttons 中移除操作按钮**

当前 `gallery_detail.tpl` 第 79-105 行的 `.buttons` 区域保留仅保留禁用态占位按钮（Favorite / Download / Comment），功能性按钮（Like、归档、恢复、修复、编辑 Tags、强制归档）移到侧边栏。

修改后（只保留非功能性占位按钮）：

```html
                    <div class="buttons">
                        <a class="btn btn-primary btn-disabled tooltip">
                            <i class="fas fa-heart"></i>
                            <span>
                                    Favorite <span class="nobold">(1911)</span>
                            </span>
                            <div class="top">
                                You need to log in to add favorites<i></i>
                            </div>
                        </a>
                        <a id="download" class="btn btn-secondary btn-disabled tooltip">
                            <i class="fa fa-download"></i> Download
                            <div class="top">
                                You need to log in to download<i></i>
                            </div>
                        </a>
                    </div>
```

删除的行（从 `gallery_detail.tpl` 中移除）：
- `addLikeGroup` 行
- `archiveToggle` 行（归档/恢复）
- `fixStatusBtn` 行（修复漫画）
- `editTagsBtn` 行（编辑 Tags）

这些按钮将放到左侧边栏中。

- [ ] **Step 2: 在 #content 内添加左侧操作侧边栏**

在 `gallery_detail.tpl` 的 `<div id="content">` 内最前面添加：

```html
        <!-- 左侧操作侧边栏 -->
        <div class="left-action-sidebar">
            <a id="sidebarLikeBtn" class="sidebar-btn {{if .HasLike}}btn-primary{{else}}{{end}}" href="javascript:;" onclick="addLikeGroup({{.CID}})">
                <i class="fas fa-heart"></i>
                <span class="label">{{if .HasLike}}Liked{{else}}Like{{end}}</span>
            </a>
            {{ if and .Archive (ne .Archive.Path "") }}
            <a id="sidebarArchiveBtn" class="sidebar-btn" href="javascript:;" onclick="restoreComic({{.CID}})">
                <i class="fa fa-undo"></i>
                <span class="label">恢复</span>
            </a>
            {{ else }}
            <a id="sidebarArchiveBtn" class="sidebar-btn" href="javascript:;" onclick="archiveComic({{.CID}})">
                <i class="fa fa-archive"></i>
                <span class="label">归档</span>
            </a>
            {{ end }}
            <a id="sidebarFixBtn" class="sidebar-btn" href="javascript:;" onclick="verifyComic({{.CID}})">
                <i class="fa fa-wrench"></i>
                <span class="label">修复</span>
            </a>
            <a id="sidebarEditTagsBtn" class="sidebar-btn" href="javascript:;" onclick="openTagEditor({{.CID}})">
                <i class="fa fa-tags"></i>
                <span class="label">编辑Tag</span>
            </a>
        </div>
```

- [ ] **Step 3: 在 #content 内添加右侧缩放侧边栏（仅 EnableLarge）**

在上一步左侧边栏之后添加：

```html
        {{if .EnableLarge}}
        <!-- 右侧缩放侧边栏 -->
        <div class="right-zoom-sidebar" id="zoomSidebar">
            <div class="zoom-title">缩放</div>
            <button type="button" class="btn btn-secondary zoom-btn" id="zoomInBtn" title="放大">+</button>
            <input type="range" id="thumbZoomSlider" min="60" max="1200" value="1200" step="20" />
            <button type="button" class="btn btn-secondary zoom-btn" id="zoomOutBtn" title="缩小">−</button>
            <div class="zoom-value"><span id="zoomValue">1200</span>px</div>
            <button type="button" class="zoom-reset-btn" id="zoomResetBtn">重置</button>
            <div class="zoom-presets">
                <span class="preset-label">预设</span>
                <a href="javascript:;" class="preset-btn" data-zoom="200">200px</a>
                <a href="javascript:;" class="preset-btn" data-zoom="400">400px</a>
                <a href="javascript:;" class="preset-btn" data-zoom="600">600px</a>
                <a href="javascript:;" class="preset-btn" data-zoom="800">800px</a>
                <a href="javascript:;" class="preset-btn" data-zoom="1000">1000px</a>
            </div>
        </div>
        <!-- 移动端缩放浮动按钮 -->
        <div class="zoom-float-btn" id="zoomFloatBtn" onclick="toggleMobileZoom()">🔍</div>
        {{end}}
```

并在移动端页面添加 toggle 函数（和 `initThumbnailZoom` 放一起）：

```javascript
// 在 scripts.js 末尾添加
function toggleMobileZoom() {
  var sidebar = document.getElementById('zoomSidebar');
  if (sidebar) sidebar.classList.toggle('mobile-open');
}
```

- [ ] **Step 4: 给缩略图容器添加 with-sidebars class（EnableLarge 时）**

在 `gallery_detail.tpl` 的 `#thumbnail-container` 元素上：

```html
        <div class="container{{if .EnableLarge}} with-sidebars{{end}}" id="thumbnail-container">
```

（在 CSS 中 `with-sidebars` 已经定义左右 margin，参考 Task 1 Step 6）

- [ ] **Step 5: 移除旧的缩略图缩放控件区域**

找到 `gallery_detail.tpl` 中的：

```html
            {{if .EnableLarge}}
            <div class="thumb-zoom-controls">
                <button type="button" class="btn btn-secondary btn-square" id="zoomOutBtn" title="缩小">−</button>
                <input type="range" id="thumbZoomSlider" min="60" max="1200" value="1200" step="50" />
                <button type="button" class="btn btn-secondary btn-square" id="zoomInBtn" title="放大">+</button>
                <span class="zoom-level"><span id="zoomValue">1200</span>px</span>
            </div>
            {{end}}
```

将其**替换**为空的占位（因为已经移到右侧边栏），或者直接删除整个块。

---

### Task 5: 模板 — 分页跳页 + 导航文案

**Files:**
- Modify: `cmd/server/view/static/tpl/index.tpl`
- Modify: `cmd/server/view/static/tpl/tag_list_result.tpl`
- Modify: `cmd/server/view/static/tpl/head.tpl`

- [ ] **Step 1: index.tpl — 分页增加跳页控件**

找到 `index.tpl` 的分页区域（约第 96-107 行），在 `.ios-mobile-webkit-bottom-spacing` 前面增加：

```html
            <span class="page-jump">
                跳至 <input type="number" class="jump-input" min="1" max="{{.LastPage}}"
                    onkeydown="if(event.key==='Enter') jumpToPage(this, '{{$.URL}}', '{{$.SearchQuery}}')" /> 页
                <button class="btn btn-secondary btn-square jump-go"
                    onclick="jumpToPage(this.previousElementSibling, '{{$.URL}}', '{{$.SearchQuery}}')">GO</button>
            </span>
```

同时在 `scripts.js` 末尾添加跳页函数：

```javascript
function jumpToPage(input, baseUrl, query) {
  var page = parseInt(input.value, 10);
  var max = parseInt(input.max, 10);
  if (isNaN(page) || page < 1 || page > max) {
    showToast('页码应在 1 ~ ' + max + ' 之间', { type: 'warning' });
    return;
  }
  var url = baseUrl + '?page=' + page;
  if (query) url += '&q=' + encodeURIComponent(query);
  window.location.href = url;
}
```

- [ ] **Step 2: tag_list_result.tpl — 分页增加跳页控件**

同 Step 1 的逻辑，在 `tag_list_result.tpl` 的分页区域增加跳页控件。

注意 `tag_list_result.tpl` 的分页 query 参数不同（`sortType=popular`），需适配。

在当前分页区域的 `.ios-mobile-webkit-bottom-spacing` 前面：

```html
            <span class="page-jump">
                跳至 <input type="number" class="jump-input" min="1" max="{{.LastPage}}"
                    onkeydown="if(event.key==='Enter') jumpToTagPage(this, '{{$.URL}}', {{$.SortType}})" /> 页
                <button class="btn btn-secondary btn-square jump-go"
                    onclick="jumpToTagPage(this.previousElementSibling, '{{$.URL}}', {{$.SortType}})">GO</button>
            </span>
```

新增辅助函数：

```javascript
function jumpToTagPage(input, baseUrl, sortType) {
  var page = parseInt(input.value, 10);
  var max = parseInt(input.max, 10);
  if (isNaN(page) || page < 1 || page > max) {
    showToast('页码应在 1 ~ ' + max + ' 之间', { type: 'warning' });
    return;
  }
  var url = baseUrl + '?page=' + page;
  if (sortType === 1) url += '&sortType=popular';
  window.location.href = url;
}
```

- [ ] **Step 3: head.tpl — enableLarge 链接改名 + 搜索框 placeholder**

在 `head.tpl` 中的 `navigation.tpl` 定义内，找到 "enableLarge" 链接（共 2 处——桌面端导航和下拉菜单）：

桌面端（约第 52 行）：
```html
                <li class="desktop ">
                    <a href="{{$.URL}}?large=true">大图模式</a>
                </li>
```

下拉菜单（约第 85 行）：
```html
                        <li>
                            <a href="{{$.URL}}?large=true">大图模式</a>
                        </li>
```

搜索框 placeholder（约第 17 行）：
```html
            <input required type="search" name="q" value="" autocapitalize="none" placeholder="搜索漫画编号或标题..." />
```

- [ ] **Step 4: 首页搜索框自动聚焦**

不需要改模板，在 `scripts.js` 中添加搜索快捷键逻辑（见 Task 6）。

---

### Task 6: JS — 操作按钮接入 Loading + 无刷新

**Files:**
- Modify: `cmd/server/view/static/custom/js/scripts.js`

- [ ] **Step 1: 重写 addLikeGroup（乐观更新 + Loading）**

将现有的 `addLikeGroup` 函数替换为：

```javascript
function addLikeGroup(cid) {
    var btn = document.getElementById('sidebarLikeBtn');
    if (!btn) btn = document.getElementById('addLikeGroup');
    if (!btn || btn.dataset.loading) return;
    LoadingManager.start(btn);

    var liked = btn.classList.contains('btn-primary');
    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/like');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');

    // 乐观更新：立即切换 UI
    var toggle = OptimisticUpdater.optimisticToggle(btn, 'btn-primary', 'btn-secondary');
    var label = btn.querySelector('.label');
    if (label) label.textContent = liked ? 'Like' : 'Liked';

    xhr.onload = function() {
        LoadingManager.done(btn);
        if (xhr.status >= 200 && xhr.status < 300) {
            // 同步详情页的标签列表（如果有 like tag 的视觉反馈）
            var detailLikeTag = document.querySelector('.tag-99999');
            if (liked && detailLikeTag) {
                detailLikeTag.remove();
            } else if (!liked) {
                addLikeTag();
            }
            showToast(liked ? '已取消 Like' : '已添加 Like', { type: 'success' });
        } else {
            // 回滚
            toggle.rollback();
            if (label) label.textContent = liked ? 'Liked' : 'Like';
            showToast('操作失败', { type: 'error' });
        }
    };
    xhr.onerror = function() {
        LoadingManager.done(btn);
        toggle.rollback();
        if (label) label.textContent = liked ? 'Liked' : 'Like';
        showToast('网络错误', { type: 'error' });
    };
    xhr.send('cid=' + encodeURIComponent(cid));
}
```

- [ ] **Step 2: 重写 archiveComic（Loading + 无刷新）**

将现有的 `archiveComic` 函数替换为：

```javascript
function archiveComic(cid) {
    var btn = document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive');
    xhr.onload = function() {
        LoadingManager.done(btn);
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('已归档', { type: 'success' });
                // 无刷新：按钮切换为"恢复"
                btn.innerHTML = '<i class="fa fa-undo"></i><span class="label">恢复</span>';
                btn.onclick = function() { restoreComic(cid); };
                btn.id = 'sidebarArchiveBtn';
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
                if (resp.head.code === -1001) {
                    var invalids = (resp.body && resp.body.invalid_images) || [];
                    if (!invalids.length && window._gallery && window._gallery.images && Array.isArray(window._gallery.images.pages)) {
                        window._gallery.images.pages.forEach(function(p, i) {
                            if (p && p.status === false) invalids.push({ index: i + 1 });
                        });
                    }
                    highlightInvalidPages(invalids);
                    ensureForceArchiveButton(cid);
                    showToast('检测到异常图片，建议先"修复漫画状态"，或使用"强制归档"', { type: 'info' });
                }
            }
        } else {
            LoadingManager.error(btn);
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        LoadingManager.error(btn);
        showToast('网络错误', { type: 'error' });
    };
    xhr.send();
}
```

- [ ] **Step 3: 重写 restoreComic（Loading + 无刷新）**

将现有的 `restoreComic` 函数替换为：

```javascript
function restoreComic(cid) {
    var btn = document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/restore');
    xhr.onload = function() {
        LoadingManager.done(btn);
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('已恢复', { type: 'success' });
                // 无刷新：按钮切换为"归档"
                btn.innerHTML = '<i class="fa fa-archive"></i><span class="label">归档</span>';
                btn.onclick = function() { archiveComic(cid); };
                btn.id = 'sidebarArchiveBtn';
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
            }
        } else {
            LoadingManager.error(btn);
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        LoadingManager.error(btn);
        showToast('网络错误', { type: 'error' });
    };
    xhr.send();
}
```

- [ ] **Step 4: 重写 verifyComic（Loading）**

将现有的 `verifyComic` 函数替换为（仅增加 Loading）：

```javascript
function verifyComic(cid) {
    var btn = document.getElementById('sidebarFixBtn');
    if (!btn || btn.dataset.loading) return;
    LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/verify');
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function() {
        LoadingManager.done(btn);
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('修复任务已启动', { type: 'success' });
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
            }
        } else {
            LoadingManager.error(btn);
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        LoadingManager.error(btn);
        showToast('网络错误', { type: 'error' });
    };
    var body = { id: String(cid), autoFix: true, maxWorkers: 1 };
    xhr.send(JSON.stringify(body));
}
```

- [ ] **Step 5: 重写 archiveComicForce（Loading）**

将现有的 `archiveComicForce` 函数替换为（仅增加 Loading）：

```javascript
function archiveComicForce(cid) {
    var btn = document.getElementById('forceArchiveBtn') || document.getElementById('sidebarArchiveBtn');
    if (!btn || btn.dataset.loading) return;
    LoadingManager.start(btn);

    var xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/v2/api/nhcomic/' + encodeURIComponent(cid) + '/archive?force=true');
    xhr.onload = function() {
        LoadingManager.done(btn);
        if (xhr.status == 200) {
            var resp = JSON.parse(xhr.responseText);
            if (resp.head.code === 0) {
                showToast('已强制归档', { type: 'success' });
                btn.innerHTML = '<i class="fa fa-undo"></i><span class="label">恢复</span>';
                btn.onclick = function() { restoreComic(cid); };
            } else {
                var msg = formatError(resp);
                showToast(msg, { type: 'error' });
            }
        } else {
            LoadingManager.error(btn);
            var msg = xhr.responseText || ('请求失败: ' + xhr.status);
            showToast(msg, { type: 'error' });
        }
    };
    xhr.onerror = function() {
        LoadingManager.error(btn);
        showToast('网络错误', { type: 'error' });
    };
    xhr.send();
}
```

- [ ] **Step 6: 更新 ensureForceArchiveButton 指向侧边栏**

修改 `ensureForceArchiveButton` 函数，让它把"强制归档"按钮插入左侧边栏而非 `#info-block .buttons`：

```javascript
function ensureForceArchiveButton(cid) {
    var existing = document.getElementById('forceArchiveBtn');
    if (existing) return;
    var sidebar = document.querySelector('.left-action-sidebar');
    if (!sidebar) return;
    var a = document.createElement('a');
    a.id = 'forceArchiveBtn';
    a.href = 'javascript:;';
    a.className = 'sidebar-btn';
    a.innerHTML = '<i class="fa fa-exclamation-triangle"></i><span class="label">强制归档</span>';
    a.onclick = function() { archiveComicForce(cid); };
    sidebar.appendChild(a);
}
```

- [ ] **Step 7: 重写 Tag 编辑器 Save 部分（无刷新）**

找到 `buildTagEditorModal` 中的 Save 按钮回调（约 `saveBtn.onclick` 部分），改为保存成功后用 `OptimisticUpdater.refreshContainer` 刷新标签区域而非 `location.reload()`：

```javascript
    saveBtn.onclick = function() {
        if (added.length === 0 && removed.length === 0) {
            showToast('没有变更', { type: 'info' });
            return;
        }
        LoadingManager.start(saveBtn);
        var payload = JSON.stringify({ cid: cid, added: added, removed: removed });
        var saveXhr = new XMLHttpRequest();
        saveXhr.withCredentials = true;
        saveXhr.open('POST', '/api/comic/tags/update');
        saveXhr.setRequestHeader('Content-Type', 'application/json');
        saveXhr.onload = function() {
            LoadingManager.done(saveBtn);
            if (saveXhr.status >= 200 && saveXhr.status < 300) {
                showToast('Tags 已更新', { type: 'success' });
                closeModal(wrapper);
                // 无刷新：通过 getComicInfo 刷新标签区域
                OptimisticUpdater.refreshContainer(
                    '/api/comic/getComicInfo',
                    '#tags',
                    function(container, data) {
                        // 重新构建 tag 列表（沿用服务端返回的 tags）
                        if (data.body && data.body.tags) {
                            rebuildTagsSection(data.body.tags);
                        }
                    }
                );
            } else {
                try {
                    var r = JSON.parse(saveXhr.responseText);
                    showToast(r.head && r.head.msg || '保存失败', { type: 'error' });
                } catch(e) {
                    showToast('保存失败: ' + saveXhr.status, { type: 'error' });
                }
            }
        };
        saveXhr.onerror = function() {
            LoadingManager.done(saveBtn);
            showToast('网络错误', { type: 'error' });
        };
        saveXhr.send(payload);
    };
```

新增 `rebuildTagsSection` 辅助函数：

```javascript
function rebuildTagsSection(tags) {
    var container = document.querySelector('#tags');
    if (!container) return;
    // 按 type 分组重建标签 HTML
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
}
```

- [ ] **Step 8: Tag 对齐器 Apply 后无刷新（仅关闭弹窗 + Toast，不 reload）**

找到 `buildTagAlignerModal` 中 `applyBtn.onclick` 成功回调，去掉 `location.reload()`：

```javascript
    applyBtn.onclick = function() {
        if (!selectedTag) { showToast('请先选择一个标签', { type: 'error' }); return; }
        LoadingManager.start(applyBtn);

        var payload = JSON.stringify({ cidList: cidList, tag: selectedTag });
        var xhr = new XMLHttpRequest();
        xhr.withCredentials = true;
        xhr.open('POST', '/api/comic/tags/batch-add');
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.onload = function() {
            LoadingManager.done(applyBtn);
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    var resp = JSON.parse(xhr.responseText);
                    var data = resp.body;
                    var msg = '标签 "' + selectedTag.name + '" 已添加到 ' + data.updated + '/' + cidList.length + ' 本漫画';
                    if (data.errors && data.errors.length > 0) {
                        msg += '，' + data.errors.length + ' 本失败';
                    }
                    showToast(msg, { type: 'success' });
                    closeModal(wrapper);
                    // 不 reload — 用户在搜索结果页可自行刷新
                } catch(e) {
                    showToast('处理完成', { type: 'success' });
                    closeModal(wrapper);
                }
            } else {
                LoadingManager.error(applyBtn);
                try {
                    var r = JSON.parse(xhr.responseText);
                    showToast(r.head && r.head.msg || '批量添加失败', { type: 'error' });
                } catch(e) {
                    showToast('批量添加失败: ' + xhr.status, { type: 'error' });
                }
            }
        };
        xhr.onerror = function() {
            LoadingManager.error(applyBtn);
            showToast('网络错误', { type: 'error' });
        };
        xhr.send(payload);
    };
```

---

### Task 7: JS — 搜索快捷键 + 自动聚焦

**Files:**
- Modify: `cmd/server/view/static/custom/js/scripts.js`

- [ ] **Step 1: 添加全局搜索快捷键**

在 `scripts.js` 末尾添加：

```javascript
/**
 * 全局搜索快捷键：按 / 聚焦搜索框
 */
document.addEventListener('keydown', function(e) {
    // 如果在 input/textarea/contenteditable 中，不触发
    var tag = e.target.tagName;
    if (tag === 'INPUT' || tag === 'TEXTAREA' || e.target.isContentEditable) return;

    if (e.key === '/') {
        e.preventDefault();
        var searchInput = document.querySelector('input[type="search"]');
        if (searchInput) {
            searchInput.focus();
            searchInput.select();
        }
    }
});
```

- [ ] **Step 2: 搜索框 Esc 失焦**

```javascript
document.addEventListener('focusin', function(e) {
    if (e.target && e.target.type === 'search') {
        e.target.addEventListener('keydown', function escHandler(ev) {
            if (ev.key === 'Escape') {
                ev.target.blur();
                ev.target.removeEventListener('keydown', escHandler);
            }
        });
    }
});
```

- [ ] **Step 3: 首页自动聚焦搜索框**

在 `initThumbnailZoom` 之后或脚本末尾添加：

```javascript
// 首页自动聚焦搜索框
(function() {
    var path = window.location.pathname;
    if (path === '/' || path === '/search/') {
        var searchInput = document.querySelector('input[type="search"]');
        if (searchInput && !searchInput.value) {
            // 延迟聚焦避免干扰页面加载
            setTimeout(function() { searchInput.focus(); }, 300);
        }
    }
})();
```

---

### Task 8: JS — 弹窗增强（Focus Trap + Scroll Lock + 键盘导航）

**Files:**
- Modify: `cmd/server/view/static/custom/js/scripts.js`

- [ ] **Step 1: 增强 showCustomModal（Focus Trap + Scroll Lock）**

将现有的 `showCustomModal` 函数替换为：

```javascript
function showCustomModal(title, contentHtml, buttonsHtml) {
    var existing = document.querySelector('.modal-wrapper');
    if (existing) existing.remove();

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

    // ===== 保存触发焦点 =====
    var prevFocus = document.activeElement;

    // ===== Scroll Lock =====
    var scrollbarWidth = window.innerWidth - document.documentElement.clientWidth;
    document.body.style.setProperty('--scrollbar-width', scrollbarWidth + 'px');
    document.body.classList.add('modal-open');

    // ===== Focus Trap =====
    var focusableSel = 'input, button, [href], select, textarea, [tabindex]:not([tabindex="-1"])';
    function trapFocus(e) {
        if (e.key !== 'Tab') return;
        var focusable = wrapper.querySelectorAll(focusableSel);
        if (focusable.length === 0) return;
        var first = focusable[0];
        var last = focusable[focusable.length - 1];
        if (e.shiftKey) {
            if (document.activeElement === first) {
                e.preventDefault();
                last.focus();
            }
        } else {
            if (document.activeElement === last) {
                e.preventDefault();
                first.focus();
            }
        }
    }
    document.addEventListener('keydown', trapFocus);

    // 聚焦到第一个可聚焦元素
    setTimeout(function() {
        var firstFocusable = wrapper.querySelector(focusableSel);
        if (firstFocusable) firstFocusable.focus();
    }, 50);

    // ===== 点击遮罩层关闭 =====
    wrapper.addEventListener('click', function(e) {
        if (e.target === wrapper) closeModal(wrapper);
    });

    // ===== Esc 键关闭 =====
    var escHandler = function(e) {
        if (e.key === 'Escape') {
            closeModal(wrapper);
        }
    };
    wrapper._escHandler = escHandler;
    document.addEventListener('keydown', escHandler);

    // 保存清理数据
    wrapper._trapFocus = trapFocus;
    wrapper._prevFocus = prevFocus;

    return wrapper;
}
```

- [ ] **Step 2: 增强 closeModal（清理焦点 + 滚动锁）**

将现有的 `closeModal` 函数替换为：

```javascript
function closeModal(wrapper) {
    if (wrapper && wrapper.parentNode) {
        if (wrapper._escHandler) {
            document.removeEventListener('keydown', wrapper._escHandler);
        }
        if (wrapper._trapFocus) {
            document.removeEventListener('keydown', wrapper._trapFocus);
        }
        // 焦点恢复
        if (wrapper._prevFocus) {
            wrapper._prevFocus.focus();
        }
        // 关闭动画
        wrapper.classList.remove('open');
        wrapper.classList.add('fade-slide-out');
        setTimeout(function() {
            if (wrapper && wrapper.parentNode) {
                wrapper.parentNode.removeChild(wrapper);
            }
        }, 150);
    }
    // 检查是否还有其他打开弹窗
    if (!document.querySelector('.modal-wrapper')) {
        document.body.classList.remove('modal-open');
        document.body.style.removeProperty('--scrollbar-width');
    }
}
```

- [ ] **Step 3: 集成自动补全键盘导航到 Tag 编辑器**

在 `buildTagEditorModal` 中，找到搜索输入框创建后的位置，在 `searchInput.addEventListener('input', ...)` 之后绑定键盘导航：

```javascript
    // 绑定键盘导航
    bindAutocompleteKeys(searchInput, autocompleteDropdown, function() {
        // Enter 无下拉选中时：选中第一个结果
        var firstItem = autocompleteDropdown.querySelector('div');
        if (firstItem) firstItem.click();
    });
```

- [ ] **Step 4: 集成自动补全键盘导航到关系管理器**

在 `buildRelationModal` 中，在 `relSearchInput.addEventListener('input', ...)` 之后：

```javascript
    // 绑定键盘导航
    bindAutocompleteKeys(relSearchInput, relDropdown, function() {
        var firstItem = relDropdown.querySelector('div');
        if (firstItem) firstItem.click();
    });
```

- [ ] **Step 5: Tag 编辑器 New 模式 Enter 快捷提交**

在 `buildTagEditorModal` 的 `nameInput` 创建后，找到 `addBtn.onclick` 之前添加：

```javascript
    nameInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            addBtn.click();
        }
    });
```

- [ ] **Step 6: 关系管理器 New 模式 Enter 快捷提交**

在 `buildRelationModal` 的 `relNewNameInput` 创建后，找到 `relAddNewBtn.onclick` 之前：

```javascript
    relNewNameInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
            e.preventDefault();
            relAddNewBtn.click();
        }
    });
```

---

### Task 9: JS — 缩略图缩放侧边栏集成

**Files:**
- Modify: `cmd/server/view/static/custom/js/scripts.js`

- [ ] **Step 1: 更新 initThumbnailZoom（重置按钮、预设快捷值、竖向 Slider、step 20）**

将现有的 `initThumbnailZoom` 函数替换为：

```javascript
function initThumbnailZoom() {
    var slider = document.getElementById('thumbZoomSlider');
    var zoomValue = document.getElementById('zoomValue');
    var zoomInBtn = document.getElementById('zoomInBtn');
    var zoomOutBtn = document.getElementById('zoomOutBtn');
    var zoomResetBtn = document.getElementById('zoomResetBtn');
    var container = document.getElementById('thumbnail-container');
    if (!slider || !container) return;

    // 从 localStorage 恢复
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

    // 初始应用
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

    // ===== 重置按钮 =====
    if (zoomResetBtn) {
        zoomResetBtn.addEventListener('click', function() {
            slider.value = 200;
            applyZoom(200);
        });
    }

    // ===== 预设快捷值 =====
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
}
```

- [ ] **Step 2: 验证页面加载后初始化**

原有的 `DOMContentLoaded` 逻辑保持不变：

```javascript
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initThumbnailZoom);
} else {
    initThumbnailZoom();
}
```

---

### Task 10: 验证 + 提交

- [ ] **Step 1: 视觉验证 — 双侧边栏布局**

启动 server，访问详情页：`/g/<cid>/?large=true`
- 确认左侧边栏显示在页面左侧固定位置
- 确认右侧缩放侧边栏显示在页面右侧固定位置
- 确认缩略图容器没有被遮挡（左右 margin 生效）
- 确认操作按钮在侧边栏中可点击

- [ ] **Step 2: 视觉验证 — 移动端适配**

浏览器 DevTools 切换到 < 768px 宽度：
- 左侧边栏折叠到底部横排工具栏
- 右侧缩放侧边栏隐藏，浮动按钮显示
- 点击浮动按钮展开缩放面板

- [ ] **Step 3: 功能验证 — Loading 反馈**

逐个测试：Like、归档、恢复、修复、强制归档
- 点击后按钮显示 spinner 动画
- 按钮不可重复点击（disabled）
- 操作完成后 spinner 消失
- 网络错误时按钮抖动

- [ ] **Step 4: 功能验证 — 无刷新更新**

- Like 操作后 UI 立即切换（乐观更新），刷新页面后状态保持
- Tag 编辑保存后标签区域更新，无刷新
- 归档后按钮变为"恢复"，无刷新
- 恢复后按钮变为"归档"，无刷新

- [ ] **Step 5: 功能验证 — 搜索 + 分页**

- 按 `/` 聚焦搜索框
- Esc 失焦
- 分页跳页输入页码跳转正确
- 越界提示合理

- [ ] **Step 6: 功能验证 — 弹窗交互**

- 弹窗打开后 Tab 循环在弹窗内
- 背景页面不可滚动
- 自动补全下拉 ↑↓ 选择，Enter 确认
- Tag 编辑器 New 模式 Enter 提交
- Esc 关闭弹窗，焦点回到触发按钮

- [ ] **Step 7: 功能验证 — 缩略图缩放**

- 竖向 Slider 拖动缩放
- 重置按钮恢复 200px
- 预设快捷值跳转正确
- 刷新页面后值保持

- [ ] **Step 8: 提交**

```bash
cd D:/workdir/leon/cocomhub/cocom
git add -A
git commit -m "feat(ui): 全面增强交互体验 — 双侧边栏、Loading 反馈、无刷新更新、弹窗交互优化、搜索快捷操作"
```