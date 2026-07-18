# go-vulkan

Hand-written cgo bindings for the subset of Vulkan 1.3 taught by
[howtovulkan.com](https://howtovulkan.com) — dynamic rendering, buffer device
address, descriptor indexing, synchronization2 — plus a pure-Go substitute for
the VMA allocator, and a demo that mirrors the tutorial's single-file C++
renderer section by section.

## Layout

```
vk/              the library: cgo bindings + the VMA substitute (vma.go)
how_to_vulkan/   the demo: Go port of the reference renderer, kept side by
                 side with the original (_reference.cpp) for comparison
demo/            demo-only support packages: OBJ loader, embedded SPIR-V
                 shaders, texture generation
```

The `vk` package is self-contained: it needs only `vulkan.h`, the Vulkan
loader (`-lvulkan`), and the Go standard library. GLFW and mathgl are demo
dependencies, not library ones.

## Quick start

```sh
go run ./how_to_vulkan   # three textured meshes; drag rotates, +/- selects, wheel zooms
go test ./vk/            # headless binding tests (skipped without a Vulkan device)
```

## Structure of the library

| File | Provides |
|---|---|
| `vk/vk.go` | `Result` error type, handle types, two-call enumeration helper, `MemCopy` |
| `vk/types.go` | Enums, flag bits, small value structs (`Extent2D`, `Rect2D`, …), constants |
| `vk/instance.go` | Instance creation, physical-device / queue-family / memory queries |
| `vk/device.go` | Logical device with the 1.0/1.2/1.3 feature chain |
| `vk/swapchain.go` | Surface queries, swapchain, image views, acquire/present |
| `vk/memory.go` | Raw allocation, binding, mapping, buffer device address, memory-type selection |
| `vk/vma.go` | Pure-Go VMA substitute: `VmaCreateAllocator` / `VmaCreateBuffer` / `VmaCreateImage` with the same API shape as VMA, one dedicated allocation per resource |
| `vk/resources.go` | Buffers, images, samplers |
| `vk/sync.go` | Fences and semaphores |
| `vk/command.go` | Command pool/buffers, synchronization2 submit |
| `vk/cmd.go` | Recording: image barriers, dynamic rendering, buffer→image copies, binds, draws |
| `vk/descriptor.go` | Descriptor set layout/pool/sets with descriptor indexing (variable-count bindless arrays) |
| `vk/pipeline.go` | Shader modules, pipeline layout, graphics pipeline |

## Coverage

- Every raw `vk*` entry point the tutorial teaches is wrapped — the full
  instance → device → swapchain → mesh → texture → draw → present → teardown
  path, ~78 functions in total.
- The tutorial delegates all memory work to VMA; this repo instead wraps the
  raw memory API (`vkAllocateMemory`, `vkBind*Memory`, `vkMapMemory`, …) and
  layers `vk/vma.go` on top so the demo keeps the reference's
  `vmaCreateBuffer`/`vmaCreateImage` call shape without a C VMA dependency.
- Submission uses `vkQueueSubmit2` (synchronization2) rather than the legacy
  `vkQueueSubmit`; barriers use `vkCmdPipelineBarrier2`.
- Every create has its matching destroy, so the library is usable outside the
  linear tutorial flow.

## Design

- **Handles are `uintptr`.** All Vulkan handles are pointer-sized on 64-bit,
  so one opaque, copyable representation covers them; conversion to cgo types
  happens at each call boundary. This is also why `go vet` reports "possible
  misuse of unsafe.Pointer" across the package — expected and benign.
- **`VkResult` becomes `error`.** `nil` on success, otherwise a comparable
  sentinel (`vk.ErrOutOfDateKHR`, `vk.SuboptimalKHR`, …) the caller branches
  on — the swapchain-recreation path depends on this.
- **Config structs omit `sType`/`pNext`.** The marshaling layer fills them;
  callers write plain Go literals.
- **Two-call enumeration is one generic helper.** Every counted query
  (`vkEnumerate*`, `vkGet*`) goes through `enumerate`/`enumerateVoid`.
- **Nested C structs live in C heap.** cgo forbids passing C a Go pointer that
  contains Go pointers, so pNext chains and struct arrays are `calloc`ed into
  an arena (freed after the call) or pinned for the call's duration.

Per-function internals, marshaling details, and links into the Vulkan/VMA
documentation: [ADVANCED.md](ADVANCED.md).
