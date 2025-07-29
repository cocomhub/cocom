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
                            <a href="/tag{{$tag.URL}}" class="tag tag-{{$tag.ID}} ">
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
                            <span>
                                    Favorite <span class="nobold">(1911)</span>
                            </span>
                            <div class="top">
                                You need to log in to add favorites<i></i>
                            </div>
                        </a>
                        <a id="addLikeGroup" class="btn btn-primary btn-disabled tooltip" href="javascript:;" onclick="addLikeGroup({{.CID}})">
                            <i class="fas fa-heart"></i> like
                        </a>
                        <a id="download" class="btn btn-secondary btn-disabled tooltip">
                            <i class="fa fa-download"></i> Download
                            <div class="top">
                                You need to log in to download<i></i>
                            </div>
                        </a>
                    </div>
                </div>
            </div>
        </div>
        <div class="container" id="thumbnail-container">
            <div class="thumbs">
            {{range $index, $page := .Images.Pages}}
                <div class="thumb-container{{if $.EnableLarge}}-large{{end}}">
                    <a class="gallerythumb" href="/g/{{$.CID}}/{{Add $index 1}}/" rel="nofollow">
                        <img class="lazyload" {{if not $.EnableLarge}}width="200" height="282"{{end}} data-src="/galleries/{{$.ShowMediaId}}/{{$.Images.PageThumbnailNameByIndex $index}}" src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" />
                        <noscript>
                                <img src="/galleries/{{$.ShowMediaId}}/{{$.Images.PageThumbnailNameByIndex $index}}" width="200" height="282"/>
                            </noscript>
                    </a>
                </div> {{end}}
            </div>
        </div>
        <!-- <section class="container advertisement advt">
            <div id="ts_ad_native_ld0p1" style="min-height:250px"></div>
        </section> -->
        <div class="container" id="related-container">
            <h2>More Like This</h2>
            {{range .MoreLikeThis}}
            <div class="gallery" data-tags="{{$.Tags.IdString}}">
                <a href="/g/{{$.CID}}/" class="cover" style="padding:0 0 141.6% 0">
                    <img class="lazyload" width="250" height="354" data-src="/galleries/{{$.ShowMediaId}}/{{$.Images.ThumbnailName}}" src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" />
                    <noscript>
                        <img src="/galleries/{{$.ShowMediaId}}/{{$.Images.ThumbnailName}}" width="250" height="354"/>
                    </noscript>
                    <div class="caption">{{$.Title.English}}</div>
                </a>
            </div>{{end}}
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