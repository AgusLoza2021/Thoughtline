---
title: SVN externals audit
tags: [svn, audit, externals]
source: external-knowledge-2026-09
---

## Context

Before migrating a checkout, list every SVN external and decide whether it
becomes a package or a vendored folder.

## Example

```bash
svn propget svn:externals -R .
```

## Notes

Pairs with the FiberCtrl coroutine migration notes; both were written for the
old repo layout.
