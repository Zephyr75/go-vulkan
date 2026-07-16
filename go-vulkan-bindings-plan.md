# Plan: Hand-Written cgo Vulkan Bindings for a "How to Vulkan 2026" Setup in Go

Goal: a Go library (`vk` package) exposing hand-written cgo declarations for every Vulkan entry point used by [howtovulkan.com](https://www.howtovulkan.com/) (Sascha Willems, Vulkan 1.3 baseline), plus a demo app that recreates the tutorial: multiple lit, textured 3D models rendered with dynamic rendering, buffer device address, descriptor indexing (bindless textures) and synchronization2.

---

## 1. Scope decisions (C++ tutorial -> Go project)

The tutorial leans on C/C++ libraries. Each needs a Go decision before writing any binding:

| Tutorial dependency | Role | Go decision |
|---|---|---|
| SDL3 | Window, surface, input | **GLFW via go-gl/glfw** (you already use it with go-gl). It provides `GetRequiredInstanceExtensions()`, `(*Window).CreateWindowSurface()`, `GetPhysicalDevicePresentationSupport()`, so zero platform-specific surface code to bind. |
| Volk | Function loader | **Skip.** Link the system loader directly (`-lvulkan`). Volk exists to avoid loader trampoline overhead and to load extensions; for ~65 functions, direct linking is fine. Can be revisited later with `vkGetDeviceProcAddr`. |
| VMA | Memory allocation | **Skip, allocate manually.** VMA is a C++ implementation; binding it means compiling C++ through cgo. Manual allocation adds ~8 functions to bind but teaches the actual Vulkan memory model, which is part of the point. Strategy mirrors the tutorial anyway: pick a `DEVICE_LOCAL | HOST_VISIBLE | HOST_COHERENT` memory type (ReBAR path), persistent map, plain `memcpy` via `unsafe.Slice`. |
| glm | Math | **Pure Go**: `go-gl/mathgl` (you already know it) or hand-rolled mat4/vec. Watch the depth range: use a Vulkan-style perspective (0..1 Z, flipped Y) instead of the GL one. |
| tinyobjloader | OBJ loading | **Pure Go**: OBJ is trivial to parse (~80 lines), or use an existing small Go OBJ lib. Keep the tutorial's y/v flips. |
| KTX-Software | Texture loading | **Skip KTX.** Load PNG via stdlib `image/png`, convert to RGBA, upload mip 0 only. Optional stretch goal: generate mips on the GPU with `vkCmdBlitImage` (one extra binding). |
| Slang runtime compilation | Shaders | **Compile offline.** Write the shader in Slang (or GLSL), compile with `slangc`/`glslc` from the LunarG SDK at build time (a `go:generate` line), embed the `.spv` with `go:embed`. Binding the Slang C++ API is a project in itself and adds nothing to the Vulkan learning goal. |

Result: the only C dependency in the whole project is `vulkan.h` + the loader. Everything else is Go.

---

## 2. Project layout

```
govk/
├── vk/                      # the binding library (importable on its own)
│   ├── vk.go                # cgo preamble, loader link flags, VkResult -> error
│   ├── instance.go          # instance, physical device, queue family queries
│   ├── device.go            # logical device, queues, features pNext chain
│   ├── memory.go            # memory requirements, allocate/bind/map, BDA
│   ├── resources.go         # buffers, images, image views, samplers
│   ├── swapchain.go         # surface caps, swapchain, present
│   ├── sync.go              # fences, semaphores, waits
│   ├── command.go           # pool, command buffers, begin/end/reset, submit
│   ├── cmd.go               # all vkCmd* recording functions
│   ├── descriptor.go        # set layouts, pool, sets, writes
│   ├── pipeline.go          # shader modules, pipeline layout, graphics pipeline
│   ├── types.go             # Go-facing structs, enums, flag constants
│   └── pnext.go             # helper for building pNext chains
├── internal/obj/            # OBJ parser
├── internal/texture/        # PNG -> RGBA loading
├── shaders/
│   ├── shader.slang
│   └── shader.spv           # generated, go:embed'ed
├── cmd/demo/
│   └── main.go              # the HowToVulkan recreation, top-to-bottom like the tutorial
└── go.mod
```

Design rule for the `vk` package, in line with what worked for go-vk: functions return `(T, error)` instead of out-pointers, slices instead of count+pointer, `error` wraps `VkResult` and is comparable against exported sentinel values (`vk.ErrOutOfDateKHR` is needed by the render loop's swapchain recreation logic). Keep `sType` out of the public structs: filled automatically in the marshaling layer.

---

## 3. Complete function inventory

Every Vulkan function the tutorial calls, plus the ones added by replacing VMA with manual allocation. Grouped by milestone. Difficulty: **S**imple (scalar args / single struct), **M**edium (arrays, strings, two-call enumeration), **C**omplex (pNext chains, nested struct arrays, unions).

### Milestone 1: Instance and device (9 functions)

| Function | Difficulty | Notes |
|---|---|---|
| `vkCreateInstance` | M | strings array marshaling (extensions from GLFW) |
| `vkDestroyInstance` | S | |
| `vkEnumeratePhysicalDevices` | M | first two-call enumeration pattern; write it once as a Go helper, reuse everywhere |
| `vkGetPhysicalDeviceProperties2` | M | returns a struct with a fixed-size char array (device name) to Goify |
| `vkGetPhysicalDeviceQueueFamilyProperties` | M | two-call enumeration |
| `vkGetPhysicalDeviceMemoryProperties2` | M | needed for manual allocation; fixed-size arrays of memory types/heaps |
| `vkCreateDevice` | **C** | the hardest single function: pNext chain of `VkPhysicalDeviceVulkan13Features` -> `VkPhysicalDeviceVulkan12Features`, plus queue create infos array, extension strings, 1.0 features struct. Budget real time here. |
| `vkDestroyDevice` | S | |
| `vkGetDeviceQueue` | S | |

Checkpoint: program prints selected GPU name and creates a 1.3 device with dynamicRendering, synchronization2, bufferDeviceAddress, descriptorIndexing features enabled. Validation layers clean.

### Milestone 2: Window, surface, swapchain (7 functions + GLFW glue)

| Function | Difficulty | Notes |
|---|---|---|
| `vkGetPhysicalDeviceSurfaceCapabilitiesKHR` | S | |
| `vkCreateSwapchainKHR` | M | large flat struct, no chain needed at first |
| `vkGetSwapchainImagesKHR` | M | two-call enumeration |
| `vkCreateImageView` | M | nested `subresourceRange` struct |
| `vkDestroyImageView` | S | |
| `vkDestroySwapchainKHR` | S | |
| `vkDestroySurfaceKHR` | S | |

GLFW glue: `glfw.GetRequiredInstanceExtensions()` feeds Milestone 1, `window.CreateWindowSurface(instance, nil)` returns a `uintptr` to cast to the surface handle. One subtlety: GLFW hands back the raw handle, so the `vk` package needs its handle types to be plain `uintptr`/pointer aliases that external code can construct.

Checkpoint: swapchain created at window size, image views for every swapchain image, clean recreation path stubbed.

### Milestone 3: Memory, buffers, images (13 functions)

| Function | Difficulty | Notes |
|---|---|---|
| `vkCreateBuffer` | M | |
| `vkCreateImage` | M | nested extent struct |
| `vkGetBufferMemoryRequirements` | S | |
| `vkGetImageMemoryRequirements` | S | |
| `vkAllocateMemory` | M | pNext with `VkMemoryAllocateFlagsInfo` for buffer device address allocations |
| `vkBindBufferMemory` | S | |
| `vkBindImageMemory` | S | |
| `vkMapMemory` | S | return `unsafe.Pointer`; write a `vk.MemCopy[T any](dst unsafe.Pointer, src []T)` generic helper on top |
| `vkUnmapMemory` | S | |
| `vkFreeMemory` | S | |
| `vkGetBufferDeviceAddress` | S | core to the tutorial's descriptor-free buffer access |
| `vkDestroyBuffer` | S | |
| `vkDestroyImage` | S | |

Plus one pure-Go piece: memory type selection (iterate `memoryTypes`, match `memoryTypeBits` and desired property flags, with fallback from `DEVICE_LOCAL|HOST_VISIBLE` to `HOST_VISIBLE|HOST_COHERENT` for machines without ReBAR).

Checkpoint: Suzanne OBJ parsed, interleaved vertex+index data memcpy'd into a mapped device buffer; depth image created and bound; shader data buffers (one per frame in flight) created with device addresses retrieved.

### Milestone 4: Sync and command buffers (12 functions)

| Function | Difficulty | Notes |
|---|---|---|
| `vkCreateFence` / `vkDestroyFence` | S | signaled-bit flag |
| `vkCreateSemaphore` / `vkDestroySemaphore` | S | |
| `vkWaitForFences` / `vkResetFences` | S | |
| `vkCreateCommandPool` / `vkDestroyCommandPool` | S | |
| `vkAllocateCommandBuffers` | M | returns a slice of handles |
| `vkResetCommandBuffer` | S | |
| `vkBeginCommandBuffer` / `vkEndCommandBuffer` | S | |

Checkpoint: frames-in-flight scaffolding compiles; per-frame fences, per-frame acquire semaphores, per-swapchain-image render-complete semaphores (the tutorial's semaphore-count subtlety).

### Milestone 5: Texture upload and descriptors (10 functions)

| Function | Difficulty | Notes |
|---|---|---|
| `vkCmdPipelineBarrier2` | **C** | sync2 image barriers; slice of `VkImageMemoryBarrier2` inside `VkDependencyInfo`. Once written, reused for every layout transition in the project. |
| `vkCmdCopyBufferToImage` | M | array of `VkBufferImageCopy` regions |
| `vkQueueSubmit` | **C** | wait/signal semaphore arrays plus wait-stage masks; used both for the one-time upload and the render loop |
| `vkCreateSampler` | M | flat struct, many fields |
| `vkCreateDescriptorSetLayout` | **C** | pNext with `VkDescriptorSetLayoutBindingFlagsCreateInfo` for variable descriptor count (descriptor indexing) |
| `vkCreateDescriptorPool` | M | pool sizes array |
| `vkAllocateDescriptorSets` | **C** | pNext with `VkDescriptorSetVariableDescriptorCountAllocateInfo` |
| `vkUpdateDescriptorSets` | **C** | array of writes each pointing at an array of `VkDescriptorImageInfo` |
| `vkDestroyDescriptorPool` / `vkDestroyDescriptorSetLayout` | S | |
| `vkDestroySampler` | S | |

Checkpoint: N textures uploaded via staging buffer + barriers + copy, transitioned to `READ_ONLY_OPTIMAL`, all bound into one variable-count combined-image-sampler array (the bindless setup).

### Milestone 6: Pipeline (7 functions)

| Function | Difficulty | Notes |
|---|---|---|
| `vkCreateShaderModule` | M | `[]byte` SPIR-V -> `uint32` pointer + size |
| `vkCreatePipelineLayout` | M | set layouts + push constant range (holds the buffer device address) |
| `vkCreateGraphicsPipelines` | **C** | the biggest marshaling job in the project: ~10 sub-structs (vertex input, input assembly, viewport, rasterization, multisample, depth stencil, color blend, dynamic state, shader stages) plus `VkPipelineRenderingCreateInfo` in pNext for dynamic rendering. Plan half a day for this one function. |
| `vkDestroyShaderModule` | S | |
| `vkDestroyPipeline` / `vkDestroyPipelineLayout` | S | |
| `vkDeviceWaitIdle` | S | |

Checkpoint: pipeline builds without validation errors against the embedded SPIR-V.

### Milestone 7: Render loop (10 functions)

| Function | Difficulty | Notes |
|---|---|---|
| `vkAcquireNextImageKHR` | S | must surface `VK_ERROR_OUT_OF_DATE_KHR` / `VK_SUBOPTIMAL_KHR` as distinguishable errors, not panics |
| `vkCmdBeginRendering` / `vkCmdEndRendering` | **C** | `VkRenderingInfo` with attachment info structs containing the `VkClearValue` union (color vs depthStencil): first real union to handle; a `[16]byte` field with setter methods works |
| `vkCmdSetViewport` / `vkCmdSetScissor` | S | |
| `vkCmdBindPipeline` | S | |
| `vkCmdBindDescriptorSets` | M | |
| `vkCmdBindVertexBuffers` / `vkCmdBindIndexBuffer` | S | |
| `vkCmdPushConstants` | S | pushes the 8-byte device address |
| `vkCmdDrawIndexed` | S | |
| `vkQueuePresentKHR` | M | same out-of-date handling as acquire |

Checkpoint: three Suzannes on screen, Phong-lit, textured via `NonUniformResourceIndex`, mouse rotation, wheel zoom, resize-driven swapchain recreation, clean shutdown with all destroys.

**Total: ~65 functions.** Roughly 35 S, 20 M, 10 C.

---

## 4. Cross-cutting binding work (do once, early)

These are not functions but shared machinery; most of the real effort lives here:

1. **`VkResult` -> Go error.** A `Result int32` type implementing `error`, exported sentinels for the codes the app must branch on (`SUCCESS`, `ERROR_OUT_OF_DATE_KHR`, `SUBOPTIMAL_KHR`, `TIMEOUT`, `ERROR_DEVICE_LOST`), `String()` for the rest.
2. **Two-call enumeration helper.** Generic `enumerate[T any](f func(*uint32, *T) C.VkResult) ([]T, error)` kills the most repetitive pattern in the API.
3. **String array marshaling.** `[]string` -> `**C.char` with cleanup, used by instance/device creation. One helper, written once.
4. **pNext chain builder.** The tutorial chains features structs at device creation and extension structs at descriptor layout/alloc and pipeline creation. A tiny `Chainable` interface (`vulkanize() unsafe.Pointer`) plus a `Chain(structs ...Chainable)` helper keeps the ugliness in one file. Keep C-side structs alive across the call with `runtime.KeepAlive` or by allocating them with `C.malloc` and freeing after: cgo pointer rules forbid storing Go pointers inside C-visible structs that outlive the call, and pNext chains are exactly where this bites.
5. **Unions.** Only `VkClearValue`/`VkClearColorValue` in this scope. Represent as raw bytes + `SetColor(r,g,b,a float32)` / `SetDepthStencil(d float32, s uint32)`.
6. **Struct layout discipline.** For structs passed in arrays (`VkImageMemoryBarrier2`, `VkBufferImageCopy`, `VkVertexInputAttributeDescription`, `VkDescriptorImageInfo`, `VkPipelineShaderStageCreateInfo`), the simplest correct approach is to use the `C.` types directly internally and convert from Go-facing structs element by element into a C-allocated array. Never pass a pointer to a Go slice of Go-defined structs and hope the layout matches: alignment of 64-bit handles inside structs differs subtly and this is the classic source of "works then crashes on another driver".
7. **CPU-side ShaderData layout.** `mat4 projection, view; mat4 model[3]; vec4 lightPos; uint32 selected` must match the Slang struct byte for byte. In Go: fixed-size float32 arrays in a struct with explicit padding; add a `unsafe.Sizeof` assertion in an init-time check against the expected size.
8. **Validation layers.** Enable `VK_LAYER_KHRONOS_validation` in debug builds via instance create info (or via vkconfig). Non-negotiable while developing the bindings themselves: layout bugs in marshaling show up as validation errors long before they show up as crashes.

---

## 5. Time estimate

Assumes evening/weekend pace, cgo basics already familiar, validation layers on from day one.

| Phase | Content | Estimate |
|---|---|---|
| 0 | Project setup, cgo preamble, Result/error, enumeration + string helpers | 0.5 day |
| 1 | Instance + device (incl. the features pNext chain and the pNext builder) | 1 to 1.5 days |
| 2 | Surface (GLFW glue) + swapchain + image views | 0.5 day |
| 3 | Memory, buffers, images, OBJ parsing, mapped copies, BDA | 1 to 1.5 days |
| 4 | Sync objects + command pool/buffers + frames-in-flight scaffolding | 0.5 day |
| 5 | Barriers (sync2), texture upload, sampler, descriptor indexing setup | 1.5 days |
| 6 | Shader offline compile pipeline + shader module + pipeline layout + graphics pipeline | 1 to 1.5 days |
| 7 | Render loop, present, resize/out-of-date handling, input, cleanup | 1 day |
| 8 | Buffer: debugging marshaling bugs found by validation layers, driver quirks | 1 to 2 days |

**Total: 8 to 11 focused days**, realistically 3 to 5 weeks of evenings. Milestones 1, 5 and 6 carry the complexity; everything else is volume.

Two scope levers if it drags:
- Cut descriptor indexing first (one texture, fixed-count descriptor set): removes both pNext chains from Milestone 5 and simplifies the shader. Saves ~1 day. Add it back after first triangle-equivalent.
- Cut textures entirely for a first pass (vertex-color Phong): removes Milestone 5 almost completely, first lit model on screen after ~5 days, then layer texturing back in.

---

## 6. Definition of done

- `go build ./...` with only `vulkan.h` + loader as native deps, GLFW via go-gl/glfw.
- Demo renders 3 lit, textured model instances with dynamic rendering, sync2 barriers, push-constant buffer device address, bindless texture array indexed by instance.
- Zero validation errors including the synchronization validation preset.
- Window resize and minimize handled through swapchain recreation.
- `vk` package importable standalone: this becomes the Vulkan backend of the engine, sitting next to the existing go-gl backend behind the device interface.
