document.body.addEventListener("htmx:beforeOnLoad", function (event) {
    if (event.detail.xhr.status >= 400 && evt.detail.xhr.status < 600) {
        event.detail.shouldSwap = true;
        event.detail.isError = false;
    }
});

// document.body.addEventListener("htmx:afterSettle", function (event) {
//     if (event.detail.boosted && window.Alpine) {
//         window.Alpine.destroyTree(document.body)
//         window.Alpine.initTree(document.body)
//     }
// })
