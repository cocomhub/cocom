{{define "head.common.tpl"}}
    <link rel="stylesheet" href="/static/cdnjs.cloudflare.com/ajax/libs/font-awesome/5.13.0/css/all.min.css" />
    <link rel="stylesheet" href="/static/fonts.googleapis.com/css?family=Noto+Sans:400,400i,700" />
    <link rel="stylesheet" href="/static/static.nhentai.net/css/styles.3880fca2c456.css" />
    <link rel="stylesheet" href="/static/custom/css/styles.css" />
    <script src="/static/static.nhentai.net/js/scripts.cad159183e0d.js"></script>
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
    <script src="/static/custom/js/modules/search-autocomplete.js"></script>
    <script src="/static/custom/js/modules/recommend.js"></script>
    <script src="/static/custom/js/scripts.js"></script>
    <script src="/static/custom/js/tag_relation.js"></script>
{{end}}

{{define "navigation.tpl"}}
    <nav role="navigation">
        <a class="logo" href="/">
            <img src="/static/static.nhentai.net/img/logo.090da3be7b51.svg" alt="logo" width="46" height="30">
        </a>
        <form role="search" action="/search/" class="search">
            <input required type="search" name="q" value="" autocapitalize="none" placeholder="搜索漫画编号或标题..." />
            <button type="submit" class="btn btn-primary btn-square"><i class="fa fa-search fa-lg"></i></button>
        </form>
        <button type="button" class="btn btn-secondary btn-square" id="hamburger">
            <span class="line"></span>
            <span class="line"></span>
            <span class="line"></span>
        </button>
        <div class="collapse">
            <ul class="menu left">
                <li class="desktop ">
                    <a href="/random/">Random</a>
                </li>
                <li class="desktop {{if .IsNavigationActive "tags"}}active{{end}}">
                    <a href="/list/tags/">Tags</a>
                </li>
                <li class="desktop {{if .IsNavigationActive "artists"}}active{{end}}">
                    <a href="/list/artists/">Artists</a>
                </li>
                <li class="desktop {{if .IsNavigationActive "characters"}}active{{end}}">
                    <a href="/list/characters/">Characters</a>
                </li>
                <li class="desktop {{if .IsNavigationActive "parodies"}}active{{end}}">
                    <a href="/list/parodies/">Parodies</a>
                </li>
                <li class="desktop {{if .IsNavigationActive "groups"}}active{{end}}">
                    <a href="/list/groups/">Groups</a>
                </li>
                <li class="desktop ">
                    <a href="/info/">Info</a>
                </li>
                <li class="desktop ">
                    <a href="/admin">Admin</a>
                </li>
                <li class="desktop">
                    <a href="https://twitter.com/nhentaiOfficial">
                        <i class="fab fa-twitter fa-lg"></i>
                    </a>
                </li>
                <li class="dropdown">
                    <button class="btn btn-secondary btn-square" type="button" id="dropdown">
                        <i class="fa fa-chevron-down"></i>
                    </button>
                    <ul class="dropdown-menu">
                        <li>
                            <a href="/random/">Random</a>
                        </li>
                        <li>
                            <a href="/tags/">Tags</a>
                        </li>
                        <li>
                            <a href="/artists/">Artists</a>
                        </li>
                        <li>
                            <a href="/characters/">Characters</a>
                        </li>
                        <li>
                            <a href="/parodies/">Parodies</a>
                        </li>
                        <li>
                            <a href="/groups/">Groups</a>
                        </li>
                        <li>
                            <a href="/info/">Info</a>
                        </li>
                        <li>
                            <a href="https://twitter.com/nhentaiOfficial">
                                <i class="fab fa-twitter fa-lg"></i>
                            </a>
                        </li>
                    </ul>
                </li>
            </ul>
            <ul class="menu right">
                <li class="menu-sign-in">
                    <a href="/login/?next={{$.URL}}">
                        <i class="fa fa-sign-in-alt"></i> Sign in
                    </a>
                </li>
                <li class="menu-register">
                    <a href="/register/">
                        <i class="fa fa-edit"></i> Register
                    </a>
                </li>
            </ul>
        </div>
    </nav>
{{end}}
