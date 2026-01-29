document.body.addEventListener("htmx:beforeOnLoad", function (evt) {
    if (evt.detail.xhr.status >= 400 && evt.detail.xhr.status < 600) {
        evt.detail.shouldSwap = true;
        evt.detail.isError = false;
    }
});
