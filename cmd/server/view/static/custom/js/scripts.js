function addLikeGroup(cid) {
    let xhr = new XMLHttpRequest();
    xhr.withCredentials = true;
    xhr.open('POST', '/api/comic/addLikeGroup?cid='+cid);

    xhr.onload = function() {
        console.log(xhr.response);
    };

    xhr.send();
}