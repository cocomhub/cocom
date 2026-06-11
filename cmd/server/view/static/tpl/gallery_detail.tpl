<!doctype html>
<html lang="en" class=" theme-black unauthenticated">

<head>
    <meta charset="utf-8" />
    <meta name="theme-color" content="#1f1f1f" />
    <meta itemprop="name" content="{{TitlePretty .Title.English}}" />
    <meta itemprop="image" content="/galleries/{{.ShowMediaId}}/{{.CoverName}}" />
    <meta property="og:type" content="article" />
    <meta property="og:title" content="{{TitlePretty .Title.English}}" />
    <meta property="og:image" content="/galleries/{{.ShowMediaId}}/{{.CoverName}}" />
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="{{TitlePretty .Title.English}}" />
    <meta name="twitter:description" content="{{(.Tags.SubTypeTags Tag).IdString}}" />
    <meta name="viewport" content="width=device-width, initial-scale=1, user-scalable=yes, viewport-fit=cover" />
    <meta name="description" content="Read and download {{TitlePretty .Title.English}}, a hentai manga by {{(.Tags.SubTypeTags Artist).NameString}} for free on nhentai." />
    <title>{{TitlePretty .Title.English}} &raquo;nhentai: hentai doujinshi and manga</title>
{{template "head.common.tpl"}}
</head>

<body>
{{template "navigation.tpl" .}}
    <div id="messages"></div>
    <div id="content">
        <!-- 操作侧边栏（互换到右侧） -->
        <div class="left-action-sidebar" style="left:auto;right:0;border-radius:8px 0 0 8px;">
            <a id="sidebarLikeBtn" class="sidebar-btn {{if .HasLike}}btn-primary{{end}}" href="javascript:;" onclick="addLikeGroup({{.CID}})">
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
            <hr class="sidebar-divider">
            <a id="sidebarPageManageBtn" class="sidebar-btn" href="javascript:;" onclick="togglePageManager()">
                <i class="fa fa-file-image-o"></i>
                <span class="label">页管理</span>
            </a>
            <a id="sidebarFixBtn" class="sidebar-btn" href="javascript:;" onclick="verifyComic({{.CID}})">
                <i class="fa fa-wrench"></i>
                <span class="label">修复</span>
            </a>
            <a id="sidebarEditTagsBtn" class="sidebar-btn" href="javascript:;" onclick="openTagEditor({{.CID}})">
                <i class="fa fa-tags"></i>
                <span class="label">编辑Tag</span>
            </a>
            <a id="sidebarLargeToggle" class="sidebar-btn" href="javascript:;" onclick="toggleLargeMode()">
                <i class="fa fa-expand"></i>
                <span class="label">大图模式</span>
            </a>
            <hr class="sidebar-divider">
            <a id="sidebarDeleteBtn" class="sidebar-btn btn-danger" href="javascript:;" onclick="openDeleteConfirm()">
                <i class="fa fa-trash-o"></i>
                <span class="label">删除</span>
            </a>
        </div>
        <script>window.reArchive = function(){ archiveComic({{.CID}}); };</script>
        <!-- 缩放侧边栏（互换到左侧） -->
        <div class="right-zoom-sidebar" id="zoomSidebar" style="display:none;left:10px;right:auto;border-radius:0 8px 8px 0;">
            <div class="zoom-title">缩放</div>
            <button type="button" class="btn btn-secondary zoom-btn" id="zoomInBtn" title="放大">+</button>
            <input type="range" id="thumbZoomSlider" min="60" max="1200" value="1200" step="20" />
            <button type="button" class="btn btn-secondary zoom-btn" id="zoomOutBtn" title="缩小">&minus;</button>
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
        <div class="zoom-float-btn" id="zoomFloatBtn" onclick="toggleMobileZoom()">&#x1F50D;</div>
        <!-- <section class="container advertisement advt">
            <iframe width="728" height="90" scrolling="no" frameborder="0" src="https://a.adtng.com/get/10000816?time=1639179179273" allowtransparency="true" marginheight="0" marginwidth="0" name="spot_id_10000816"></iframe>
        </section> -->
        <div class="container" id="bigcontainer">
            <div id="cover">
                <a href="/g/{{.CID}}/1/">
                    <img class="lazyload" width="350" data-src="/galleries/{{.ShowMediaId}}/{{.CoverName}}" src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" />
                    <noscript>
                            <img src="/galleries/{{.ShowMediaId}}/{{.CoverName}}" width="350" height="494"/>
                        </noscript>
                </a>
            </div>
            <div id="info-block">
                <div id="info">
                    <h1 class="title">
                        <span class="before">{{TitleBefore .Title.English}}</span>
                        <span class="pretty">{{TitlePretty .Title.English}}</span>
                        <span class="after">{{TitleAfter .Title.English}}</span>
                    </h1>
                    <h2 class="title">
                        <span class="before">{{TitleBefore .Title.Japanese}}</span>
                        <span class="pretty">{{TitlePretty .Title.Japanese}}</span>
                        <span class="after">{{TitleAfter .Title.Japanese}}</span>
                    </h2>
                    <h3 id="gallery_id">
                        <span class="hash">#</span> {{.CID}}
                    </h3>
                    <section id="tags">
{{range $index, $TagType := TagTypeList}}
                        <div class="tag-container field-name {{if eq (len ($.Tags.SubTypeTags $TagType.Name)) 0}}hidden{{end}}">
                            {{$TagType.FieldName}}:
                            <span class="tags">
{{range $index, $tag := ($.Tags.SubTypeTags $TagType.Name)}}
                            <a href="/tag{{$tag.URL}}" class="tag tag-{{$tag.ID}} {{if $.IsTagLiked $tag}}tag-like{{end}}">
                                        <span class="name">{{$tag.Name}}</span>
                            <span class="count">{{$tag.Count}}</span>
                            </a> {{end}}
                            </span>
                        </div> {{end}}
                        <div class="tag-container field-name">
                            Pages:
                            <span class="tags">
                            <a href="#" class="tag">
                                <span class="name">{{.NumPages}}</span>
                            </a>
                            </span>
                        </div>
                        <div class="tag-container field-name">
                            Uploaded:
                            <span class="tags">
                                <time class="nobold" datetime="{{.UploadDate}}"></time>
                            </span>
                        </div>>
                    </section>
                    <div class="buttons">
                        <a class="btn btn-primary btn-disabled tooltip">
                            <i class="fas fa-heart"></i>
                            <span>Favorite <span class="nobold">(1911)</span></span>
                            <div class="top">You need to log in to add favorites<i></i></div>
                        </a>
                        <a id="download" class="btn btn-secondary btn-disabled tooltip">
                            <i class="fa fa-download"></i> Download
                            <div class="top">You need to log in to download<i></i></div>
                        </a>
                    </div>
                </div>
            </div>
        </div>
        <div class="container with-sidebars" id="thumbnail-container">

        {{/* 页管理操作栏 */}}
        <div id="page-manager-bar" class="page-manager-bar" style="display:none;">
          <span class="pm-title">📄 页管理</span>
          <div class="pm-actions">
            <button class="btn btn-danger btn-sm" onclick="pmDeleteMode()">删除</button>
            <button class="btn btn-secondary btn-sm" onclick="pmInsertMode()">插入</button>
            <button class="btn btn-secondary btn-sm" onclick="pmReplaceMode()">替换</button>
            <button class="btn btn-secondary btn-sm" onclick="pmReorderMode()">重排</button>
            <span class="pm-sep">|</span>
            <button class="btn btn-secondary btn-sm" onclick="pmUndo()">撤销</button>
            <button class="btn btn-primary btn-sm" onclick="pmSave()">保存</button>
            <button class="btn btn-secondary btn-sm" onclick="pmExit()">退出</button>
          </div>
          <div class="pm-status" id="pm-status">未保存变更: 0</div>
        </div>

        {{/* 插入表单 */}}
        <div id="insert-form" class="insert-form" style="display:none;">
          <div class="insert-form-inner">
            <label>源 CID: <input type="number" id="insert-source-cid" class="form-control" style="width:100px;display:inline;"></label>
            <button class="btn btn-secondary btn-sm" onclick="pmFetchPreview()">获取页面</button>
            <label style="margin-left:8px;">插入到第 <input type="number" id="insert-after-page" class="form-control" style="width:60px;display:inline;"> 页之后</label>
            <button class="btn btn-primary btn-sm" onclick="pmConfirmInsert()" style="margin-left:8px;">确认插入</button>
            <button class="btn btn-secondary btn-sm" onclick="pmCancelInsert()">取消</button>
          </div>
          <div id="insert-preview" class="insert-preview-row" style="display:none;"></div>
        </div>

        {{/* 归档过期横幅 */}}
        {{if .ArchiveStale}}
        <div id="archive-stale-banner" class="archive-stale-banner">
          <i class="fa fa-exclamation-triangle"></i>
          ⚠ 页面内容已变更，存档已过期。
          <button class="btn btn-warning btn-sm" onclick="reArchive()">重新归档</button>
        </div>
        {{end}}
            <div class="thumbs">
            {{range $index, $page := .Images.Pages}}
                <div class="thumb-container">
                    <a class="gallerythumb" href="/g/{{$.CID}}/{{Add $index 1}}/" rel="nofollow">
                        <img class="lazyload" width="200" height="282" data-src="/galleries/{{$.ShowMediaId}}/{{$.Images.PageThumbnailNameByIndex $index}}?w=200" data-original-src="/galleries/{{$.ShowMediaId}}/{{$.Images.PageThumbnailNameByIndex $index}}" src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" />
                        <noscript>
                                <img src="/galleries/{{$.ShowMediaId}}/{{$.Images.PageThumbnailNameByIndex $index}}" width="200" height="282"/>
                            </noscript>
                    </a>
                </div> {{end}}
            </div>
        </div>
        <!-- 多维推荐容器（异步加载） -->
        <div class="container" id="recommend-container" data-cid="{{.CID}}" style="display:none;">
            <section class="recommend-section" data-recommend-type="artist">
                <div class="recommend-header">
                    <h2>同作者 · More by Artist</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'artist')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="group">
                <div class="recommend-header">
                    <h2>同团体 · More from Group</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'group')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="parody">
                <div class="recommend-header">
                    <h2>同系列 · More from Parody</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'parody')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="character">
                <div class="recommend-header">
                    <h2>同角色 · More by Character</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'character')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="tag">
                <div class="recommend-header">
                    <h2>同标签 · More Like This</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'tag')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
        </div>
        <!-- 多维推荐容器（异步加载） -->
        <div id="recommend-container" data-cid="{{.CID}}" style="display:none;">
            <section class="recommend-section" data-recommend-type="artist">
                <div class="recommend-header">
                    <h2>同作者 · More by Artist</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'artist')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="group">
                <div class="recommend-header">
                    <h2>同团体 · More from Group</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'group')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="parody">
                <div class="recommend-header">
                    <h2>同系列 · More from Parody</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'parody')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="character">
                <div class="recommend-header">
                    <h2>同角色 · More by Character</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'character')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
            <section class="recommend-section" data-recommend-type="tag">
                <div class="recommend-header">
                    <h2>同标签 · More Like This</h2>
                    <button class="btn btn-secondary btn-sm recommend-refresh" onclick="refreshRecommend(this, 'tag')" title="重新获取">
                        <i class="fa fa-sync-alt"></i>
                    </button>
                </div>
                <div class="recommend-grid">
                    <div class="skeleton-grid">
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                        <div class="skeleton-card"><div class="skeleton-thumb"></div><div class="skeleton-line"></div></div>
                    </div>
                </div>
            </section>
        </div>
        <div class="container" id="comment-post-container">
            <h3>
                <i class="fa fa-comments color-icon"></i> Post a comment
            </h3>
            <div class="row">
                <p>
                    <a class="login-comment" href="/login/">Login</a> or <a class="login-comment" href="/register/">register</a> to post a comment.

                </p>
            </div>
        </div>
        <div class="container" id="comment-container">
            <div id="comments"></div>
        </div>
    </div>
    <script>
        window._n_app = new App({
            csrf_token: "{{.CSRFToken}}",
            user: {},
            blacklisted_tags: null,
            media_server: 3,
            ads: {
                show_popunders: true
            }
        });
    </script>
    <script>
        window._gallery = JSON.parse("{{.GalleryRawStr}}");
    </script>
</body>

</html>
