# Bindings only Overdrive uses

The `vk` surface Overdrive needs beyond what `how_to_vulkan/main.go` calls, and
what is still to be added. The tutorial's set, which Overdrive also uses in
full, is in `BINDINGS_HOWTO.md`.

Audited 2026-10-06 against `overdrive/src/vulkan/*.go`. Every function below has
a caller there; anything that lost its caller has been removed rather than kept
in case.

---

## 1. Functions

15, none of them in the reference program.

| function | caller | why it exists |
|---|---|---|
| `EnumerateInstanceExtensionProperties` | `backend.go` | `VK_EXT_debug_utils` is optional, so it is asked for rather than assumed — enabling an extension the loader lacks fails instance creation |
| `LoadDebugUtils` `CmdBeginDebugLabel` `CmdEndDebugLabel` | `backend.go`, `frame.go` | Groups a RenderDoc capture by pass. The label entry points come from an extension, so `LoadDebugUtils` fetches them after instance creation and leaves both calls as no-ops when the extension is absent (`vk/label.go`) |
| `QueueWaitIdle` | `backend.go` | One-time-submit teardown without a fence |
| `CreateComputePipeline` | `pipeline.go` | One stage plus a layout. `DestroyPipeline` covers teardown |
| `CmdDispatch` | `frame.go` | Compute grid, in workgroups |
| `CmdDispatchIndirect` | `frame.go` | One compute pass sizes the next — variable-length GPU work |
| `CmdDraw` | `frame.go` | Non-indexed draw: fullscreen quads and the skybox |
| `CmdDrawIndexedIndirect` `CmdDrawIndirect` | `frame.go` | GPU-driven draws |
| `CmdCopyBuffer` | `buffer.go`, `frame.go` | A device-local buffer's initial contents, and readback out of one |
| `CmdCopyImage` | `frame.go` | Shadow atlas: a depth tile is copied from the static atlas into the dynamic one, and only the movable casters are redrawn on top |
| `CmdCopyImageToBuffer` | `frame.go` | Mirror of `CmdCopyBufferToImage`, sharing `BufferImageCopy`. Dumps the shadow atlas to a PNG — the only way to eyeball a depth target with no screenshot path |
| `CmdClearColorImage` | `frame.go` | Zeroes a storage image before the compute pass that accumulates into it |

## 2. Types, enums and fields

Beyond the tutorial's needs, all added for the `Backend`/`Frame`/`Pass`/`Compute`
rewrite (`overdrive/notes/tmp/INTERFACE_PLAN.md`). `vk/types.go` groups the
constants by the batch that added them.

| addition | for |
|---|---|
| `DependencyInfo.Buffer` / `.Memory`, `BufferMemoryBarrier2`, `MemoryBarrier2` | Compute→graphics hazards on buffers, and the cheap global barrier. `main.go` fills only `DependencyInfo.Image` |
| `ComputePipelineCreateInfo`, `PipelineBindPointCompute`, `ShaderStageCompute` / `All`, compute stage and storage access masks | Compute |
| `DescriptorTypeStorageImage`, `ImageUsageStorage`, `ImageViewType3D` / `1D` / `CubeArray`, `BufferUsageIndirectBuffer` | Froxel grids, compute output targets, probe arrays, indirect args |
| HDR, single-channel and BC formats; `FormatFeature*` probing bits | HDR targets and tonemapping, compressed textures |
| Extra `BlendFactor` / `BlendOp`, `CompareOp`, `SamplerAddressModeMirroredRepeat`, `PolygonModeLine`, depth dynamic states | Glass, water, reverse-Z, wireframe |
| `BufferCopy`, `ImageCopy` | `CmdCopyBuffer`, `CmdCopyImage` |
| `SamplerCreateInfo.MinLod` / `CompareEnable` / `CompareOp` | Mip ranges, and hardware PCF |
| `VmaAllocationCreateHostAccessRandom` | A readback maps and reads every byte, which write-combined memory makes unusably slow |

Already expressible without anything Overdrive-specific: multiple colour
attachments (the attachment lists are slices), cube render targets
(`ImageCreateCubeCompatible` + `ImageViewTypeCube` + `RenderingInfo.LayerCount`),
MSAA resolve, bindless texture arrays, and storage buffers reached by device
address the same way the uniform buffer is.

## 3. Removed

Bound for Overdrive, then dropped once nothing called them:

| function | removed | why |
|---|---|---|
| `CreateQueryPool` `DestroyQueryPool` `CmdResetQueryPool` `CmdWriteTimestamp2` `GetQueryPoolResults` | 2026-09-05 | Nothing displayed per-pass GPU timings, and RenderDoc profiles per pass. `vk/query.go` went with them |
| `SetDebugUtilsObjectNameEXT`, `CreateDebugMessenger` / `DestroyDebugMessenger` | 2026-09-05 | Size and format already identify this engine's dozen images |
| `CmdSetFrontFace` | 2026-08-05 | Front face is a pass's winding convention, so it belongs in the pipeline |
| `CmdSetCullMode` `CmdSetDepthCompareOp` | 2026-10-06 | Cull mode and depth compare are baked into pipeline objects. `DynamicStateCullMode` / `FrontFace` / `DepthCompareOp` went with them |
| `CmdBlitImage` | 2026-10-06 | Mip generation never landed. `ImageBlit` and `Offset3D` went with it |
| `GetPhysicalDeviceSurfaceFormatsKHR` `GetPhysicalDeviceSurfacePresentModesKHR` | 2026-10-06 | The swapchain hardcodes an sRGB BGRA format and FIFO. `SurfaceFormat` went with them |

## 4. Still to add

### Ray queries — 5 functions

| function | note |
|---|---|
| `GetAccelerationStructureBuildSizesKHR` `CreateAccelerationStructureKHR` `DestroyAccelerationStructureKHR` `CmdBuildAccelerationStructuresKHR` `GetAccelerationStructureDeviceAddressKHR` | The BLAS/TLAS lifecycle. **This is all ray queries need** — `rayQueryEXT` is inline in a compute shader, so no new pipeline type and no shader binding table |

Plus device-feature and extension plumbing: `VK_KHR_acceleration_structure`,
`VK_KHR_ray_query`, `VK_KHR_deferred_host_operations`, and the
`PhysicalDeviceAccelerationStructureFeaturesKHR` / `RayQueryFeaturesKHR` chain.
Realistically 600–1000 lines including the structure-geometry descriptors, which
are the fiddliest part of the Vulkan API. Additive — nothing existing changes.

`CreateRayTracingPipelinesKHR` `GetRayTracingShaderGroupHandlesKHR`
`CmdTraceRaysKHR` are only for the full stage machinery
(raygen/anyhit/closesthit/callable). Skip unless intersection or callable
shaders are actually wanted.

### When a caller needs them

| addition | trigger |
|---|---|
| `CmdBlitImage` (restore) | Runtime mip generation: PBR texture quality, IBL and probe prefiltering |
| `WriteDescriptorSet.BufferInfo` | A shader needs a buffer the compiler will not accept as a pointer. Device address works for everything so far |
