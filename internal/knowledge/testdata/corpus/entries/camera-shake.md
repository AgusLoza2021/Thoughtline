---
title: "Camera Shake as an Additive Local Position Offset"
tags: [Camera, camera, Camera Shake, Feel]
source: external-knowledge-2026-09
---

## Guidance

Add the shake as an offset on top of whatever else is moving the camera, never
as the camera's absolute local position.

## Applies when

A hit, impact or landing needs a short physical kick and the camera is already
driven by a follow or intro routine.

## Why this works

Additive offsets compose; absolute positions fight. The follower keeps its own
state and the shake is just noise layered on top.

## Procedure

1. Sample the shake curve into a Vector3 offset.
2. Apply it in LateUpdate, after the follower wrote its position.
3. Zero the offset when the curve ends.

## Validate the result

Trigger the shake while a pan is running: the pan must not stutter or rewind.

## Limits

Does not replace a real hit-stop; combine both only if the game reads clearly.

## Notes from the field

A one-off heading that is not part of either template still has to survive as
its own section.
