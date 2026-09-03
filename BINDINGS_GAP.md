# Binding inventory and gap list

Every exported function in `vk`, whether `how_to_vulkan/_reference.cpp` uses it,
and what still has to be added for compute, storage images, HDR, volumetrics,
reflection probes and ray tracing.

Audited 2026-08-05 against `vk/*.go` (3080 lines) and `_reference.cpp` (712
lines). The reference is this repository's copy of the howtovulkan.com program,
and is what the **Ref** column below means. `✓` = the C reference calls it. `✗` =
it does not.

Scope: the `vk` package's surface. Not here: why individual functions are shaped
the way they are (`ADVANCED.md`), or what the engine does with them
(`overdrive/notes/tmp/BACKEND_DECISION.md`).

---

## 0. Table of contents

1. [Verdict](#1-verdict)
2. [Existing — used by the reference](#2-existing--used-by-the-reference)
3. [Existing — not in the reference](#3-existing--not-in-the-reference)
4. [Already possible with no new bindings](#4-already-possible-with-no-new-bindings)
5. [New functions](#5-new-functions)
6. [New enums and struct fields](#6-new-enums-and-struct-fields)
7. [Suggested order](#7-suggested-order)

---

## 1. Verdict

**The function-level gap is small: 3 new functions and 1 signature change unlock
compute, storage images, volumetrics and HDR.** The bulk of the remaining work is
enum constants and struct fields, which are cheap and mechanical.

Everything past the current surface is **outside the reference's scope**. The
reference is a single-pass forward renderer that draws one textured indexed mesh.
It has no compute, no offscreen targets, no mipmaps, no second pass. So the
**Ref** column is ✓ for most of what exists and ✗ for essentially everything
new — the tutorial stops being a guide precisely where this list starts. That is
a fact about the tutorial's scope, not a reason to hesitate.

Three findings that shrink the work below earlier estimates:

- **Blend state is already complete.** `PipelineColorBlendAttachmentState` has
  `BlendEnable` and all six factor/op fields. Glass and water need enum values,
  not code.
- **Multiple color attachments already work.** `RenderingInfo.ColorAttachments`,
  `PipelineRenderingCreateInfo.ColorAttachmentFormats` and
  `PipelineColorBlendStateCreateInfo.Attachments` are all slices. A G-buffer
  needs no bindings work.
- **Storage buffers need nothing** if they are reached by device address, which
  is how the engine already passes uniforms. Descriptor-bound storage buffers
  are a separate, optional path (§5).

One real structural gap: `CmdPipelineBarrier2` takes only image barriers, so
there is no way to express a compute→graphics hazard on a *buffer*. That is the
one signature that must change.

## 2. Existing — used by the reference

62 functions. All ✓ in the reference; listed for completeness, grouped as the
reference orders them.

| group | functions |
|---|---|
| instance & device | `CreateInstance` `DestroyInstance` `EnumeratePhysicalDevices` `GetPhysicalDeviceProperties2` `GetPhysicalDeviceQueueFamilyProperties` `GetPhysicalDeviceFormatProperties2` `CreateDevice` `DestroyDevice` `GetDeviceQueue` `DeviceWaitIdle` |
| surface & swapchain | `CreateSwapchainKHR` `DestroySwapchainKHR` `GetSwapchainImagesKHR` `GetPhysicalDeviceSurfaceCapabilitiesKHR` `DestroySurfaceKHR` `AcquireNextImageKHR` `QueuePresentKHR` |
| images & views | `CreateImage` `DestroyImage` `CreateImageView` `DestroyImageView` `CreateSampler` `DestroySampler` |
| buffers | `GetBufferDeviceAddress` |
| descriptors | `CreateDescriptorSetLayout` `DestroyDescriptorSetLayout` `CreateDescriptorPool` `DestroyDescriptorPool` `AllocateDescriptorSets` `UpdateDescriptorSets` |
| pipeline | `CreateShaderModule` `DestroyShaderModule` `CreatePipelineLayout` `DestroyPipelineLayout` `CreateGraphicsPipeline` `DestroyPipeline` |
| commands | `CreateCommandPool` `DestroyCommandPool` `AllocateCommandBuffers` `BeginCommandBuffer` `EndCommandBuffer` `ResetCommandBuffer` |
| recording | `CmdBeginRendering` `CmdEndRendering` `CmdBindPipeline` `CmdBindVertexBuffer` `CmdBindIndexBuffer` `CmdBindDescriptorSets` `CmdPushConstants` `CmdDrawIndexed` `CmdSetViewport` `CmdSetScissor` `CmdCopyBufferToImage` `CmdPipelineBarrier2` |
| sync | `CreateFence` `DestroyFence` `WaitForFences` `ResetFences` `CreateSemaphore` `DestroySemaphore` |
| VMA | `VmaCreateAllocator` `VmaDestroyAllocator` `VmaCreateBuffer` `VmaDestroyBuffer` `VmaCreateImage` `VmaDestroyImage` |

Two intentional modernisations, both worth keeping:

| binding | reference | why the binding differs |
|---|---|---|
| `QueueSubmit2` | `vkQueueSubmit` (1.0) | Synchronization2 throughout; the reference mixes the old submit with `vkCmdPipelineBarrier2` |
| `CreateGraphicsPipeline` (singular), `CmdBindVertexBuffer` (singular) | plural | Batch creation and multi-buffer binding are unused; the singular form drops a slice parameter |

## 3. Existing — not in the reference

20 functions the binding adds. Every one has a reason; none is speculative.

| function | Ref | why it exists |
|---|---|---|
| `AllocateMemory` `FreeMemory` `MapMemory` `UnmapMemory` `BindBufferMemory` `BindImageMemory` `GetBufferMemoryRequirements` `GetImageMemoryRequirements` `CreateBuffer` `DestroyBuffer` `FindMemoryType` `GetPhysicalDeviceMemoryProperties2` | ✗ | **The VMA reimplementation.** The reference links the real C VMA and never touches raw memory. `vk/vma.go` is pure Go and composes these 12. Non-negotiable — they are `vma.go`'s substrate |
| `GetPhysicalDeviceSurfaceFormatsKHR` `GetPhysicalDeviceSurfacePresentModesKHR` `GetPhysicalDeviceSurfaceSupportKHR` | ✗ | The reference assumes a format and a present mode. Querying is correct, cheap and already written |
| `CmdSetCullMode` `CmdSetDepthCompareOp` | ✗ | Dynamic state, backing Overdrive's `SetCullMode` / `SetDepthCompare`. **These become optional** if the engine moves to baked pipeline objects — but keep them; per-draw cull flips are still the cheap way to render a two-sided material. `CmdSetFrontFace` was bound alongside them and removed on 2026-08-05: front face is a pass's winding convention, so it belongs in the pipeline, and nothing ever set it dynamically |
| `CmdDraw` | ✗ | Non-indexed draw. The reference only draws indexed; the engine's fullscreen quads and skybox are non-indexed |
| `CmdCopyImage` | ✗ | Added 2026-08-13 for Overdrive's shadow atlas: a depth tile is copied from the static atlas into the dynamic one, and only the movable casters are redrawn on top. Sits beside `CmdBlitImage` (§5.2), which is still missing — a copy needs no format conversion or filtering, so it is the smaller of the two |
| `CmdCopyImageToBuffer` | ✗ | Added 2026-08-16, the mirror of `CmdCopyBufferToImage` and sharing its `BufferImageCopy`. An image has no host-visible form, so a readback is always a copy into a mapped buffer; Overdrive dumps its shadow atlas to a PNG through this, the only way to eyeball a depth target on a session with no screenshot path |
| `QueueWaitIdle` | ✗ | One-time-submit teardown without a fence |
| `MemCopy` `ClearColor` `ClearDepthStencil` | ✗ | Go-side helpers, no C counterpart |

Two things the reference does that the binding deliberately does **not** expose:
`vkGetInstanceProcAddr` / `vkGetDeviceProcAddr` (extension loading — cgo links
`libvulkan` directly instead), and `vkCreateGraphicsPipelines` batching.

## 4. Already possible with no new bindings

Check these off before writing anything. Each is a feature the current surface
already expresses:

| feature | how |
|---|---|
| **Transparency, glass, water blending** | `PipelineColorBlendAttachmentState.BlendEnable` + factors. Needs only more `BlendFactor` values (§6) |
| **G-buffer / multiple render targets** | `RenderingInfo.ColorAttachments` and `PipelineRenderingCreateInfo.ColorAttachmentFormats` are slices |
| **Cube render targets (reflection probes)** | `ImageCreateCubeCompatible` + `ImageViewTypeCube` + `RenderingInfo.LayerCount` + `ShaderStageGeometry` — the engine's point-shadow cube already does exactly this |
| **Storage buffers via device address** | `BufferUsageStorageBuffer` + `BufferUsageShaderDeviceAddress` + `GetBufferDeviceAddress` + `CmdPushConstants`. Same path the uniform ring uses. **No descriptors involved** |
| **MSAA resolve** | `RenderingAttachmentInfo.ResolveMode` / `ResolveImageView` |
| **Bindless texture arrays** | `DescriptorSetLayoutCreateInfo.UseBindingFlags` + variable descriptor count, already wired |
| **Async compute on a second queue** | `GetDeviceQueue` with a second family index; `SubmitInfo2`; queue-family transfer fields already on `ImageMemoryBarrier2`. Plumbing only |
| **3D image *creation*** | `ImageType3D` exists on `ImageCreateInfo`. Only the *view* type is missing (§6) |

## 5. New functions

Sorted by what they unlock. **Ref is ✗ for every row** — the reference program
does none of this.

### 5.1 Required for compute — 2 new, 1 changed

| function | Ref | note |
|---|---|---|
| `CreateComputePipeline(d, ComputePipelineCreateInfo) (Pipeline, error)` | ✗ | One stage plus a layout. Much smaller than the graphics version — no vertex input, no blend, no rendering-info `pNext`. `DestroyPipeline` already covers teardown |
| `CmdDispatch(cb, groupsX, groupsY, groupsZ uint32)` | ✗ | Three-line wrapper |
| `CmdPipelineBarrier2` — **signature change** | ✗ | Today: `(cb, []ImageMemoryBarrier2)`. Needed: `(cb, DependencyInfo)` where `DependencyInfo` carries image, buffer and global barriers. **Without this a compute pass that writes a storage buffer cannot be synchronised with the graphics pass that reads it.** The one genuine structural gap |

`CmdBindPipeline` and `CmdBindDescriptorSets` already take a `PipelineBindPoint`,
so they need no change — only the `PipelineBindPointCompute` constant (§6).

### 5.2 Mipmaps — 1 new

| function | Ref | note |
|---|---|---|
| `CmdBlitImage` | ✗ | Generates a mip chain by successive half-size blits. Wanted for **PBR texture quality**, **IBL and probe prefiltering** (roughness levels are mips of a cube), and KTX-less texture loading. The reference sidesteps it by loading KTX files with mips baked in |

Worth pairing with `FormatFeatureSampledImageFilterLinear` to probe support (§6).

### 5.3 Indirect / GPU-driven — 3 new, all optional

| function | Ref | want it? |
|---|---|---|
| `CmdDispatchIndirect` | ✗ | **Yes, eventually.** One compute pass sizes the next — how variable-length GPU work (ray queues, cluster lists, surfel allocation) is built. Cheap once `CmdDispatch` exists |
| `CmdDrawIndexedIndirect` | ✗ | Only when GPU culling happens. Skip for now |
| `CmdDrawIndirect` | ✗ | Same. Skip |

Also needs `BufferUsageIndirectBuffer` (§6).

### 5.4 GPU timing — 5 new, optional but high value

| function | Ref | note |
|---|---|---|
| `CreateQueryPool` `DestroyQueryPool` `CmdResetQueryPool` `CmdWriteTimestamp2` `GetQueryPoolResults` | ✗ | Per-pass GPU milliseconds. Currently the engine reports only CPU frame count, which cannot tell a slow shadow pass from a slow forward pass. **`FEATURES.md` already notes FPS subtraction is not a valid measurement** — this is the thing that would replace it |

### 5.5 Debug labelling — 2 new, optional

| function | Ref | note |
|---|---|---|
| `SetDebugUtilsObjectNameEXT` | ✗ | Names appear in validation messages and RenderDoc instead of raw handles. Genuinely large quality-of-life win once there are dozens of images |
| `CmdBeginDebugUtilsLabelEXT` / `CmdEndDebugUtilsLabelEXT` | ✗ | Groups a capture by pass. Pairs with the pass list |

### 5.6 Descriptor-bound buffers — 0 new functions, 1 struct field

`WriteDescriptorSet` has only `ImageInfo`; its comment says buffer descriptors
are not modelled because the reference passes buffers by device address. Adding
`BufferInfo []DescriptorBufferInfo` would enable classic UBO/SSBO binding.

**Probably not wanted.** Device address already works, costs no descriptor pool
management, and is what the engine does. Add only if a shader needs a buffer the
compiler will not accept as a pointer.

### 5.7 Ray tracing — 5 new for ray queries, 3 more for pipelines

| function | Ref | note |
|---|---|---|
| `GetAccelerationStructureBuildSizesKHR` `CreateAccelerationStructureKHR` `DestroyAccelerationStructureKHR` `CmdBuildAccelerationStructuresKHR` `GetAccelerationStructureDeviceAddressKHR` | ✗ | The BLAS/TLAS lifecycle. **This is all ray queries need** — `rayQueryEXT` is inline in a compute shader, so no new pipeline type and no shader binding table |
| `CreateRayTracingPipelinesKHR` `GetRayTracingShaderGroupHandlesKHR` `CmdTraceRaysKHR` | ✗ | Only for the full stage machinery (raygen/anyhit/closesthit/callable). Skip unless intersection or callable shaders are actually wanted |

Plus device-feature and extension plumbing: `VK_KHR_acceleration_structure`,
`VK_KHR_ray_query`, `VK_KHR_deferred_host_operations`, and the
`PhysicalDeviceAccelerationStructureFeaturesKHR` / `RayQueryFeaturesKHR` chain.

Realistically 600–1000 lines including the structure-geometry descriptors, which
are the fiddliest part of the Vulkan API. Additive — nothing existing changes.

## 6. New enums and struct fields

The actual bulk of the work, and all of it is one-line additions to `types.go`.

### Compute

```go
PipelineBindPointCompute = PipelineBindPoint(C.VK_PIPELINE_BIND_POINT_COMPUTE)
ShaderStageCompute       = ShaderStageFlags(C.VK_SHADER_STAGE_COMPUTE_BIT)
ShaderStageAll           = ShaderStageFlags(C.VK_SHADER_STAGE_ALL)

PipelineStage2ComputeShader  = ...VK_PIPELINE_STAGE_2_COMPUTE_SHADER_BIT
PipelineStage2VertexShader   = ...VK_PIPELINE_STAGE_2_VERTEX_SHADER_BIT
PipelineStage2DrawIndirect   = ...VK_PIPELINE_STAGE_2_DRAW_INDIRECT_BIT
Access2ShaderWrite           = ...VK_ACCESS_2_SHADER_WRITE_BIT
Access2ShaderStorageRead     = ...VK_ACCESS_2_SHADER_STORAGE_READ_BIT
Access2ShaderStorageWrite    = ...VK_ACCESS_2_SHADER_STORAGE_WRITE_BIT
Access2IndirectCommandRead   = ...VK_ACCESS_2_INDIRECT_COMMAND_READ_BIT
```

### Storage images and 3D

```go
DescriptorTypeStorageImage = ...VK_DESCRIPTOR_TYPE_STORAGE_IMAGE
ImageUsageStorage          = ...VK_IMAGE_USAGE_STORAGE_BIT
ImageViewType3D            = ...VK_IMAGE_VIEW_TYPE_3D
ImageViewType1D            = ...VK_IMAGE_VIEW_TYPE_1D
ImageViewTypeCubeArray     = ...VK_IMAGE_VIEW_TYPE_CUBE_ARRAY   // probe arrays
BufferUsageIndirectBuffer  = ...VK_BUFFER_USAGE_INDIRECT_BUFFER_BIT
```

`ImageLayoutGeneral` already exists, which is the layout a storage image is
written in. `WriteDescriptorSet.ImageInfo` already works for storage images —
leave `Sampler` zero.

### Formats

```go
FormatR16G16B16A16Sfloat     // HDR colour, the format FEATURES.md records as blocking HDR
FormatR16Sfloat              // single-channel HDR: volumetric density, AO
FormatR16G16Sfloat           // velocity buffers, BRDF LUT
FormatR11G11B10UfloatPack32  // cheaper HDR, no alpha
FormatR32Uint                // atomics, GPU-built lists
FormatR8Unorm                // masks, single-channel data
FormatBC7UnormBlock / BC7SrgbBlock / BC5UnormBlock / BC6HUfloatBlock  // compressed
```

`BC6H` is the one to pair with HDR cubemaps; `BC5` is the normal-map format.

### Format capability probing

```go
FormatFeatureStorageImage
FormatFeatureColorAttachment
FormatFeatureColorAttachmentBlend
FormatFeatureSampledImageFilterLinear
```

`GetPhysicalDeviceFormatProperties2` already exists — these are the bits to test
against, the same way the depth-format probe works today.

### Blend, for glass and water

```go
BlendFactorSrcColor / OneMinusSrcColor / DstColor / OneMinusDstColor
BlendFactorDstAlpha / OneMinusDstAlpha
BlendFactorConstantColor / ConstantAlpha
BlendOpSubtract / ReverseSubtract / Min / Max
```

Alpha blending needs only `SrcAlpha` / `OneMinusSrcAlpha`, which **already
exist** — additive glow needs `One`/`One`, also present. So the minimum for
transparent glass and water is zero additions; the list above is for the rest.

### Misc

```go
CompareOpGreater / GreaterOrEqual / Equal / Never   // reverse-Z, stencil-like tricks
SamplerAddressModeMirroredRepeat
SamplerMipmapModeLinear                              // check; needed for trilinear
DynamicStateDepthWriteEnable / DepthTestEnable / DepthBias
PolygonModeLine                                      // wireframe debug
```

### Struct fields

| struct | add | for |
|---|---|---|
| `DependencyInfo` (new) | `ImageBarriers`, `BufferBarriers`, `MemoryBarriers` | §5.1 |
| `BufferMemoryBarrier2` (new) | stage/access masks, buffer, offset, size | §5.1 |
| `MemoryBarrier2` (new) | stage/access masks only | §5.1, the cheap global barrier |
| `ComputePipelineCreateInfo` (new) | `Stage`, `Layout` | §5.1 |
| `WriteDescriptorSet` | `BufferInfo []DescriptorBufferInfo` | §5.6, optional |
| `SamplerCreateInfo` | `MinLod`, `CompareEnable`, `CompareOp` | comparison samplers — check whether the shadow path wants hardware PCF |

## 7. Suggested order

| # | batch | new funcs | effort | unlocks |
|---|---|---|---|---|
| 1 | **Formats** (§6) | 0 | 1h | HDR targets, tonemapping — the item `FEATURES.md` records as blocked |
| 2 | **Barrier rework** — `DependencyInfo`, buffer and global barriers | 0 (1 changed) | 3h | Prerequisite for every compute hazard |
| 3 | **Compute** — `CreateComputePipeline`, `CmdDispatch`, enums | 2 | 4h | Volumetrics, probe prefilter, custom tracing, GPU-driven work |
| 4 | **Storage images + 3D views** (§6 enums) | 0 | 1h | Froxel grids, compute output targets |
| 5 | **`CmdBlitImage`** | 1 | 2h | Mip generation → PBR texture quality, IBL prefilter |
| 6 | **Blend and misc enums** (§6) | 0 | 1h | Glass, water, reverse-Z, wireframe |
| 7 | **Timestamp queries** | 5 | 4h | Real per-pass GPU timing |
| 8 | **Debug labels** | 2–3 | 2h | Readable validation output and RenderDoc captures |
| 9 | **`CmdDispatchIndirect`** | 1 | 1h | Variable-length GPU work |
| 10 | **Ray query acceleration structures** | 5 | days | Hardware `traceRay` |

**Batches 1–6 are about two days** and cover everything on the engine's current
roadmap short of ray tracing. Batches 7–8 are not features but they are the
difference between guessing and measuring — worth doing early.

Nothing in batches 1–9 is breaking except the `CmdPipelineBarrier2` signature,
which has exactly one call site per backend.
