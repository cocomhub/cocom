<!doctype html>
<html lang="en" class=" theme-black unauthenticated">

<head>
    <meta charset="utf-8" />
    <meta name="theme-color" content="#1f1f1f" />
    <meta name="viewport" content="width=device-width, initial-scale=1, user-scalable=yes, viewport-fit=cover" />
    <meta name="description" content="nhentai is a free hentai manga and doujinshi reader with over 463,000 galleries to read and download." />
    <title>nhentai: hentai doujinshi and manga</title>
    <link rel="stylesheet" href="/static/cdnjs.cloudflare.com/ajax/libs/font-awesome/5.13.0/css/all.min.css" />
    <link rel="stylesheet" href="/static/fonts.googleapis.com/css?family=Noto+Sans:400,400i,700" />
    <link rel="stylesheet" href="/static/static.nhentai.net/css/styles.3880fca2c456.css" />
    <link rel="stylesheet" href="/static/custom/css/styles.css" />
    <script src="/static/static.nhentai.net/js/scripts.cad159183e0d.js"></script>
    <script src="/static/custom/js/scripts.js"></script>
    <script src="/static/custom/js/tag_relation.js"></script>
</head>

<body>
{{template "navigation.tpl" .}}
<div id="messages"></div>
<div id="content">
{{if .CurTag}}
    <div class="container">
        <h2><i class="fa fa-tag color-icon"></i> Tag:
            <a id="currentTagLink" href="/tag{{.CurTag.URL}}" class="tag {{if .CurTag.Like}}tag-like{{end}}">
                <span class="name">{{.CurTag.Name}}</span>
            </a>
        </h2>
        <div class="buttons">
            <a id="toggleLikeTag" class="btn {{if .CurTag.Like}}btn-primary{{else}}btn-secondary{{end}}" href="javascript:;" onclick="toggleLikeTag('{{.CurTag.Type}}','{{.CurTag.Name}}', {{.CurTag.ID}})">
                <i class="fas fa-heart"></i> like
            </a>
            <a id="manageRelationsBtn" class="btn btn-secondary" href="javascript:;" onclick="openTagRelationManager('{{.CurTag.Type}}','{{.CurTag.Name}}', {{.CurTag.ID}})">
                <i class="fa fa-link"></i> Manage Relations
            </a>
        </div>
    </div>
{{end}}
{{if .CurTag}}
    {{if gt (len .RelatedTags) 0}}
    <div class="container" style="margin-top:10px;">
        <h3><i class="fa fa-link color-icon"></i> Related Tags</h3>
        <div id="related-tags-content">
            {{range $tag := .RelatedTags}}
            <a href="/tag{{$tag.URL}}" class="tag tag-{{$tag.ID}} {{if $tag.Like}}tag-like{{end}} {{if $tag.Explicit}}tag-explicit{{end}}"
               {{if $tag.Explicit}}title="Explicit relation"{{end}}>
                <span class="name">{{$tag.Name}}</span>
                <span class="count">{{if $tag.Explicit}}★{{else}}{{$tag.Count}}{{end}}</span>
            </a>
            {{end}}
        </div>
    </div>
    {{end}}
{{end}}
{{if .SearchQuery}}
    <div class="container" style="margin-top:5px;">
        <a id="alignTagsBtn" class="btn btn-secondary" href="javascript:;" onclick="openTagAligner('{{.SearchQuery}}')">
            <i class="fa fa-tags"></i> Align Tags
        </a>
    </div>
{{end}}
<!--    <section class="container advertisement advt">-->
<!--        <iframe width="728" height="90" scrolling="no" frameborder="0" src="https://a.adtng.com/get/10000815?time=1639179157904" allowtransparency="true" marginheight="0" marginwidth="0" name="spot_id_10000815"></iframe>-->
<!--    </section>-->
{{if eq .CurPage 1}}
    <div class="container index-container index-popular">
        <h2><i class="fa fa-fire color-icon"></i> Popular Now</h2>
{{range $index, $detail := .PopularNow}}
        <div class="gallery" data-tags="{{.Tags.IdString}}">
            <a href="/g/{{$detail.CID}}/" class="cover" style="padding:0 0 145.6% 0">
                <img class="lazyload" width="250" height="364" data-src="/galleries/{{$detail.ShowMediaId}}/{{$detail.Images.ThumbnailName}}" src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" /><noscript>
                <img src="/galleries/{{$detail.ShowMediaId}}/{{$detail.Images.ThumbnailName}}" width="250" height="364"  /></noscript>
                <div class="caption">{{$detail.Title.English}}</div>
            </a>
        </div>
{{end}}
    </div>
{{end}}
    <div class="container index-container">

        <h2><i class="fa fa-box-tissue color-icon"></i> New Uploads</h2>

{{range $index, $detail := .NewUpdates}}
            <div class="gallery" data-tags="{{$detail.Tags.IdString}}">
                <a href="/g/{{$detail.CID}}/" class="cover" style="padding:0 0 145.6% 0">
                    <img class="lazyload" width="250" height="364" data-src="/galleries/{{$detail.ShowMediaId}}/{{$detail.Images.ThumbnailName}}" src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" /><noscript>
                    <img src="/galleries/{{$detail.ShowMediaId}}/{{$detail.Images.ThumbnailName}}" width="250" height="364"  /></noscript>
                    <div class="caption">{{$detail.Title.English}}</div>
                </a>
            </div>
{{end}}

    </div>

    <section class="pagination">

        {{if ne .CurPage 1}}<a href="{{$.URL}}?page=1{{if ne $.SearchQuery ""}}&q={{$.SearchQuery}}{{end}}" class="first"><i class="fa fa-chevron-left"></i><i class="fa fa-chevron-left"></i></a>{{end}}
        {{if ne .CurPage 1}}<a href="{{$.URL}}?page={{Add .CurPage -1}}{{if ne $.SearchQuery ""}}&q={{$.SearchQuery}}{{end}}" class="previous"><i class="fa fa-chevron-left"></i></a>{{end}}
        {{range $index, $num := .PageNumList}}<a href="{{$.URL}}?page={{$num}}{{if ne $.SearchQuery ""}}&q={{$.SearchQuery}}{{end}}" class="page{{if eq $.CurPage $num}} current{{end}}">{{$num}}</a>{{end}}
        {{if ne .CurPage .LastPage}}<a href="{{$.URL}}?page={{Add .CurPage 1}}{{if ne $.SearchQuery ""}}&q={{$.SearchQuery}}{{end}}" class="next"><i class="fa fa-chevron-right"></i></a>{{end}}
        {{if ne .CurPage .LastPage}}<a href="{{$.URL}}?page={{.LastPage}}{{if ne $.SearchQuery ""}}&q={{$.SearchQuery}}{{end}}" class="last"><i class="fa fa-chevron-right"></i><i class="fa fa-chevron-right"></i></a>{{end}}

            <span class="page-jump">
                跳至 <input type="number" class="jump-input" min="1" max="{{.LastPage}}"
                    onkeydown="if(event.key==='Enter') jumpToPage(this, '{{$.URL}}', '{{$.SearchQuery}}')" /> 页
                <button class="btn btn-secondary btn-square jump-go"
                    onclick="jumpToPage(this.previousElementSibling, '{{$.URL}}', '{{$.SearchQuery}}')">GO</button>
            </span>

        <div class="ios-mobile-webkit-bottom-spacing">
            &nbsp; &nbsp;
        </div>
    </section>
</div>

<script>
    window._n_app = new App({
        csrf_token: "VgwNPqB0rKTfAPcTWcv3LGEDJY40j2AgtNheeysH8XqKVik8I35EiO9afuJ9lczy",
        user: {},
        blacklisted_tags: null,
        media_server: 7,
        ads: {
            show_popunders: true
        }
    });
</script>
</body>

</html>
