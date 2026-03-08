/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

function addLikeGroup(cid) {
    const btn = document.getElementById('addLikeGroup');
    const liked = btn.classList.contains('btn-primary');
    const xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open(liked ? 'DELETE' : 'POST', '/api/comic/tags/like');
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    xhr.onload = function() {
        if (xhr.status >= 200 && xhr.status < 300) {
            if (liked) {
                btn.classList.remove('btn-primary');
                btn.classList.add('btn-secondary');
                removeLikeTag();
            } else {
                btn.classList.remove('btn-secondary');
                btn.classList.add('btn-primary');
                addLikeTag();
            }
        } else {
            console.error('like request failed:', xhr.status, xhr.responseText);
        }
    };
    xhr.onerror = function() {
        console.error('like request network error');
    };
    xhr.send('cid=' + encodeURIComponent(cid));
}

function findCustomsContainer() {
    const containers = document.querySelectorAll('.tag-container.field-name');
    for (const c of containers) {
        const text = (c.textContent || '').trim();
        if (text.startsWith('Customs')) {
            return c;
        }
    }
    return null;
}

function addLikeTag() {
    const container = findCustomsContainer();
    if (!container) return;
    container.classList.remove('hidden');
    const span = container.querySelector('span.tags');
    if (!span) return;
    if (span.querySelector('.tag-99999')) return;
    const a = document.createElement('a');
    a.href = '/tag/custom/like/';
    a.className = 'tag tag-99999';
    const name = document.createElement('span');
    name.className = 'name';
    name.textContent = 'like';
    const count = document.createElement('span');
    count.className = 'count';
    count.textContent = '1';
    a.appendChild(name);
    a.appendChild(count);
    span.appendChild(a);
}

function removeLikeTag() {
    const container = findCustomsContainer();
    if (!container) return;
    const span = container.querySelector('span.tags');
    if (!span) return;
    const a = span.querySelector('.tag-99999');
    if (a) {
        a.remove();
    }
    if (!span.querySelector('a.tag')) {
        container.classList.add('hidden');
    }
}