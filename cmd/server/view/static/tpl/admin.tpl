<!doctype html>
<html lang="en" class=" theme-black unauthenticated">
<head>
    <meta charset="utf-8" />
    <meta name="theme-color" content="#1f1f1f" />
    <meta name="viewport" content="width=device-width, initial-scale=1, user-scalable=yes, viewport-fit=cover" />
    <title>Admin Dashboard</title>
    {{template "head.common.tpl" .}}
</head>
<body>
{{template "navigation.tpl" .}}
<div id="messages"></div>
<div id="content">
    <div class="container index-container">
        <h2><i class="fa fa-cogs color-icon"></i> 系统设置</h2>
        <div class="gallery" style="width:100%">
            <div style="margin-bottom:8px">
                <button id="btn-load-settings" class="btn btn-secondary" onclick="loadSettings()">读取设置</button>
                <button id="btn-save-settings" class="btn btn-primary" onclick="saveSettings()">保存设置</button>
            </div>
            <div>
                <label>类型(type)：</label>
                <input id="settings-type" type="text" value="view" style="width:200px"/>
            </div>
            <div style="margin-top:8px">
                <label>键(keys，逗号分隔)：</label>
                <input id="settings-keys" type="text" value="show_status_not_true" style="width:300px"/>
            </div>
            <div style="margin-top:8px">
                <label>内容(JSON)：</label>
                <textarea id="settings-json" style="width:100%; height:160px">{}</textarea>
            </div>
            <pre id="settings-result" style="margin-top:8px"></pre>
        </div>
    </div>

    <div class="container index-container">
        <h2><i class="fa fa-trash-alt color-icon"></i> 缓存</h2>
        <button class="btn btn-primary" onclick="resetCache()">重置缓存</button>
        <pre id="cache-result" style="margin-top:8px"></pre>
    </div>

    <div class="container index-container">
        <h2><i class="fa fa-server color-icon"></i> 服务器控制</h2>
        <div style="margin-bottom:8px">
            <label>X-Admin-Token（可选）：</label>
            <input id="admin-token" type="text" style="width:300px"/>
        </div>
        <button class="btn btn-danger" onclick="shutdownServer()">关闭服务器（危险操作）</button>
        <pre id="server-result" style="margin-top:8px"></pre>
    </div>

    <div class="container index-container">
        <h2><i class="fa fa-check color-icon"></i> NHComic 校验</h2>
        <div style="margin-bottom:8px">
            <button class="btn btn-secondary" onclick="listVerifyTasks()">查看任务</button>
            <button class="btn btn-primary" onclick="startVerify()">启动校验</button>
        </div>
        <pre id="verify-result" style="margin-top:8px"></pre>
    </div>
</div>

<script>
function showResult(id, ok, data) {
    const el = document.getElementById(id);
    let text = (ok ? 'SUCCESS' : 'ERROR') + '\\n';
    try {
        text += JSON.stringify(data, null, 2);
    } catch (e) {
        text += data;
    }
    el.textContent = text;
}

async function loadSettings() {
    const type = document.getElementById('settings-type').value;
    const keys = document.getElementById('settings-keys').value;
    try {
        const resp = await fetch(`/api/settings?type=${encodeURIComponent(type)}&keys=${encodeURIComponent(keys)}`);
        const data = await resp.json();
        showResult('settings-result', resp.ok, data);
        if (resp.ok) {
            document.getElementById('settings-json').value = JSON.stringify(data.data || {}, null, 2);
        }
    } catch (e) {
        showResult('settings-result', false, String(e));
    }
}

async function saveSettings() {
    const type = document.getElementById('settings-type').value;
    let bodyObj = {};
    try {
        const raw = document.getElementById('settings-json').value || '{}';
        bodyObj = JSON.parse(raw);
    } catch (e) {
        return showResult('settings-result', false, 'JSON 解析失败: ' + e);
    }
    try {
        const resp = await fetch('/api/settings', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({type: type, settings: bodyObj})
        });
        const data = await resp.json();
        showResult('settings-result', resp.ok, data);
    } catch (e) {
        showResult('settings-result', false, String(e));
    }
}

async function resetCache() {
    if (!confirm('确认重置缓存？')) return;
    try {
        const resp = await fetch('/api/cache/reset', {method: 'POST'});
        const data = await resp.json();
        showResult('cache-result', resp.ok, data);
    } catch (e) {
        showResult('cache-result', false, String(e));
    }
}

async function shutdownServer() {
    if (!confirm('确认关闭服务器？这是危险操作。')) return;
    const token = document.getElementById('admin-token').value.trim();
    const headers = {};
    if (token) headers['X-Admin-Token'] = token;
    try {
        const resp = await fetch('/admin/server/shutdown', {method: 'POST', headers});
        let data = null;
        try { data = await resp.json(); } catch (e) { data = await resp.text(); }
        showResult('server-result', resp.ok, data);
    } catch (e) {
        showResult('server-result', false, String(e));
    }
}

async function listVerifyTasks() {
    try {
        const resp = await fetch('/v2/api/nhcomic/verify');
        const data = await resp.json();
        showResult('verify-result', resp.ok, data);
    } catch (e) {
        showResult('verify-result', false, String(e));
    }
}

async function startVerify() {
    try {
        const resp = await fetch('/v2/api/nhcomic/verify', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({})
        });
        const data = await resp.json();
        showResult('verify-result', resp.ok, data);
    } catch (e) {
        showResult('verify-result', false, String(e));
    }
}
</script>
</body>
</html>
