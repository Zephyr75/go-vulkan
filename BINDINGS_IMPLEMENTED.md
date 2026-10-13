# Bindings implemented

Every Vulkan command the `vk` package wraps, with who calls it. The commands still
missing from core are in `BINDINGS_MISSING.md`.

Audited 2026-10-13 against `how_to_vulkan/main.go`, `overdrive/src/vulkan/*.go`
and `gutter/examples/vulkan/host.go`. A command's name is given as its Go wrapper,
with the C command underneath when the two differ.

Caller files are separated by program, in the order the **used by** column
lists them: `main.go`; `backend.go`; `host.go` means `how_to_vulkan/main.go`,
`overdrive/src/vulkan/backend.go` and `gutter/examples/vulkan/host.go`.

**Used by** lists the programs that reach the command. Some commands are only
reached through `vk/vma.go`, the pure-Go VMA substitute. All three programs create
buffers and images through it, so these list all three, with `vk/vma.go` as the
caller.

Not listed here: Go-side helpers that wrap no Vulkan command (`FindMemoryType`,
`MemCopy`, `ClearColor`, `ClearDepthStencil`) and the VMA API itself
(`VmaCreateAllocator`, `VmaCreateBuffer`, `VmaCreateImage` and their destroys).

---

## 1. Core — 77 functions

Vulkan 1.0 unless marked. The 1.1–1.3 commands are why the package needs a Vulkan
1.3 device.

| name | used by | caller file(s) | what it does |
|---|---|---|---|
| **Instance and physical device** | | | |
| `CreateInstance` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Opens the connection to the Vulkan loader with a set of instance extensions, e.g. the surface extensions GLFW requires |
| `DestroyInstance` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Closes the connection to the loader |
| `EnumerateInstanceExtensionProperties` | overdrive | `backend.go` | Lists the instance extensions the loader offers. Used to check for `VK_EXT_debug_utils` before enabling it, since enabling an extension the loader lacks makes instance creation fail |
| `EnumeratePhysicalDevices` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Lists the GPUs in the machine |
| `GetPhysicalDeviceProperties2` (1.1) | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Returns a GPU's properties and limits, e.g. its name and device type |
| `GetPhysicalDeviceQueueFamilyProperties` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Lists a GPU's queue families and what each supports, used to find one with graphics |
| `GetPhysicalDeviceFormatProperties2` (1.1) | how_to_vulkan, overdrive | `main.go`; `backend.go` | Reports what a format can be used for, e.g. whether `D32SfloatS8Uint` works as a depth attachment |
| `GetPhysicalDeviceMemoryProperties2` (1.1) | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Lists the GPU's memory types and heaps. Queried once when the allocator is created |
| **Device and queues** | | | |
| `CreateDevice` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Creates the logical device with its queues, extensions and the 1.0/1.2/1.3 feature chain |
| `DestroyDevice` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys the logical device |
| `GetDeviceQueue` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Returns a queue created with the device |
| `DeviceWaitIdle` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `buffer.go`, `pipeline.go`, `swapchain.go`; `host.go` | Blocks until the GPU has finished all work, e.g. before a swapchain rebuild or teardown |
| `QueueWaitIdle` | overdrive | `backend.go` | Blocks until one queue has finished its work. Used for one-time-submit uploads without a fence |
| `QueueSubmit2` (1.3) | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `frame.go`; `host.go` | Submits command buffers with semaphores to wait on and signal, and a fence to signal |
| **Memory** | | | |
| `AllocateMemory` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Allocates device memory. One dedicated allocation per resource |
| `FreeMemory` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Frees device memory |
| `MapMemory` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Maps device memory into the CPU's address space. It stays mapped for the allocation's lifetime |
| `UnmapMemory` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Unmaps memory mapped with `MapMemory` |
| `GetBufferMemoryRequirements` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Returns the size, alignment and allowed memory types a buffer needs |
| `GetImageMemoryRequirements` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Returns the size, alignment and allowed memory types an image needs |
| `BindBufferMemory` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Attaches an allocation to a buffer |
| `BindImageMemory` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Attaches an allocation to an image |
| `GetBufferDeviceAddress` (1.2) | how_to_vulkan, overdrive | `main.go`; `backend.go`, `buffer.go` | Returns a buffer's GPU pointer, passed as a push constant so shaders read buffers without descriptors |
| **Buffers, images and samplers** | | | |
| `CreateBuffer` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Creates a buffer with no memory behind it |
| `DestroyBuffer` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Destroys a buffer |
| `CreateImage` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Creates an image with no memory behind it, with single-sample, optimal-tiling defaults |
| `DestroyImage` | how_to_vulkan, overdrive, gutter | `vk/vma.go` | Destroys an image |
| `CreateImageView` | how_to_vulkan, overdrive, gutter | `main.go`; `image.go`, `swapchain.go`; `host.go` | Describes how an image is accessed: format, aspect, view type, and mip and layer range |
| `DestroyImageView` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `slots.go`, `swapchain.go`; `host.go` | Destroys an image view |
| `CreateSampler` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `convert.go`; `host.go` | Creates the filtering, addressing and comparison state a shader samples with |
| `DestroySampler` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys a sampler |
| **Descriptors** | | | |
| `CreateDescriptorSetLayout` | how_to_vulkan, overdrive, gutter | `main.go`; `slots.go`; `host.go` | Declares the bindings a set holds, including variable-count bindless arrays |
| `DestroyDescriptorSetLayout` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys a set layout |
| `CreateDescriptorPool` | how_to_vulkan, overdrive, gutter | `main.go`; `slots.go`; `host.go` | Reserves memory for descriptor sets |
| `DestroyDescriptorPool` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys a pool and every set allocated from it |
| `AllocateDescriptorSets` | how_to_vulkan, overdrive, gutter | `main.go`; `slots.go`; `host.go` | Allocates sets from a pool. `VariableCounts` fixes the size of a variable-count array |
| `UpdateDescriptorSets` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `slots.go`; `host.go` | Writes images, samplers or buffers into a set |
| **Shaders and pipelines** | | | |
| `CreateShaderModule` | how_to_vulkan, overdrive, gutter | `main.go`; `pipeline.go`; `host.go` | Wraps SPIR-V bytecode |
| `DestroyShaderModule` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `pipeline.go`; `host.go` | Destroys a shader module |
| `CreatePipelineLayout` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Declares the set layouts and push-constant ranges a pipeline uses |
| `DestroyPipelineLayout` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys a pipeline layout |
| `CreateGraphicsPipeline`<br>`vkCreateGraphicsPipelines` | how_to_vulkan, overdrive, gutter | `main.go`; `pipeline.go`; `host.go` | Bakes shaders and fixed-function state into one object, including the attachment formats dynamic rendering needs. Creates one pipeline per call |
| `CreateComputePipeline`<br>`vkCreateComputePipelines` | overdrive | `pipeline.go` | Creates a compute pipeline from one shader stage and a layout. Creates one pipeline per call |
| `DestroyPipeline` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `pipeline.go`, `slots.go`; `host.go` | Destroys a graphics or compute pipeline |
| **Command buffers** | | | |
| `CreateCommandPool` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Creates the allocator command buffers come from |
| `DestroyCommandPool` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys a pool and every command buffer allocated from it |
| `AllocateCommandBuffers` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Allocates command buffers from a pool |
| `BeginCommandBuffer` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `frame.go`; `host.go` | Starts recording |
| `EndCommandBuffer` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `frame.go`; `host.go` | Finishes recording |
| `ResetCommandBuffer` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Clears a buffer's recorded commands so it can be recorded again |
| **Recording: barriers and rendering** | | | |
| `CmdPipelineBarrier2` (1.3) | how_to_vulkan, overdrive, gutter | `main.go`; `barrier.go`; `host.go` | Orders work and changes image layouts, using global, buffer and image barriers |
| `CmdBeginRendering` (1.3) | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Opens a rendering pass on the given attachments with their load/store ops and clear values, without a render pass object |
| `CmdEndRendering` (1.3) | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Closes the rendering pass |
| `CmdSetViewport` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Sets the dynamic viewport |
| `CmdSetScissor` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Sets the dynamic scissor rectangle |
| **Recording: binding** | | | |
| `CmdBindPipeline` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Binds a graphics or compute pipeline for the commands that follow |
| `CmdBindDescriptorSets` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Binds descriptor sets to a pipeline layout |
| `CmdBindVertexBuffer`<br>`vkCmdBindVertexBuffers` | how_to_vulkan, overdrive | `main.go`; `frame.go` | Binds a vertex buffer. Binds one buffer per call |
| `CmdBindIndexBuffer` | how_to_vulkan, overdrive | `main.go`; `frame.go` | Binds the index buffer |
| `CmdPushConstants` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Writes a few bytes straight into the command stream, e.g. a buffer's device address |
| **Recording: draws and dispatches** | | | |
| `CmdDraw` | overdrive, gutter | `frame.go`; `host.go` | Draws non-indexed geometry, e.g. fullscreen quads |
| `CmdDrawIndexed` | how_to_vulkan, overdrive | `main.go`; `frame.go` | Draws indexed geometry, optionally instanced |
| `CmdDrawIndirect` | overdrive | `frame.go` | Draws non-indexed geometry with arguments read from a GPU buffer |
| `CmdDrawIndexedIndirect` | overdrive | `frame.go` | Draws indexed geometry with arguments read from a GPU buffer |
| `CmdDispatch` | overdrive | `frame.go` | Runs a compute grid, counted in workgroups |
| `CmdDispatchIndirect` | overdrive | `frame.go` | Runs a compute grid sized by a GPU buffer, so one pass can size the next |
| **Recording: transfers** | | | |
| `CmdCopyBuffer` | overdrive | `buffer.go`, `frame.go` | Copies regions between buffers, e.g. staging uploads and readback |
| `CmdCopyImage` | overdrive | `frame.go` | Copies regions between images, e.g. a shadow-atlas tile |
| `CmdCopyBufferToImage` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`, `image.go`; `host.go` | Copies pixels from a buffer into an image, e.g. a texture upload |
| `CmdCopyImageToBuffer` | overdrive | `frame.go` | Copies pixels from an image into a buffer, e.g. dumping the shadow atlas to a PNG |
| `CmdClearColorImage` | overdrive | `frame.go` | Clears a colour image outside a rendering pass, e.g. zeroing a storage image |
| **Synchronisation** | | | |
| `CreateFence` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Creates a GPU→CPU signal, optionally already signalled |
| `DestroyFence` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys a fence |
| `WaitForFences` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `frame.go`; `host.go` | Blocks the CPU until fences signal |
| `ResetFences` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Returns fences to the unsignalled state |
| `CreateSemaphore` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `swapchain.go`; `host.go` | Creates a binary GPU→GPU signal |
| `DestroySemaphore` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`, `swapchain.go`; `host.go` | Destroys a semaphore |

## 2. Extensions — 13 functions

| name | used by | caller file(s) | what it does |
|---|---|---|---|
| **`VK_KHR_surface`** | | | |
| `GetPhysicalDeviceSurfaceSupportKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Tells whether a queue family can present to a surface |
| `GetPhysicalDeviceSurfaceCapabilitiesKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `swapchain.go`; `host.go` | Returns the surface's limits: image counts, current extent, supported transforms |
| `GetPhysicalDeviceSurfaceFormatsKHR` | gutter | `host.go` | Lists the format and colour-space pairs the surface accepts. gutter uses it to pick a UNORM format, because its colours are already sRGB-encoded |
| `GetPhysicalDeviceSurfacePresentModesKHR` | overdrive | `swapchain.go` | Lists the present modes the surface accepts. Used to pick mailbox or immediate when vsync is off |
| `DestroySurfaceKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `backend.go`; `host.go` | Destroys the window surface GLFW created |
| **`VK_KHR_swapchain`** | | | |
| `CreateSwapchainKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `swapchain.go`; `host.go` | Creates the chain of presentable images. Recreated on resize with the old one passed as `OldSwapchain` |
| `DestroySwapchainKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `swapchain.go`; `host.go` | Destroys a swapchain |
| `GetSwapchainImagesKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `swapchain.go`; `host.go` | Returns the swapchain's images |
| `AcquireNextImageKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Picks the next image to draw into, and signals a semaphore once it is free |
| `QueuePresentKHR` | how_to_vulkan, overdrive, gutter | `main.go`; `frame.go`; `host.go` | Hands a rendered image to the screen after waiting on a semaphore |
| **`VK_EXT_debug_utils`** | | | |
| `LoadDebugUtils`<br>`vkGetInstanceProcAddr` | overdrive | `backend.go` | Fetches the two label entry points after instance creation. `vkGetInstanceProcAddr` is core 1.0, but it is only used here. If the extension is absent, the label calls do nothing |
| `CmdBeginDebugLabel`<br>`vkCmdBeginDebugUtilsLabelEXT` | overdrive | `frame.go` | Opens a labelled region, which groups a RenderDoc capture by pass |
| `CmdEndDebugLabel`<br>`vkCmdEndDebugUtilsLabelEXT` | overdrive | `frame.go` | Closes the region `CmdBeginDebugLabel` opened |
