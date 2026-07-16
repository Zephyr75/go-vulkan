# govk Walkthrough — understand and improve the library

This is an onboarding guide for the `vk` package. It assumes you know Vulkan
concepts at a tutorial level but have never touched cgo or Go's `unsafe`. Read it
top to bottom once, then keep the "Recipe" and "Gotchas" sections nearby while
editing.

If you only remember one thing: this library's whole job is to translate between
two worlds — safe, garbage-collected **Go** and raw, manually-managed **C** — for
every Vulkan call. Almost every "weird" line in the code is doing that
translation. Section 0 explains the vocabulary; everything after is that idea
applied over and over.

---

## 0. Primer: cgo, pointers, and `unsafe` (read this first)

Skip this section only if you've written cgo before.

### 0.1 What cgo is
Go can call C functions directly through a feature called **cgo**. You write a
comment block with C `#include`s right above `import "C"`, and afterwards you can
say `C.vkCreateInstance(...)` as if it were a Go function. cgo compiles the C
header, figures out the C types, and generates the glue so Go and C can call each
other. Vulkan is a C API, so cgo is how we reach it at all.

```go
/*
#include <vulkan/vulkan.h>   // C declarations become available below
*/
import "C"                    // this exact line turns on cgo for the file

C.vkDeviceWaitIdle(dev)       // now you can call C functions via the "C" package
```

### 0.2 Why Go and C don't mix cleanly
Two differences make the border tricky:

1. **Memory management.** In Go, you allocate a value and forget it — the garbage
   collector (GC) frees it later, and it may even *move* the value to a different
   memory address while your program runs. In C, memory never moves and never
   frees itself; you call `malloc`/`free` by hand. So a pointer that Go hands to C
   is dangerous: the GC might move or reclaim the thing out from under C.
2. **Type safety.** Go won't normally let you treat a `Device` as a raw address,
   or reinterpret one struct as another. C does this constantly. Vulkan "handles"
   (like `VkDevice`) are really just opaque pointers/numbers.

To bridge these two, Go gives you one deliberately-dangerous tool: `unsafe`.

### 0.3 What `unsafe.Pointer` is and why we need it
`unsafe.Pointer` is a pointer with **no type** — a raw address the compiler will
let you convert to or from *any* other pointer type. It's the only legal way in
Go to say "take these bytes and treat them as that type instead."

We need it here for two unavoidable reasons:

- **Handle conversion.** Our Go handle types are `uintptr` (a plain integer that
  holds an address). The C side wants `C.VkDevice` (a C pointer type). Go won't
  convert an integer straight into a C pointer — but it *will* let both sides pass
  through `unsafe.Pointer`:
  ```go
  C.VkDevice(unsafe.Pointer(d))   // Go handle (uintptr) -> C handle
  Device(unsafe.Pointer(out))     // C handle -> Go handle (uintptr)
  ```
- **Reinterpreting memory.** Copying a Go slice into a mapped GPU pointer, or
  reading a C union as bytes, means treating one region of memory as a different
  type. Only `unsafe.Pointer` can express that (see `MemCopy` in `vk.go`, the
  `ClearValue` union in `cmd.go`).

Without `unsafe`, you simply cannot talk to a C API from Go. It is expected and
load-bearing here, not a smell.

### 0.4 Why `go vet` prints "possible misuse of unsafe.Pointer"
`go vet` is a linter. It flags `unsafe.Pointer` conversions that *often* indicate
a bug — specifically, going through the integer type `uintptr`. The danger it
worries about is real in general: if you convert a pointer to `uintptr`, do math
on it, then convert back, the GC may have moved the object in between, so your
address now points at garbage.

In this library the warning fires on nearly every handle conversion because our
handles are declared as `uintptr`. But our case is safe: a Vulkan handle is an
opaque token owned by the *driver*, not a Go object the GC manages or moves. We
never do pointer arithmetic on it; we just carry the number across the border.
So the warnings are **false positives inherent to the uintptr-handle design** —
not defects. (Section 7 lists an optional way to quiet them.)

> Rule of thumb for this codebase: `unsafe.Pointer` used to *convert a handle* or
> *reinterpret a fixed struct* = fine. The thing you actually must be careful
> about is **lifetime** — keeping Go memory alive and unmoved during a C call —
> which is a different rule, covered in Section 3.

---

## 1. Reading order

Don't read the files alphabetically. Follow the complexity gradient — each step
introduces one new marshaling idea:

1. **`vk/vk.go`** — the whole binding's foundation. Read every line.
2. **`vk/types.go`** — skim. It's just enums/flags/small structs. Come back when
   you need a specific constant.
3. **`vk/sync.go`** — the simplest real bindings. See the shape of a
   `Create*/Destroy*` pair with nothing tricky.
4. **`vk/resources.go`** — flat create-info structs, nested value structs
   (`Extent3D`, `subresourceRange`).
5. **`vk/memory.go`** — first conditional pNext chain (`AllocateMemory` +
   `VkMemoryAllocateFlagsInfo`) and the pure-Go `FindMemoryType`.
6. **`vk/device.go`** — the hardest single function, `CreateDevice`: a three-link
   feature pNext chain plus a nested array of queue-create-infos. This is the
   template for "allocate C structs, keep them alive, free after".
7. **`vk/command.go`** — arrays of structs built into C memory (`QueueSubmit2`).
8. **`vk/descriptor.go`** and **`vk/pipeline.go`** — the big marshaling jobs:
   arrays + pNext chains + `runtime.Pinner`. Read these last.
9. **`vk/cmd.go`** — recording commands; the `VkClearValue` union.
10. **`vk/swapchain.go`** — surface/present; note the acquire/present error
    handling that the render loop depends on.
11. **`cmd/demo/main.go`** — see how a real caller wires all of the above into a
    frame loop. Good for verifying your mental model.

---

## 2. The eight ideas that explain everything

Everything in the package is one of these patterns repeated. Learn them once.

### 2.1 The cgo preamble
Every file that touches Vulkan starts with:
```go
/*
#include <vulkan/vulkan.h>
*/
import "C"
```
This makes `C.VkFoo`, `C.vkBar(...)`, `C.VK_SOME_ENUM` available. `vk.go` adds
`#include <stdlib.h>`/`<string.h>` and the `-lvulkan` link flag. cgo generates a
Go view of every C type: `C.VkInstance` is a pointer, `C.uint32_t` is a number,
struct fields keep their C names (`sType`, `pNext`, `queueFamilyIndex`).

### 2.2 Handles are `uintptr`
See `vk.go`. Every Vulkan handle (`Instance`, `Device`, `Buffer`, …) is a
`uintptr` alias — a plain integer holding the driver's opaque token. On 64-bit
every handle is really a pointer-sized value, so `uintptr` holds them all, and —
crucially — external code (GLFW's surface creation) can construct one without
importing our package. At the C boundary you convert through `unsafe.Pointer`,
exactly as explained in Section 0.3:
```go
C.VkDevice(unsafe.Pointer(d))     // Go handle -> C handle
Device(unsafe.Pointer(out))       // C handle -> Go handle
```
This is the line that triggers the `go vet` "possible misuse" warning (Section
0.4). It's safe here and inherent to the design, not a bug.

### 2.3 `VkResult` → `error`
`check(r C.VkResult) error` returns `nil` on `VK_SUCCESS`, else a `Result`
(which implements `error`). The app compares against exported sentinels, e.g.
`if err == vk.ErrOutOfDateKHR`. Every fallible call ends with `return check(...)`
or `if err := check(...); err != nil`.

### 2.4 Two-call enumeration
Vulkan's "call once for the count, once for the data" pattern is captured by the
generic `enumerate` / `enumerateVoid` in `vk.go`. Any new `vkEnumerate*` or
`vkGet*` that returns a counted array should reuse them (see
`EnumeratePhysicalDevices`, `GetSwapchainImagesKHR`).

### 2.5 `sType` is never in the public API
Go-facing config structs (`BufferCreateInfo`, `GraphicsPipelineCreateInfo`, …)
omit `sType` and `pNext`. The marshaling layer fills `sType` when it builds the
`C.Vk*CreateInfo`. Callers never see it.

### 2.6 pNext chains
Optional feature/extension structs are linked through `pNext`. Two forms:
- **Static chain** (`CreateDevice`): allocate each struct, set its `sType`, point
  the previous struct's `pNext` at it.
- **Conditional chain** (`AllocateMemory`, `CreateDescriptorSetLayout`,
  `AllocateDescriptorSets`): only build+link the extra struct when a flag is set.

### 2.7 Arrays of structs → C memory, element by element
Never cast a Go slice of Go-defined structs to a C pointer and hope the layout
matches — 64-bit handle alignment inside structs differs subtly and breaks on
some drivers. Instead `calloc` a C array and fill it field by field. See
`QueueSubmit2`, `UpdateDescriptorSets`, `CreateGraphicsPipeline`,
`CmdPipelineBarrier2`. (Slices of plain handles like `[]DescriptorSet` are the
one exception — no inner pointers, so a direct cast is safe.)

### 2.8 Unions
Only `VkClearValue` appears here. It's modeled as `ClearValue [16]byte` with
`ClearColor(...)` / `ClearDepthStencil(...)` constructors and an internal `.c()`
that reinterprets the bytes as the C union. See `cmd.go`.

---

## 3. The pointer-lifetime rule (the one that bites)

This is the *other* pointer concern, separate from the `unsafe.Pointer` warning
in Section 0.4. That one was about type conversion and is harmless. This one is
about **lifetime**, and getting it wrong crashes the program.

Recall from Section 0.2 that Go's garbage collector can move or free your memory
at any time. If you hand C a pointer into Go memory and the GC moves it mid-call,
C is now reading garbage. To prevent that, cgo enforces a rule:

> You may pass a Go pointer to C as a direct argument, but the memory it points at
> must not itself *contain* another Go pointer — unless you "pin" that inner
> pointer first. ("Pinning" tells the GC: don't move or free this until I say.)

pNext chains and arrays of structs are exactly where a struct you pass to C ends
up holding another Go pointer inside it. Two strategies fix this, and you must
pick one for any new binding:

**A. Allocate the C-visible structs on the C heap (`C.calloc`), free after.**
Used by `CreateDevice`, the pipeline builder, the descriptor builders. Pattern:
```go
var frees []unsafe.Pointer
alloc := func(size uintptr) unsafe.Pointer {
    p := C.calloc(1, C.size_t(size)); frees = append(frees, p); return p
}
defer func() { for _, p := range frees { C.free(p) } }()
```
Because the structs live in C memory, no Go pointer is stored in anything handed
to C.

**B. Pin the Go memory with `runtime.Pinner`.**
Used when a Go slice/local is pointed at directly from a C struct — e.g.
`info.pSetLayouts = &goSlice[0]`. Pin it for the duration of the call:
```go
var pin runtime.Pinner
defer pin.Unpin()
pin.Pin(&slice[0])
```
`AllocateDescriptorSets`, `CreatePipelineLayout`, and `QueuePresentKHR` use this.

> Concrete failure mode: forgetting this yields a runtime panic
> `argument of cgo function has Go pointer to unpinned Go pointer` — not a
> compile error. It only fires at the exact call, so it can hide until that path
> runs. All three of the above were caught this way during bring-up.

Direct scalar/handle arguments (`vkWaitForFences(d, n, &fences[0], …)`) are fine
unpinned — cgo pins them automatically for the call. The rule is only about Go
pointers *stored inside* a struct you pass by pointer.

---

## 4. A couple of driver/toolchain quirks

- **sync2 `Flags64` constants are hardcoded** in `types.go`
  (`PipelineStageFlags2`, `AccessFlags2`). They're `static const VkFlags64` in the
  header, which cgo cannot resolve as `C.VK_...`; the numeric values are copied
  from `vulkan_core.h`. If you add a sync2 stage/access bit, copy its literal.
- **GLFW surface interop** (`cmd/demo/main.go`): `CreateWindowSurface` wants an
  instance value of *pointer kind* (so the demo wraps the handle as `*byte`), and
  it returns a `uintptr` that is the *address of* the surface handle, which the
  demo dereferences. This is a go-gl API shape, not a Vulkan one.
- **Memory selection** (`FindMemoryType`): the demo asks for
  `DEVICE_LOCAL|HOST_VISIBLE|HOST_COHERENT` (the ReBAR path — persistent-mapped,
  plain `memcpy`) and falls back to `HOST_VISIBLE|HOST_COHERENT`. Device-local
  images use `DEVICE_LOCAL` and are filled via a staging buffer + copy.

---

## 5. Recipe: add a new Vulkan function

1. **Classify difficulty.** Scalar/single-struct = easy. Arrays/strings = medium.
   pNext chain / nested struct arrays / union = hard.
2. **Add missing enums/flags** to `types.go` as `type Foo int32` (enums) or
   `type FooFlags uint32` (flag bits), each const `= Foo(C.VK_FOO)`. For
   `Flags64`, hardcode the literal.
3. **Define a Go-facing input struct** without `sType`/`pNext`. Use library value
   types (`Extent2D`, `ImageSubresourceRange`) so callers stay in Go.
4. **Marshal:** build the `C.Vk*` struct, set `sType`, convert each field. For
   arrays, `calloc` and fill element by element. For chains, allocate the extra
   struct and link `pNext`.
5. **Satisfy the pointer rule** (Section 3): strategy A or B. If in doubt, use A.
6. **Call and convert the result:** handles via `unsafe.Pointer`, errors via
   `check`, counted outputs via `enumerate`.
7. **Return `(T, error)`**, slices instead of count+pointer, never out-params.

Put the function in the file matching its milestone/topic (a barrier goes in
`cmd.go`, a new resource in `resources.go`, etc.).

---

## 6. How to verify a change

- **Validation layers are non-negotiable.** The demo and smoke test enable
  `VK_LAYER_KHRONOS_validation`. Marshaling/layout bugs surface as validation
  errors long before crashes.
- **Fast path:** `go test ./vk/ -run Instance -v` exercises instance + device +
  the feature pNext chain headlessly.
- **Full path:** `go run ./cmd/demo` under validation. Zero validation output =
  the frame graph (barriers, dynamic rendering, descriptors, BDA) is correct.
- After editing shaders: `go generate ./shaders` to recompile the SPIR-V.

---

## 7. Where to improve (open threads)

Roughly ordered by value:

- **Extract a `pnext.go` chain builder.** The plan called for a `Chainable`
  interface; the code instead inlines chains with `calloc`+`Pinner`. A small
  helper would de-duplicate `CreateDevice` / descriptor / pipeline code and make
  chains harder to get wrong.
- **Round out `Result`.** Only a handful of codes have `String()` cases; the rest
  fall through to `VkResult(n)`. Add the common negatives and a helper to tell
  soft codes (`TIMEOUT`, `SUBOPTIMAL`, `NOT_READY`) from hard failures.
- **Device-local vertex/index buffers.** The demo keeps geometry host-visible for
  simplicity. Add a staging-buffer upload path (mirrors `uploadTextures`) to move
  it to `DEVICE_LOCAL`-only memory.
- **Mipmaps / MSAA.** `CreateImage` hardcodes 1 sample; `CreateGraphicsPipeline`
  hardcodes `SAMPLE_COUNT_1_BIT` and no blend. Both are single-field extensions.
  Mip generation needs one `vkCmdBlitImage` binding (a plan stretch goal).
- **Legacy `vkQueueSubmit`.** Only `QueueSubmit2` exists. If you bind non-sync2
  code paths you'll need the v1 submit and `VkPipelineStageFlags` (v1) constants.
- **Multiple meshes / real assets.** `cmd/demo` draws one instanced mesh with
  solid-color textures. Loading several OBJs + PNGs into the bindless array is
  wiring, not new bindings.
- **Silence `unsafe.Pointer` vet noise.** Optional: wrap the handle→C conversions
  in tiny helper methods so `go vet` is quiet and misuse stands out.
