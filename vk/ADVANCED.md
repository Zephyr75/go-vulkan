# vk package internals

How the non-trivial functions of the binding work, and where their behavior is
specified. Trivial 1:1 wrappers (`DestroyFence`, `CmdEndRendering`, …) are not
listed — they convert handles and call the C function, nothing else.

Layering, bottom to top:

```
libvulkan / driver          real work
vk (cgo bindings)           1:1 wrappers, Go types in / Go errors out
vk/vma.go (pure Go)         VMA-shaped allocator built on the bindings
cmd/howto                   the demo, mirroring _reference.cpp
```

General references:

- Vulkan man pages (per function): `https://registry.khronos.org/vulkan/specs/latest/man/html/<vkName>.html`
- Vulkan spec chapters: <https://docs.vulkan.org/spec/latest/index.html>
- cgo pointer-passing rules (why the C-heap marshaling below exists):
  <https://pkg.go.dev/cmd/cgo#hdr-Passing_pointers>

---

## Cross-cutting machinery (`vk.go`, `pipeline.go`)

### `check(r C.VkResult) error`

Maps `VK_SUCCESS` to `nil`, everything else to the `Result` error type. Callers
compare against exported sentinels (`ErrOutOfDateKHR`, `SuboptimalKHR`) with
`==`. Note `VK_SUBOPTIMAL_KHR` is a *success* code in Vulkan (positive value)
but is surfaced as an error here so swapchain callers can branch on it.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkResult.html>

### `enumerate[T]` / `enumerateVoid[T]`

Centralizes Vulkan's two-call enumeration idiom: call with `pCount, nil` to get
the element count, allocate `make([]T, count)`, call again with `pCount,
&out[0]` to fill it, return `out[:count]` (the driver may legally shrink the
count between calls). The closure passed in binds the fixed leading arguments
(device, surface, …) and exposes only the `(count, out)` tail. `T` is inferred
from the closure's `out` parameter. Passing `&out[0]` (Go memory) to C is legal
because the slice contains no Go pointers and C does not retain it.

- Idiom description: <https://docs.vulkan.org/spec/latest/chapters/fundamentals.html#fundamentals-commandsyntax-results>

### `cStrings([]string)`

Marshals a Go string slice to `char**` for `ppEnabledExtensionNames` /
`ppEnabledLayerNames`: one `malloc` for the pointer array, one `C.CString`
per element, and a returned `free` closure that releases all of it after the
create call (Vulkan deep-copies create-info structs, so freeing immediately
after the call is safe).

### `arena` (`pipeline.go`)

cgo forbids passing C a Go pointer that itself points to Go memory, so every
*nested* Vulkan struct (`pNext` chains, `pStages`, `pQueueCreateInfos`, …) is
built in C heap via `calloc`. The arena records each allocation and frees them
all together once the `vkCreate*` call returns. Same deep-copy argument as
above makes the immediate free safe.

- <https://pkg.go.dev/cmd/cgo#hdr-Passing_pointers>

### `MemCopy[T](dst unsafe.Pointer, src []T)`

`memcpy` of a Go slice into a mapped device pointer. This is the upload path
for persistently mapped buffers (equivalent of the reference's `memcpy` into
`VmaAllocationInfo::pMappedData`).

---

## VMA substitute (`vma.go`)

Pure Go, no cgo. Reimplements the *shape* of the VMA API by composing the raw
memory bindings. Real VMA sub-allocates large `VkDeviceMemory` blocks and hands
out offsets; this substitute gives every resource its own dedicated allocation,
which is enough at tutorial scale and sidesteps alignment/offset bookkeeping.

Primary references (these are the docs to read before extending it):

- VMA documentation root: <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/>
- Quick start (allocator + resource creation): <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/quick_start.html>
- Choosing memory types (what `VMA_MEMORY_USAGE_AUTO` + host-access flags mean): <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/choosing_memory_type.html>
- Memory mapping / persistently mapped memory (`VMA_ALLOCATION_CREATE_MAPPED_BIT`): <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/memory_mapping.html>
- Usage patterns (staging, ReBAR, GPU-only): <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/usage_patterns.html>
- VMA source (the real implementation, single header): <https://github.com/GPUOpen-LibrariesAndSDKs/VulkanMemoryAllocator>

### `VmaCreateAllocator(VmaAllocatorCreateInfo) *VmaAllocator`

Mirrors `vmaCreateAllocator`. Real VMA builds function tables and heap
bookkeeping; this substitute only caches the device, the physical device's
memory properties (queried once via `GetPhysicalDeviceMemoryProperties2`), and
the flags. Nothing can fail, hence no error return. The
`VmaAllocatorCreateBufferDeviceAddressBit` flag is remembered so buffer
allocations can request device-address-capable memory later.

- <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/quick_start.html#quick_start_initialization>

### `(*VmaAllocator) VmaCreateBuffer(BufferCreateInfo, VmaAllocationCreateInfo) (Buffer, VmaAllocation, VmaAllocationInfo, error)`

Mirrors `vmaCreateBuffer`, which bundles five raw Vulkan steps:

1. `vkCreateBuffer` — create the handle (no memory yet).
2. `vkGetBufferMemoryRequirements` — size (may exceed the requested size),
   alignment, and the `memoryTypeBits` mask of compatible memory types.
3. Pick a memory type index (`memoryType`, below) from that mask plus the
   host-access flags.
4. `vkAllocateMemory` — dedicated allocation. If the allocator was created with
   the buffer-device-address flag *and* the buffer usage includes
   `BufferUsageShaderDeviceAddress`, the allocation chains
   `VkMemoryAllocateFlagsInfo{VK_MEMORY_ALLOCATE_DEVICE_ADDRESS_BIT}`
   (see `AllocateMemory` below) — required or `vkGetBufferDeviceAddress` on the
   buffer is invalid.
5. `vkBindBufferMemory` at offset 0 (real VMA binds at a sub-allocation
   offset; dedicated allocations always bind at 0).
6. With `VmaAllocationCreateMapped`: `vkMapMemory(0, VK_WHOLE_SIZE)` once, and
   the pointer stays mapped for the allocation's lifetime ("persistent
   mapping"); it is returned in `VmaAllocationInfo.MappedData`, mirroring
   `VmaAllocationInfo::pMappedData`.

Every failure path unwinds what was created before returning the error.

- <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/quick_start.html#quick_start_resource_creation>
- Buffer creation without VMA (what steps 1–5 replace): <https://docs.vulkan.org/tutorial/latest/04_Vertex_buffers/01_Vertex_buffer_creation.html>

### `(*VmaAllocator) VmaCreateImage(ImageCreateInfo, VmaAllocationCreateInfo) (Image, VmaAllocation, error)`

Same pipeline as `VmaCreateBuffer` minus mapping (images here are always
device-local and never host-visible): create, query requirements, pick type,
allocate, bind. Mirrors `vmaCreateImage`, whose `pAllocationInfo` parameter the
reference passes as `nullptr` — hence no `VmaAllocationInfo` return.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkGetImageMemoryRequirements.html>

### `(*VmaAllocator) VmaDestroyBuffer` / `VmaDestroyImage`

Mirror `vmaDestroyBuffer` / `vmaDestroyImage`: unmap if the allocation was
persistently mapped (buffers only), destroy the resource, free its memory.
Order matters only in that the resource must not be in use by the GPU; the
callers guarantee that with fences / `DeviceWaitIdle`.

### `memoryType(bits uint32, flags VmaAllocationCreateFlags) (uint32, error)`

The heap walk VMA normally hides. Given the `memoryTypeBits` mask from the
requirements query:

- host-writable request (`VmaAllocationCreateHostAccessSequentialWrite`):
  prefer `DEVICE_LOCAL | HOST_VISIBLE | HOST_COHERENT` — a ReBAR/SAM memory
  type where CPU writes land directly in VRAM — falling back to plain
  `HOST_VISIBLE | HOST_COHERENT` (system RAM visible to the GPU);
- otherwise: `DEVICE_LOCAL`.

`HOST_COHERENT` is required by both host paths so writes need no explicit
`vkFlushMappedMemoryRanges`. This mirrors what `VMA_MEMORY_USAGE_AUTO` +
host-access flags resolve to in real VMA.

- <https://gpuopen-librariesandsdks.github.io/VulkanMemoryAllocator/html/choosing_memory_type.html>
- Memory types/heaps explained: <https://docs.vulkan.org/spec/latest/chapters/memory.html#memory-device>

---

## Raw memory (`memory.go`)

### `AllocateMemory(Device, MemoryAllocateInfo) (DeviceMemory, error)`

Wraps `vkAllocateMemory`. With `DeviceAddress: true` it chains
`VkMemoryAllocateFlagsInfo` with `VK_MEMORY_ALLOCATE_DEVICE_ADDRESS_BIT` into
`pNext` — mandatory for memory backing buffers whose address is taken with
`vkGetBufferDeviceAddress`. The flags struct is a Go local, so it is pinned
(`runtime.Pinner`) for the duration of the call to satisfy cgo.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkMemoryAllocateFlagsInfo.html>
- Buffer device address overview: <https://docs.vulkan.org/samples/latest/samples/extensions/buffer_device_address/README.html>

### `MapMemory` / `UnmapMemory`

`vkMapMemory` returns a host pointer into the allocation; the binding exposes
`WholeSize` (`VK_WHOLE_SIZE`) for the common map-everything case. Only valid on
`HOST_VISIBLE` memory. A mapping stays valid until `UnmapMemory` or
`FreeMemory` — that's what makes persistent mapping possible.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkMapMemory.html>

### `GetBufferDeviceAddress(Device, Buffer) uint64`

Wraps `vkGetBufferDeviceAddress`. Returns the 64-bit GPU pointer the shader
dereferences. Valid only for buffers created with
`BufferUsageShaderDeviceAddress` on BDA-flagged memory, with the
`bufferDeviceAddress` device feature enabled.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkGetBufferDeviceAddress.html>

### `FindMemoryType(props, typeBits, want) (uint32, error)`

Linear scan over `VkPhysicalDeviceMemoryProperties.memoryTypes`: index `i`
qualifies when bit `i` is set in `typeBits` *and* the type's property flags
contain all of `want`. This is the canonical loop from every Vulkan tutorial.

- <https://docs.vulkan.org/tutorial/latest/04_Vertex_buffers/01_Vertex_buffer_creation.html> (section "Memory requirements")

---

## Instance & device (`instance.go`, `device.go`)

### `CreateInstance(InstanceCreateInfo) (Instance, error)`

Builds `VkApplicationInfo` (+ API version) and `VkInstanceCreateInfo`,
marshaling the extension list with `cStrings`. The extension list comes from
the windowing system (`window.GetRequiredInstanceExtensions()`), which is why
the demo creates the window first.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkCreateInstance.html>

### `Features.chain(*arena) *C.VkPhysicalDeviceFeatures2`

Flattens the Go `Features` struct into Vulkan's versioned feature chain:
`VkPhysicalDeviceFeatures2` → `VkPhysicalDeviceVulkan12Features` →
`VkPhysicalDeviceVulkan13Features`, linked through `pNext` and allocated in the
arena (nested structs must live in C heap). Each bool lands in the struct that
owns that feature: anisotropy is core 1.0, descriptor indexing/BDA are 1.2,
sync2/dynamic rendering are 1.3.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkPhysicalDeviceVulkan12Features.html>
- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkPhysicalDeviceVulkan13Features.html>

### `CreateDevice(PhysicalDevice, DeviceCreateInfo) (Device, error)`

Assembles queue create infos (each with its own priorities array in the
arena), the feature chain above (hung off `pNext` — note
`pEnabledFeatures` stays nil because features2 supersedes it), and the
extension list. One arena free tears all of it down after the call.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkCreateDevice.html>

---

## Swapchain & presentation (`swapchain.go`)

### `CreateSwapchainKHR(Device, SwapchainCreateInfo) (SwapchainKHR, error)`

Flat marshal of `VkSwapchainCreateInfoKHR` with two opinionated defaults:
`ImageArrayLayers` 0 → 1, sharing mode fixed to `EXCLUSIVE` (single-queue
renderer). `OldSwapchain` enables seamless recreation on resize: the driver
can carry state over, and the old swapchain is destroyed by the caller after
the new one exists.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkSwapchainCreateInfoKHR.html>
- Swapchain recreation: <https://docs.vulkan.org/tutorial/latest/03_Drawing_a_triangle/01_Presentation/02_Swap_chain.html>

### `AcquireNextImageKHR` / `QueuePresentKHR`

Both return the out-of-date/suboptimal conditions as comparable errors instead
of treating them as fatal — the render loop branches to swapchain recreation.
`QueuePresentKHR` builds `VkPresentInfoKHR` pointing at three Go locals
(semaphore, swapchain, image index); they are pinned with `runtime.Pinner`
because the struct passed to C embeds pointers to them.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkAcquireNextImageKHR.html>
- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkQueuePresentKHR.html>

---

## Descriptors (`descriptor.go`)

### `CreateDescriptorSetLayout(Device, DescriptorSetLayoutCreateInfo)`

Marshals the bindings array; with `UseBindingFlags` it additionally chains
`VkDescriptorSetLayoutBindingFlagsCreateInfo` carrying per-binding flags
(here: `VK_DESCRIPTOR_BINDING_VARIABLE_DESCRIPTOR_COUNT_BIT` for the bindless
texture array).

- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkDescriptorSetLayoutBindingFlagsCreateInfo.html>
- Descriptor indexing sample: <https://docs.vulkan.org/samples/latest/samples/extensions/descriptor_indexing/README.html>

### `AllocateDescriptorSets(Device, DescriptorSetAllocateInfo)`

When `VariableCounts` is set, chains
`VkDescriptorSetVariableDescriptorCountAllocateInfo` so the variable-count
binding's final array size is fixed at allocation time (the layout only
declared an upper bound).

- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkDescriptorSetVariableDescriptorCountAllocateInfo.html>

### `UpdateDescriptorSets(Device, []WriteDescriptorSet)`

Builds the `VkWriteDescriptorSet` array with each write's `pImageInfo` array
in C heap (arena), then one `vkUpdateDescriptorSets` call. Only image
descriptors are supported — buffer descriptors aren't needed because the
uniform data travels via buffer device address.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkUpdateDescriptorSets.html>

---

## Pipeline (`pipeline.go`)

### `CreateShaderModule(Device, []byte)`

Validates the blob is non-empty and 4-byte aligned (SPIR-V is a word stream),
copies it into C memory (`C.CBytes`) so `pCode` carries no Go pointer, and
creates the module. The demo feeds it `go:embed`-ed SPIR-V.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkCreateShaderModule.html>

### `CreateGraphicsPipeline(Device, GraphicsPipelineCreateInfo)`

The largest marshal in the binding. Each optional sub-state
(`VertexInputState`, `InputAssemblyState`, viewport, rasterization,
multisample, depth-stencil, color-blend, dynamic state) is a Go mirror struct
translated into its `Vk*StateCreateInfo` in the arena and hung off the main
`VkGraphicsPipelineCreateInfo`. The Go structs default nothing — zero values
pass through, exactly like C. `Rendering` (dynamic rendering) is chained via
`pNext` as `VkPipelineRenderingCreateInfo`, replacing the render-pass +
subpass pair, which is why `renderPass` stays `VK_NULL_HANDLE`.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkGraphicsPipelineCreateInfo.html>
- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkPipelineRenderingCreateInfo.html>
- Dynamic rendering sample: <https://docs.vulkan.org/samples/latest/samples/extensions/dynamic_rendering/README.html>

---

## Commands (`cmd.go`, `command.go`)

### `CmdPipelineBarrier2(CommandBuffer, []ImageMemoryBarrier2)`

The single synchronization primitive the project uses. Marshals the barrier
array into C heap and wraps it in a `VkDependencyInfo` for
`vkCmdPipelineBarrier2`. Each barrier is a synchronization2 image barrier:
src/dst stage+access masks plus an old→new layout transition over a
subresource range. Queue family indices must be set explicitly
(`QueueFamilyIgnored` unless doing an ownership transfer).

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkCmdPipelineBarrier2.html>
- Synchronization2 guide: <https://docs.vulkan.org/guide/latest/extensions/VK_KHR_synchronization2.html>
- Understanding Vulkan synchronization: <https://www.khronos.org/blog/understanding-vulkan-synchronization>

### `CmdBeginRendering(CommandBuffer, RenderingInfo)`

Dynamic rendering: begins a render scope directly on the command buffer with
no `VkRenderPass`/`VkFramebuffer` objects. Marshal steps:

1. `LayerCount` 0 defaults to 1.
2. `RenderArea` → `VkRect2D`.
3. Color attachments: `calloc` an array of `VkRenderingAttachmentInfo`, fill
   each (view, layout, load/store op, clear value) via `fillAttachment`.
4. Optional depth attachment: same, single struct, hung on `pDepthAttachment`
   (nil pointer = no depth).
5. `vkCmdBeginRendering`; the C arrays are freed when the call returns
   (recorded state is copied into the command buffer).

Load/store ops replace the render-pass attachment descriptions: `Clear`+`Store`
for color, `Clear`+`DontCare` for depth (depth is never read after the frame).

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkCmdBeginRendering.html>
- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkRenderingInfo.html>
- Khronos blog on streamlining render passes: <https://www.khronos.org/blog/streamlining-render-passes>

### `CmdCopyBufferToImage(CommandBuffer, Buffer, Image, ImageLayout, []BufferImageCopy)`

Marshals copy regions (buffer offset → image mip/layer/extent) into C heap and
issues the copy. The image must already be in `TRANSFER_DST_OPTIMAL` — that's
the first barrier in the texture-upload sequence. `LayerCount` 0 defaults to 1.
A `BufferRowLength`/`BufferImageHeight` of 0 means tightly packed.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkCmdCopyBufferToImage.html>
- Texture upload walkthrough: <https://docs.vulkan.org/tutorial/latest/06_Texture_mapping/00_Images.html>

### `QueueSubmit2(Queue, []SubmitInfo2, Fence)`

Synchronization2 submit. Per submit: wait semaphores and signal semaphores are
each marshaled as `VkSemaphoreSubmitInfo` arrays (semaphore + the pipeline
stage where the wait/signal applies — the sync2 replacement for
`pWaitDstStageMask`), command buffers as `VkCommandBufferSubmitInfo`. The
optional fence is signaled when all submits complete; the render loop waits on
it to recycle the frame slot.

- <https://registry.khronos.org/vulkan/specs/latest/man/html/vkQueueSubmit2.html>
- <https://registry.khronos.org/vulkan/specs/latest/man/html/VkSemaphoreSubmitInfo.html>

---

## Handle representation (`vk.go`)

All handles — dispatchable (`VkDevice`) and non-dispatchable (`VkBuffer`) —
are `uintptr` aliases. On 64-bit platforms every Vulkan handle is
pointer-sized, so one representation covers both kinds, keeps handles opaque
and copyable, and lets external code (GLFW surface creation) construct them.
Conversions to cgo types go through `unsafe.Pointer` at each call boundary,
which is the source of `go vet`'s "possible misuse of unsafe.Pointer" noise —
expected and benign here.

- <https://docs.vulkan.org/spec/latest/chapters/fundamentals.html#fundamentals-objectmodel-overview>
