// Copyright 2026 The Cocomhub Authors. All rights reserved.
// Use of this source code is governed by a Apache-2.0 license that can be
// found in the LICENSE file.

package helpers

// CSS 选择器常量 — 所有稳定 ID 均提取自模板文件
const (
	// 顶部导航栏
	LogoLink       = "a.logo"
	SearchForm     = "form.search"
	SearchInput    = "form.search input[type=search]"
	SearchSubmit   = "form.search button[type=submit]"
	HamburgerBtn   = "#hamburger"
	NavTagsLink    = "a[href='/list/tags/']"
	NavArtistsLink = "a[href='/list/artists/']"
	NavAdminLink   = "a[href='/admin']"
	NavDropdown    = "#dropdown"

	// 首页快速操作侧边栏
	QuickSidebar   = "#quick-action-sidebar"
	LinkModeBtn    = "#btn-link-mode"
	CompareModeBtn = "#btn-compare-mode"
	NewTabCheckbox = "#comic-link-target"
	SidebarStatus  = "#sidebar-status"
	ConfirmBtn     = "button[onclick='confirmAction()']"
	CancelBtn      = "button[onclick='cancelAction()']"

	// 首页漫画卡片
	GalleryCard      = "div.gallery"
	GalleryCoverLink = "div.gallery a.cover"

	// 漫画详情页侧边栏
	LikeBtn        = "#sidebarLikeBtn"
	ArchiveBtn     = "#sidebarArchiveBtn"
	PageManageBtn  = "#sidebarPageManageBtn"
	FixBtn         = "#sidebarFixBtn"
	EditTagsBtn    = "#sidebarEditTagsBtn"
	LargeToggleBtn = "#sidebarLargeToggle"
	DeleteBtn      = "#sidebarDeleteBtn"

	// 漫画详情页缩放控制 — 注意 zoom sidebar 默认 display:none
	ZoomSidebar   = "#zoomSidebar"
	ZoomInBtn     = "#zoomInBtn"
	ZoomOutBtn    = "#zoomOutBtn"
	ResetBtn      = "#zoomResetBtn"
	ZoomSlider    = "#thumbZoomSlider"
	ZoomValue     = "#zoomValue"
	PresetBtn200  = "a.preset-btn[data-zoom='200']"
	PresetBtn400  = "a.preset-btn[data-zoom='400']"
	PresetBtn600  = "a.preset-btn[data-zoom='600']"
	PresetBtn800  = "a.preset-btn[data-zoom='800']"
	PresetBtn1000 = "a.preset-btn[data-zoom='1000']"

	// Admin 漫画比对
	CIDMain        = "#cid-main"
	CIDTarget      = "#cid-target"
	CompareBtn     = "button.btn.btn-primary[onclick='compareComics()']"
	SwapBtn        = "button.btn.btn-secondary[onclick='swapCids()']"
	MultiComicBar  = "#multi-comic-bar"
	CompareResult  = "#compare-result"
	StatsBar       = "#stats-bar"
	CompareTable   = "#compare-table-container"
	PreviewPanel   = "#preview-panel"
	LinkAction     = "#link-action"
	BtnShowCurrent = "#btn-show-current"
	BtnShowAll     = "#btn-show-all"
	LinkedTable    = "#linked-table-container"
	ComicInfoPair  = "#comic-info-pair"

	// 通用
	Messages       = "#messages"
	ThumbContainer = "div.thumbs"
	Cover          = "#cover"
)
