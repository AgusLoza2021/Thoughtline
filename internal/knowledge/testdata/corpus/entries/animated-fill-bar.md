---
title: "Animated fill bar without fillAmount"
tags: [UI, fill, Bar]
source: external-knowledge-2026-09
---

## Context

A radial or linear fill that has to animate smoothly cannot rely on
`fillAmount` alone, because it snaps at the ends.

## Example

Drive the RectTransform width instead, clamped to the track width.

## Notes

Saved as UTF-8 with a BOM and CRLF endings on purpose: the parser has to
normalize both.
