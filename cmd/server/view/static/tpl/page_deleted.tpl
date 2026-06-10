{{/* 已删除漫画提示页 */}}
{{define "content"}}
<div class="container" style="text-align:center;padding:80px 20px;">
  <div style="font-size:64px;margin-bottom:20px;">🗑️</div>
  <h1 style="color:#e53935;">该漫画已被删除</h1>
  <p style="color:#999;font-size:16px;margin-top:12px;">
    漫画 CID: <strong>{{.CID}}</strong>
  </p>
  <a href="/" class="btn btn-primary" style="margin-top:20px;">返回首页</a>
</div>
{{end}}
