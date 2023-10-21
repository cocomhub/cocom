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
    <script src="/static/static.nhentai.net/js/scripts.cad159183e0d.js"></script>
</head>

<body>
<nav role="navigation">
    <a class="logo" href="/">
        <img src="/static/static.nhentai.net/img/logo.090da3be7b51.svg" alt="logo" width="46" height="30">
    </a>

    <form role="search" action="/search/" class="search">
        <input required type="search" name="q" value="" autocapitalize="none" placeholder="e.g. #110631" />
        <button type="submit" class="btn btn-primary btn-square"><i class="fa fa-search fa-lg"></i></button>
    </form>

    <button type="button" class="btn btn-secondary btn-square" id="hamburger">
        <span class="line"></span><span class="line"></span><span class="line"></span>
    </button>

    <div class="collapse">
        <ul class="menu left">
            <li class="desktop "><a href="/random/">Random</a></li>
            <li class="desktop "><a href="/tags/">Tags</a></li>
            <li class="desktop "><a href="/artists/">Artists</a></li>
            <li class="desktop "><a href="/characters/">Characters</a></li>
            <li class="desktop "><a href="/parodies/">Parodies</a></li>
            <li class="desktop "><a href="/groups/">Groups</a></li>
            <li class="desktop "><a href="/info/">Info</a></li>
            <li class="desktop"><a href="https://twitter.com/nhentaiOfficial"><i class="fab fa-twitter fa-lg"></i></a></li>
            <li class="dropdown">
                <button class="btn btn-secondary btn-square" type="button" id="dropdown"><i class="fa fa-chevron-down"></i></button>
                <ul class="dropdown-menu">
                    <li><a href="/random/">Random</a></li>
                    <li><a href="/tags/">Tags</a></li>
                    <li><a href="/artists/">Artists</a></li>
                    <li><a href="/characters/">Characters</a></li>
                    <li><a href="/parodies/">Parodies</a></li>
                    <li><a href="/groups/">Groups</a></li>
                    <li><a href="/info/">Info</a></li>
                    <li><a href="https://twitter.com/nhentaiOfficial"><i class="fab fa-twitter fa-lg"></i></a></li>
                </ul>
            </li>
        </ul>
        <ul class="menu right">
            <li class="menu-sign-in"><a href="/login/?next=/"><i class="fa fa-sign-in-alt"></i> Sign in</a></li>
            <li class="menu-register"><a href="/register/"><i class="fa fa-edit"></i> Register</a></li>
        </ul>
    </div>
</nav>
<div id="messages"></div>
<div id="content">
<!--    <section class="container advertisement advt">-->
<!--        <iframe width="728" height="90" scrolling="no" frameborder="0" src="https://a.adtng.com/get/10000815?time=1639179157904" allowtransparency="true" marginheight="0" marginwidth="0" name="spot_id_10000815"></iframe>-->
<!--    </section>-->

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
        <a href="/?page=1" class="page current">1</a>
        <a href="/?page=2" class="page">2</a>
        <a href="/?page=3" class="page">3</a>
        <a href="/?page=4" class="page">4</a>
        <a href="/?page=5" class="page">5</a>
        <a href="/?page=6" class="page">6</a>
        <a href="/?page=2" class="next"><i class="fa fa-chevron-right"></i></a>
        <a href="/?page=18552" class="last"><i class="fa fa-chevron-right"></i><i class="fa fa-chevron-right"></i></a>

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