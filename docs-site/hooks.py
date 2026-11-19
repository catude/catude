"""MkDocs hooks for the documentation site.

The pages in docs/ are also read on GitHub, so they link to the repository the way GitHub needs:
images as ../assets/..., and files outside docs/ (SECURITY.md, CONTRIBUTING.md...) by relative path.
These hooks make the same Markdown work on the site:

- every file in assets/ is published at /assets/ (no copy kept in docs/);
- ../assets/ links become site-relative, and links to files outside docs/ point to GitHub.
"""

from __future__ import annotations

import os
import re

from mkdocs.structure.files import File, Files

REPO = "https://github.com/catude/catude/blob/main/"
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
ASSETS = os.path.join(ROOT, "assets")

# ](../something) or ](../../something), not followed by "assets/"
_OUTSIDE = re.compile(r"\]\((?:\.\./)+(?!assets/)([^)#\s]+)(#[^)\s]*)?\)")


def on_files(files: Files, config) -> Files:
    for folder, _, names in os.walk(ASSETS):
        for name in names:
            rel = os.path.relpath(os.path.join(folder, name), ROOT).replace(os.sep, "/")
            files.append(
                File(rel, src_dir=ROOT, dest_dir=config["site_dir"], use_directory_urls=config["use_directory_urls"])
            )
    return files


def on_page_markdown(markdown: str, page, config, files) -> str:
    depth = page.file.src_uri.count("/")  # 0 for docs/x.md, 1 for docs/translations/x.md
    site_assets = "../" * depth + "assets/"
    markdown = re.sub(r"(\]\(|src=\")(?:\.\./)+assets/", lambda m: m.group(1) + site_assets, markdown)
    here = os.path.dirname(page.file.src_uri)

    def outside(m: re.Match) -> str:
        target = os.path.normpath(os.path.join("docs", here, m.group(0)[2:-1].split("#")[0])).replace(os.sep, "/")
        if target.startswith("docs/"):
            return m.group(0)  # still inside docs/: MkDocs resolves it
        return f"]({REPO}{target}{m.group(2) or ''})"

    return _OUTSIDE.sub(outside, markdown)
