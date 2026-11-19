// Buttons at the top of every page: copy the page as Markdown, open its Markdown, or ask Claude about it.
// The Markdown versions are written by the mkdocs-llmstxt plugin next to each page (index.md).
(function () {
  function siteRoot() {
    var base = document.querySelector("base");
    var href = base ? base.href : new URL(".", window.location.href).href;
    var logo = document.querySelector(".md-header__button.md-logo");
    return logo ? logo.href : href;
  }

  function markdownUrl() {
    var url = new URL(window.location.href);
    url.hash = "";
    url.search = "";
    if (!url.pathname.endsWith("/")) url.pathname += "/";
    return url.href + "index.md";
  }

  function render() {
    var article = document.querySelector("article.md-content__inner");
    if (!article || article.querySelector(".ct-ai-actions")) return;

    var md = markdownUrl();
    var root = siteRoot();
    var prompt =
      "Read " + md + " (catude documentation; the full docs are at " + root + "llms-full.txt) " +
      "and explain it to me step by step. I use Claude Code. Give me the exact commands and keys to try.";

    var bar = document.createElement("div");
    bar.className = "ct-ai-actions";
    bar.setAttribute("aria-label", "Use this page with an AI assistant");

    var copy = document.createElement("button");
    copy.type = "button";
    copy.textContent = "Copy page as Markdown";
    copy.addEventListener("click", function () {
      fetch(md)
        .then(function (r) { if (!r.ok) throw new Error(r.status); return r.text(); })
        .then(function (text) { return navigator.clipboard.writeText(text); })
        .then(function () { copy.textContent = "Copied"; })
        .catch(function () { copy.textContent = "Could not copy: open the Markdown instead"; })
        .finally(function () { setTimeout(function () { copy.textContent = "Copy page as Markdown"; }, 2500); });
    });

    var view = document.createElement("a");
    view.href = md;
    view.textContent = "View as Markdown";

    var ask = document.createElement("a");
    ask.href = "https://claude.ai/new?q=" + encodeURIComponent(prompt);
    ask.target = "_blank";
    ask.rel = "noopener";
    ask.textContent = "Ask Claude about this page";

    bar.append(copy, view, ask);
    var h1 = article.querySelector("h1");
    if (h1) h1.after(bar); else article.prepend(bar);
  }

  if (window.document$ && typeof window.document$.subscribe === "function") {
    window.document$.subscribe(render); // Material's instant navigation
  } else {
    document.addEventListener("DOMContentLoaded", render);
  }
})();
