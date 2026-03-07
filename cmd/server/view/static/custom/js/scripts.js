/**
 * Copyright 2026 The Cocomhub Authors. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

function addLikeGroup(cid) {
    let xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/api/comic/addLikeGroup?cid='+cid);

    xhr.onload = function() {
        console.log(xhr.response);
    };

    xhr.send();
}