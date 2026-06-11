<!doctype html>
<html lang="en" class=" theme-black unauthenticated">

<head>
    <meta charset="utf-8" />
    <meta name="theme-color" content="#1f1f1f" />
    <meta name="viewport" content="width=device-width, initial-scale=1, user-scalable=yes, viewport-fit=cover" />
    <title>已删除 - nhentai</title>
{{template "head.common.tpl" .}}
</head>

<body>
{{template "navigation.tpl" .}}
<div id="messages"></div>
<div id="content">
<div class="container" style="text-align:center;padding:80px 20px;">
  <div style="font-size:64px;margin-bottom:20px;">🗑️</div>
  <h1 style="color:#e53935;">该漫画已被删除</h1>
  <p style="color:#999;font-size:16px;margin-top:12px;">
    漫画 CID: <strong>{{.CID}}</strong>
  </p>
  <a href="/" class="btn btn-primary" style="margin-top:20px;">返回首页</a>
</div>
</div>
</body>
</html>
