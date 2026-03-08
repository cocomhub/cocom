<!doctype html>
<html lang="en" class=" theme-black unauthenticated">

<head>
    <meta charset="utf-8" />
    <meta name="theme-color" content="#1f1f1f" />
    <meta name="viewport" content="width=device-width, initial-scale=1, user-scalable=yes, viewport-fit=cover" />
    <meta name="description" content="Browse {{.Total}} {{.TagType}} on nhentai, a hentai doujinshi and manga reader." />
    <title>
        {{.TagType}}
        &raquo; nhentai: hentai doujinshi and manga</title>
{{template "head.common.tpl"}}
</head>

<body>
{{template "navigation.tpl" .}}
    <div id="messages"></div>
    <div id="content">
        <!-- <section class="container advertisement advt">
            <iframe width="728" height="90" scrolling="no" frameborder="0" src="https://a.adtng.com/get/10000816?time=1639179179273" allowtransparency="true" marginheight="0" marginwidth="0" name="spot_id_10000816"></iframe>
        </section> -->
        <div class="sort">
            <div class="sort-type"><a href="{{$.URL}}" {{if eq $.SortType 0}}class="current"{{end}}>A-Z</a></div>
            <div class="sort-type"><a href="{{$.URL}}?sortType=popular" {{if eq $.SortType 1}}class="current"{{end}}>Popular</a></div>
        </div>
{{if eq $.SortType 0}}
        <ul class="alphabetical-pagination">
    {{range $idx, $tagIndex := $.TagIndices}}
            <li><a href="{{$.URL}}?page={{$tagIndex.Page}}#{{$tagIndex.Name}}" {{if eq $tagIndex.Page $.CurPage}}class="current"{{end}}>{{$tagIndex.Name}}</a></li>
    {{end}}
        </ul>
        <div class="container" id="tag-container">
        {{range $index, $tagsSection := .TagsSections}}
            <section id="{{$tagsSection.Name}}">
                <h2>{{$tagsSection.Name}}</h2>
                {{range $index2, $tag := $tagsSection.Tags}}
                <a href="/tag{{$tag.URL}}" class="tag tag-{{$tag.ID}} {{if $tag.Like}}tag-like{{end}}">
                    <span class="name">{{$tag.Name}}</span>
                    <span class="count">{{$tag.Count}}</span></a>
                {{end}}
            </section>
        {{end}}
        </div>
{{else if eq $.SortType 1}}
        <div class="container" id="tag-container">
        {{range $index, $tagsSection := .TagsSections}}
            {{range $index2, $tag := $tagsSection.Tags}}
            <a href="/tag{{$tag.URL}}" class="tag tag-{{$tag.ID}} {{if $tag.Like}}tag-like{{end}}">
                <span class="name">{{$tag.Name}}</span>
                <span class="count">{{$tag.Count}}</span></a>
            {{end}}
        {{end}}
        </div>
{{end}}
        <section class="pagination">
            {{if ne .CurPage 1}}<a href="{{$.URL}}?page=1{{if eq $.SortType 1}}&sortType=popular{{end}}" class="first"><i class="fa fa-chevron-left"></i><i class="fa fa-chevron-left"></i></a>{{end}}
            {{if ne .CurPage 1}}<a href="{{$.URL}}?page={{Add .CurPage -1}}{{if eq $.SortType 1}}&sortType=popular{{end}}" class="previous"><i class="fa fa-chevron-left"></i></a>{{end}}
            {{range $index, $num := .PageNumList}}<a href="{{$.URL}}?page={{$num}}{{if eq $.SortType 1}}&sortType=popular{{end}}" class="page{{if eq $.CurPage $num}} current{{end}}">{{$num}}</a>{{end}}
            {{if ne .CurPage .LastPage}}<a href="{{$.URL}}?page={{Add .CurPage 1}}{{if eq $.SortType 1}}&sortType=popular{{end}}" class="next"><i class="fa fa-chevron-right"></i></a>{{end}}
            {{if ne .CurPage .LastPage}}<a href="{{$.URL}}?page={{.LastPage}}{{if eq $.SortType 1}}&sortType=popular{{end}}" class="last"><i class="fa fa-chevron-right"></i><i class="fa fa-chevron-right"></i></a>{{end}}

            <div class="ios-mobile-webkit-bottom-spacing">
                &nbsp; &nbsp;
            </div>
        </section>
    <script>
        window._n_app = new App({
            csrf_token: "{{.CSRFToken}}",
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
