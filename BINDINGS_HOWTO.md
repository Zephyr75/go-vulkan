# Bindings used by `how_to_vulkan`

The `vk` functions `how_to_vulkan/main.go` calls. It is a port of the
howtovulkan.com program (`how_to_vulkan/_reference.cpp`): a single-pass forward
renderer that draws one textured indexed mesh. It has no compute, no offscreen
targets, no mipmaps and no second pass.

Everything else in `vk` exists for Overdrive or gutter and is listed in
`BINDINGS_OVERDRIVE_GUTTER.md`. Not here: why individual functions are shaped the way
they are, or what the engine does with them
(`overdrive/notes/tmp/BACKEND_DECISION.md`).

Audited 2026-10-06. Every function in `vk` is called by `main.go`, by Overdrive,
by gutter, or by `vk/vma.go`; the six no caller used (`CmdSetCullMode`,
`CmdSetFrontFace`, `CmdSetDepthCompareOp`, `CmdBlitImage`,
`GetPhysicalDeviceSurfaceFormatsKHR` / `GetPhysicalDeviceSurfacePresentModesKHR`)
were removed that day. `GetPhysicalDeviceSurfaceFormatsKHR` was restored on
2026-10-07 for gutter, and `GetPhysicalDeviceSurfacePresentModesKHR` on
2026-10-08 for Overdrive's vsync setting.

---

## 1. Called by `main.go`

69 functions, grouped as the reference orders them. Overdrive calls all of them
too.

### Instance and device

| function | what it does |
|---|---|
| `CreateInstance` `DestroyInstance` | Opens the connection to the Vulkan loader with a set of instance extensions — e.g. the surface extensions GLFW reports as required |
| `EnumeratePhysicalDevices` | Lists the GPUs in the machine — `main.go` takes the first, or the index given on the command line |
| `GetPhysicalDeviceProperties2` | Returns a GPU's properties — e.g. its name, for the "Selected device" log line |
| `GetPhysicalDeviceQueueFamilyProperties` | Lists a GPU's queue families and what each supports — used to find the first one with graphics |
| `GetPhysicalDeviceFormatProperties2` | Reports what a format can be used for — e.g. whether `D32SfloatS8Uint` works as a depth attachment, falling back to `D24UnormS8Uint` |
| `CreateDevice` `DestroyDevice` | Creates the logical device with its queues, extensions and features — e.g. `VK_KHR_swapchain`, dynamic rendering, buffer device address |
| `GetDeviceQueue` | Returns a queue created with the device — the single graphics queue |
| `DeviceWaitIdle` | Blocks until the GPU has finished all work — before rebuilding the swapchain on resize, and before teardown |

### Surface and swapchain

| function | what it does |
|---|---|
| `GetPhysicalDeviceSurfaceSupportKHR` | Tells whether a queue family can present to a surface. The reference assumes it; the port checks |
| `GetPhysicalDeviceSurfaceCapabilitiesKHR` | Returns the surface's limits — the minimum image count, and the current extent used to size the swapchain |
| `CreateSwapchainKHR` `DestroySwapchainKHR` | Creates the chain of presentable images — recreated on resize with the old one passed as `OldSwapchain` |
| `GetSwapchainImagesKHR` | Returns the swapchain's images — each gets an image view and a render-complete semaphore |
| `DestroySurfaceKHR` | Destroys the window surface GLFW created |
| `AcquireNextImageKHR` | Picks the next image to draw into and signals a semaphore once it is free |
| `QueuePresentKHR` | Hands a rendered image to the screen after waiting on a semaphore |

### Images and samplers

| function | what it does |
|---|---|
| `CreateImageView` `DestroyImageView` | Describes how an image is accessed — format, aspect, mip and layer range. One each for swapchain images, the depth buffer and textures |
| `CreateSampler` `DestroySampler` | Creates the filtering and addressing state a shader samples with — here linear filtering with 8× anisotropy |

### Buffers

| function | what it does |
|---|---|
| `GetBufferDeviceAddress` | Returns a buffer's GPU pointer — pushed as a constant so the vertex shader reads the uniform buffer without a descriptor |

### Descriptors

| function | what it does |
|---|---|
| `CreateDescriptorSetLayout` `DestroyDescriptorSetLayout` | Declares the bindings a set holds — here one variable-count array of combined image samplers |
| `CreateDescriptorPool` `DestroyDescriptorPool` | Reserves memory for descriptor sets — sized for one set of three samplers |
| `AllocateDescriptorSets` | Allocates sets from a pool — `VariableCounts` fixes the array's final size to the texture count |
| `UpdateDescriptorSets` | Writes resources into a set — all three texture view/sampler pairs in one write |

### Pipeline

| function | what it does |
|---|---|
| `CreateShaderModule` `DestroyShaderModule` | Wraps SPIR-V bytecode — the embedded `shaders.Vert` and `shaders.Frag` |
| `CreatePipelineLayout` `DestroyPipelineLayout` | Declares the set layouts and push-constant ranges a pipeline uses — the texture set plus 8 bytes for the buffer address |
| `CreateGraphicsPipeline` `DestroyPipeline` | Bakes shaders and fixed-function state into one object — vertex layout, depth test, blend, and the attachment formats dynamic rendering needs |

### Command buffers and submission

| function | what it does |
|---|---|
| `CreateCommandPool` `DestroyCommandPool` | Creates the allocator command buffers come from — with `ResetCommandBuffer` allowed per buffer |
| `AllocateCommandBuffers` | Allocates command buffers from a pool — one per in-flight frame, plus a one-time buffer per texture upload |
| `BeginCommandBuffer` `EndCommandBuffer` | Start and finish recording — `OneTimeSubmit` since each recording is submitted once |
| `ResetCommandBuffer` | Clears a buffer's recorded commands so it can be recorded again — once the frame's fence has signalled |
| `QueueSubmit2` | Submits command buffers with semaphores to wait on and signal, and a fence to signal — see §2 |

### Recording

| function | what it does |
|---|---|
| `CmdPipelineBarrier2` | Orders work and changes image layouts — e.g. `Undefined` → `TransferDstOptimal` before a texture upload, `ColorAttachmentOptimal` → `PresentSrcKHR` before present. `main.go` fills only `DependencyInfo.Image` |
| `CmdCopyBufferToImage` | Copies pixels from a buffer into an image — a texture's staging buffer into the texture |
| `CmdBeginRendering` `CmdEndRendering` | Open and close a rendering pass on the given attachments, with their load/store ops and clear values — no `VkRenderPass` object |
| `CmdSetViewport` `CmdSetScissor` | Set the dynamic viewport and scissor — the full window, so a resize needs no new pipeline |
| `CmdBindPipeline` | Binds a pipeline for the draws that follow |
| `CmdBindDescriptorSets` | Binds descriptor sets to a pipeline layout — the texture array at set 0 |
| `CmdBindVertexBuffer` `CmdBindIndexBuffer` | Bind the vertex and index buffers — here the same buffer, indices starting at offset `vBufSize` |
| `CmdPushConstants` | Writes a few bytes straight into the command stream — this frame's uniform-buffer address |
| `CmdDrawIndexed` | Draws indexed geometry, optionally instanced — the mesh with 3 instances in one call |

### Synchronisation

| function | what it does |
|---|---|
| `CreateFence` `DestroyFence` | A GPU→CPU signal — one per in-flight frame (created signalled so frame 0 does not block), plus one per texture upload |
| `WaitForFences` `ResetFences` | Block the CPU until fences signal, then rearm them — before reusing a frame slot |
| `CreateSemaphore` `DestroySemaphore` | A GPU→GPU signal — one per frame for image acquisition, one per swapchain image for render completion |

### VMA

| function | what it does |
|---|---|
| `VmaCreateAllocator` `VmaDestroyAllocator` | Creates the allocator: a pure-Go stand-in for the C VMA the reference links, same API shape |
| `VmaCreateBuffer` `VmaDestroyBuffer` | Creates a buffer with its memory, mapped if asked — e.g. the vertex/index buffer, written through `MappedData` |
| `VmaCreateImage` `VmaDestroyImage` | Creates an image with its memory — e.g. the depth buffer, with a dedicated allocation |

### Go-side helpers

| function | what it does |
|---|---|
| `MemCopy` | Copies a Go slice into mapped memory — e.g. vertices at offset 0, indices right after |
| `ClearColor` `ClearDepthStencil` | Build the 16-byte `ClearValue` union, which Go cannot express directly — e.g. black, or depth 1 |

## 2. Differences from the reference

| binding | reference | why the binding differs |
|---|---|---|
| `QueueSubmit2` | `vkQueueSubmit` (1.0) | Synchronization2 throughout; the reference mixes the old submit with `vkCmdPipelineBarrier2` |
| `CreateGraphicsPipeline`, `CmdBindVertexBuffer` (singular) | plural | Batch creation and multi-buffer binding are unused; the singular form drops a slice parameter |

Two things the reference does that the binding deliberately does **not** expose:
`vkGetInstanceProcAddr` / `vkGetDeviceProcAddr` (extension loading — cgo links
`libvulkan` directly instead), and `vkCreateGraphicsPipelines` batching.

## 3. Called only through VMA

The reference links the real C VMA and never touches raw memory. `vk/vma.go` is
pure Go and composes these 14, so `main.go` depends on them without calling
them. `vk/vk_test.go` exercises them directly.

| function | what it does |
|---|---|
| `CreateBuffer` `DestroyBuffer` | Creates a buffer with no memory behind it — `VmaCreateBuffer` binds memory afterwards |
| `CreateImage` `DestroyImage` | Creates an image with no memory behind it, filling in single-sample, optimal-tiling defaults |
| `GetBufferMemoryRequirements` `GetImageMemoryRequirements` | Return the size, alignment and allowed memory types a resource needs |
| `GetPhysicalDeviceMemoryProperties2` | Lists the GPU's memory types and heaps — queried once when the allocator is created |
| `FindMemoryType` | Picks a memory type that fits the requirements and the wanted properties — e.g. host-visible for a mapped buffer |
| `AllocateMemory` `FreeMemory` | Allocate and free device memory — one dedicated allocation per resource |
| `BindBufferMemory` `BindImageMemory` | Attach an allocation to a buffer or image |
| `MapMemory` `UnmapMemory` | Map device memory into the CPU's address space — kept mapped for the allocation's lifetime |
