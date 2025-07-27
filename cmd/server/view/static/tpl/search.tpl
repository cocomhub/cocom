{{define "search_post.tpl"}}
    <form role="search" action="/search" class="search" id="searchForm">
        <input 
            required 
            type="search" 
            name="q" 
            id="searchInput" 
            value="" 
            autocapitalize="none" 
            placeholder="e.g. #110631" 
        />
        <button type="submit" class="btn btn-primary btn-square">
            <i class="fa fa-search fa-lg"></i>
        </button>
    </form>

    <script>
    document.getElementById('searchForm').addEventListener('submit', function(event) {
        event.preventDefault(); // 阻止默认提交行为
        const searchValue = document.getElementById('searchInput').value.trim();
        if (searchValue) {
            // 将搜索词编码后拼接到URL路径中
            window.location.href = `/search/${encodeURIComponent(searchValue)}`;
        }
    });
    </script>
{{end}}

{{define "search.tpl"}}
    <form role="search" action="/search/" class="search">
         <input required type="search" name="q" value="" autocapitalize="none" placeholder="e.g. #110631" />
        <button type="submit" class="btn btn-primary btn-square"><i class="fa fa-search fa-lg"></i></button>
    </form>
{{end}}