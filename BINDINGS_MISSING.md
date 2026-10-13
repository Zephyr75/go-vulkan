# Bindings missing

Every core Vulkan command (1.0–1.4) the `vk` package does not wrap yet. Extension
commands are left out. The commands already bound are in
`BINDINGS_IMPLEMENTED.md`.

Audited 2026-10-13 against `vulkan_core.h` header version 363: 156 core commands
are missing, of 234 in core. Grouped by the core version that introduced them.
Many later commands are newer forms of earlier ones that take an extensible info
struct (the `2` suffix). These are marked "Extensible form of …".

---

## Vulkan 1.0 — 67

| name | what it does |
|---|---|
| **Instance and device queries** | |
| `vkEnumerateInstanceLayerProperties` | Lists the layers the loader can enable, e.g. validation |
| `vkEnumerateDeviceExtensionProperties` | Lists the extensions a GPU supports |
| `vkEnumerateDeviceLayerProperties` | Lists device layers. Deprecated, since device layers no longer exist |
| `vkGetPhysicalDeviceFeatures` | Returns which optional features a GPU supports |
| `vkGetPhysicalDeviceProperties` | Returns a GPU's properties and limits. Superseded by `vkGetPhysicalDeviceProperties2` |
| `vkGetPhysicalDeviceFormatProperties` | Reports what a format can be used for. Superseded by `vkGetPhysicalDeviceFormatProperties2` |
| `vkGetPhysicalDeviceImageFormatProperties` | Reports whether an image with a given format, type, tiling and usage can be created, and its maximum size |
| `vkGetPhysicalDeviceMemoryProperties` | Lists memory types and heaps. Superseded by `vkGetPhysicalDeviceMemoryProperties2` |
| `vkGetPhysicalDeviceSparseImageFormatProperties` | Returns the block shape a sparse image of a given format would use |
| `vkGetDeviceProcAddr` | Returns a device-level function pointer, used to load extension commands or skip the loader's dispatch |
| **Submission** | |
| `vkQueueSubmit` | Submits command buffers with semaphores and a fence. Superseded by `vkQueueSubmit2` |
| `vkQueueBindSparse` | Binds memory to regions of sparse buffers and images on a queue |
| **Memory** | |
| `vkFlushMappedMemoryRanges` | Makes CPU writes to non-coherent mapped memory visible to the GPU |
| `vkInvalidateMappedMemoryRanges` | Makes GPU writes to non-coherent mapped memory visible to the CPU |
| `vkGetDeviceMemoryCommitment` | Returns how many bytes of lazily allocated memory are actually committed |
| `vkGetImageSparseMemoryRequirements` | Returns the sparse memory requirements of a sparse image |
| `vkGetImageSubresourceLayout` | Returns the offset, size and pitches of a subresource in a linear-tiled image |
| **Resources** | |
| `vkCreateBufferView` | Creates a typed view of a buffer, for texel buffers |
| `vkDestroyBufferView` | Destroys a buffer view |
| **Command buffers** | |
| `vkFreeCommandBuffers` | Returns command buffers to their pool |
| `vkResetCommandPool` | Resets every command buffer allocated from a pool at once |
| `vkCmdExecuteCommands` | Runs secondary command buffers from a primary one |
| **Descriptors** | |
| `vkFreeDescriptorSets` | Returns descriptor sets to a pool created with the free-descriptor-set flag |
| `vkResetDescriptorPool` | Returns every set allocated from a pool at once |
| **Pipeline cache** | |
| `vkCreatePipelineCache` | Creates a cache that speeds up pipeline creation, optionally seeded from saved data |
| `vkDestroyPipelineCache` | Destroys a pipeline cache |
| `vkGetPipelineCacheData` | Serialises a pipeline cache so it can be saved to disk |
| `vkMergePipelineCaches` | Merges several pipeline caches into one |
| **Render passes and framebuffers** | |
| `vkCreateRenderPass` | Creates a render pass object describing attachments, subpasses and their dependencies |
| `vkDestroyRenderPass` | Destroys a render pass |
| `vkCreateFramebuffer` | Binds a set of image views to a render pass's attachments |
| `vkDestroyFramebuffer` | Destroys a framebuffer |
| `vkGetRenderAreaGranularity` | Returns the render-area alignment that is fastest for a render pass |
| `vkCmdBeginRenderPass` | Begins a render pass instance on a framebuffer |
| `vkCmdNextSubpass` | Moves to the next subpass of the current render pass |
| `vkCmdEndRenderPass` | Ends the current render pass instance |
| **Events** | |
| `vkCreateEvent` | Creates an event, a fine-grained signal that can be set from the CPU or GPU |
| `vkDestroyEvent` | Destroys an event |
| `vkGetEventStatus` | Reads whether an event is set, from the CPU |
| `vkSetEvent` | Sets an event from the CPU |
| `vkResetEvent` | Unsets an event from the CPU |
| `vkCmdSetEvent` | Sets an event from the GPU once earlier work reaches a stage |
| `vkCmdResetEvent` | Unsets an event from the GPU |
| `vkCmdWaitEvents` | Waits on GPU events, with barriers |
| **Synchronisation** | |
| `vkGetFenceStatus` | Reads whether a fence is signalled, without blocking |
| `vkCmdPipelineBarrier` | Orders work and changes layouts. Superseded by `vkCmdPipelineBarrier2` |
| **Queries** | |
| `vkCreateQueryPool` | Creates a pool of queries: occlusion, pipeline statistics or timestamps |
| `vkDestroyQueryPool` | Destroys a query pool |
| `vkGetQueryPoolResults` | Reads query results back on the CPU |
| `vkCmdResetQueryPool` | Resets queries from the GPU before reuse |
| `vkCmdBeginQuery` | Starts an occlusion or pipeline-statistics query |
| `vkCmdEndQuery` | Ends a query |
| `vkCmdWriteTimestamp` | Writes a GPU timestamp once earlier work reaches a stage |
| `vkCmdCopyQueryPoolResults` | Copies query results into a buffer on the GPU |
| **Transfers** | |
| `vkCmdBlitImage` | Copies image regions with scaling and filtering, e.g. for mip generation |
| `vkCmdResolveImage` | Resolves a multisampled image into a single-sampled one |
| `vkCmdClearDepthStencilImage` | Clears a depth/stencil image outside a rendering pass |
| `vkCmdClearAttachments` | Clears regions of the current attachments inside a rendering pass |
| `vkCmdFillBuffer` | Fills a buffer range with a repeated 32-bit value |
| `vkCmdUpdateBuffer` | Writes up to 64 KiB of inline data into a buffer |
| **Dynamic state** | |
| `vkCmdSetLineWidth` | Sets the rasterised line width |
| `vkCmdSetDepthBias` | Sets the depth bias constant, clamp and slope, e.g. for shadow maps |
| `vkCmdSetBlendConstants` | Sets the constant colour used by constant blend factors |
| `vkCmdSetDepthBounds` | Sets the depth-bounds test range |
| `vkCmdSetStencilCompareMask` | Sets the stencil compare mask |
| `vkCmdSetStencilWriteMask` | Sets the stencil write mask |
| `vkCmdSetStencilReference` | Sets the stencil reference value |

## Vulkan 1.1 — 25

| name | what it does |
|---|---|
| **Instance and device queries** | |
| `vkEnumerateInstanceVersion` | Returns the highest Vulkan version the loader supports |
| `vkEnumeratePhysicalDeviceGroups` | Lists groups of GPUs that can act as one device, e.g. linked multi-GPU |
| `vkGetPhysicalDeviceFeatures2` | Extensible form of `vkGetPhysicalDeviceFeatures`. Fills the 1.1/1.2/1.3 feature structs in one call |
| `vkGetPhysicalDeviceQueueFamilyProperties2` | Extensible form of `vkGetPhysicalDeviceQueueFamilyProperties` |
| `vkGetPhysicalDeviceImageFormatProperties2` | Extensible form of `vkGetPhysicalDeviceImageFormatProperties` |
| `vkGetPhysicalDeviceSparseImageFormatProperties2` | Extensible form of `vkGetPhysicalDeviceSparseImageFormatProperties` |
| `vkGetPhysicalDeviceExternalBufferProperties` | Reports whether buffers can be shared with other APIs or processes, and how |
| `vkGetPhysicalDeviceExternalFenceProperties` | Reports whether fences can be shared with other APIs or processes |
| `vkGetPhysicalDeviceExternalSemaphoreProperties` | Reports whether semaphores can be shared with other APIs or processes |
| **Device and queues** | |
| `vkGetDeviceQueue2` | Extensible form of `vkGetDeviceQueue`, needed for protected queues |
| `vkGetDeviceGroupPeerMemoryFeatures` | Reports how one GPU in a group can access another's memory |
| **Memory** | |
| `vkBindBufferMemory2` | Binds memory to several buffers in one call. Extensible form of `vkBindBufferMemory` |
| `vkBindImageMemory2` | Binds memory to several images in one call. Extensible form of `vkBindImageMemory` |
| `vkGetBufferMemoryRequirements2` | Extensible form of `vkGetBufferMemoryRequirements`. Can report whether a dedicated allocation is preferred |
| `vkGetImageMemoryRequirements2` | Extensible form of `vkGetImageMemoryRequirements`. Can report whether a dedicated allocation is preferred |
| `vkGetImageSparseMemoryRequirements2` | Extensible form of `vkGetImageSparseMemoryRequirements` |
| **Command buffers** | |
| `vkTrimCommandPool` | Returns a command pool's unused memory to the system |
| `vkCmdSetDeviceMask` | Selects which GPUs in a group run the commands that follow |
| `vkCmdDispatchBase` | Runs a compute grid starting at a non-zero workgroup offset |
| **Descriptors** | |
| `vkGetDescriptorSetLayoutSupport` | Checks whether a set layout can be created, e.g. one with a very large bindless array |
| `vkCreateDescriptorUpdateTemplate` | Creates a template that writes a whole set from one packed struct |
| `vkDestroyDescriptorUpdateTemplate` | Destroys a descriptor update template |
| `vkUpdateDescriptorSetWithTemplate` | Writes a set from a packed struct using a template |
| **Samplers** | |
| `vkCreateSamplerYcbcrConversion` | Creates a YCbCr→RGB conversion for sampling video formats |
| `vkDestroySamplerYcbcrConversion` | Destroys a YCbCr conversion |

## Vulkan 1.2 — 12

| name | what it does |
|---|---|
| **Render passes** | |
| `vkCreateRenderPass2` | Extensible form of `vkCreateRenderPass` |
| `vkCmdBeginRenderPass2` | Extensible form of `vkCmdBeginRenderPass` |
| `vkCmdNextSubpass2` | Extensible form of `vkCmdNextSubpass` |
| `vkCmdEndRenderPass2` | Extensible form of `vkCmdEndRenderPass` |
| **Draws** | |
| `vkCmdDrawIndirectCount` | Indirect draws whose draw count is also read from a GPU buffer |
| `vkCmdDrawIndexedIndirectCount` | Indexed indirect draws whose draw count is also read from a GPU buffer, the core of GPU-driven culling |
| **Timeline semaphores** | |
| `vkGetSemaphoreCounterValue` | Reads a timeline semaphore's current value |
| `vkWaitSemaphores` | Blocks the CPU until timeline semaphores reach given values |
| `vkSignalSemaphore` | Sets a timeline semaphore's value from the CPU |
| **Queries** | |
| `vkResetQueryPool` | Resets queries from the CPU, with no command buffer |
| **Buffer device address** | |
| `vkGetBufferOpaqueCaptureAddress` | Returns the opaque address used to replay a buffer at the same device address, for capture tools |
| `vkGetDeviceMemoryOpaqueCaptureAddress` | Returns the same opaque address for a memory allocation |

## Vulkan 1.3 — 33

| name | what it does |
|---|---|
| **Device queries** | |
| `vkGetPhysicalDeviceToolProperties` | Lists the tools attached to the device, e.g. validation layers or RenderDoc |
| `vkGetDeviceBufferMemoryRequirements` | Returns a buffer's memory requirements from its create info, without creating it |
| `vkGetDeviceImageMemoryRequirements` | Returns an image's memory requirements from its create info, without creating it |
| `vkGetDeviceImageSparseMemoryRequirements` | Returns a sparse image's memory requirements from its create info, without creating it |
| **Private data** | |
| `vkCreatePrivateDataSlot` | Creates a slot for attaching a 64-bit user value to any Vulkan object |
| `vkDestroyPrivateDataSlot` | Destroys a private data slot |
| `vkSetPrivateData` | Stores a value on an object in a slot |
| `vkGetPrivateData` | Reads the value stored on an object in a slot |
| **Synchronisation2** | |
| `vkCmdSetEvent2` | Extensible form of `vkCmdSetEvent`, with a full dependency |
| `vkCmdResetEvent2` | Extensible form of `vkCmdResetEvent` |
| `vkCmdWaitEvents2` | Extensible form of `vkCmdWaitEvents` |
| `vkCmdWriteTimestamp2` | Extensible form of `vkCmdWriteTimestamp`, with synchronisation2 stage flags |
| **Copy commands 2** | |
| `vkCmdCopyBuffer2` | Extensible form of `vkCmdCopyBuffer` |
| `vkCmdCopyImage2` | Extensible form of `vkCmdCopyImage` |
| `vkCmdCopyBufferToImage2` | Extensible form of `vkCmdCopyBufferToImage` |
| `vkCmdCopyImageToBuffer2` | Extensible form of `vkCmdCopyImageToBuffer` |
| `vkCmdBlitImage2` | Extensible form of `vkCmdBlitImage` |
| `vkCmdResolveImage2` | Extensible form of `vkCmdResolveImage` |
| **Binding** | |
| `vkCmdBindVertexBuffers2` | Binds vertex buffers with sizes and strides set dynamically |
| **Extended dynamic state** | |
| `vkCmdSetCullMode` | Sets the cull mode dynamically |
| `vkCmdSetFrontFace` | Sets the front-face winding dynamically |
| `vkCmdSetPrimitiveTopology` | Sets the primitive topology dynamically, within the pipeline's topology class |
| `vkCmdSetViewportWithCount` | Sets the viewports and their count dynamically |
| `vkCmdSetScissorWithCount` | Sets the scissors and their count dynamically |
| `vkCmdSetDepthTestEnable` | Turns the depth test on or off dynamically |
| `vkCmdSetDepthWriteEnable` | Turns depth writes on or off dynamically |
| `vkCmdSetDepthCompareOp` | Sets the depth compare operation dynamically |
| `vkCmdSetDepthBoundsTestEnable` | Turns the depth-bounds test on or off dynamically |
| `vkCmdSetStencilTestEnable` | Turns the stencil test on or off dynamically |
| `vkCmdSetStencilOp` | Sets the stencil operations dynamically |
| `vkCmdSetRasterizerDiscardEnable` | Turns rasterizer discard on or off dynamically |
| `vkCmdSetDepthBiasEnable` | Turns depth bias on or off dynamically |
| `vkCmdSetPrimitiveRestartEnable` | Turns primitive restart on or off dynamically |

## Vulkan 1.4 — 19

| name | what it does |
|---|---|
| **Memory** | |
| `vkMapMemory2` | Extensible form of `vkMapMemory` |
| `vkUnmapMemory2` | Extensible form of `vkUnmapMemory` |
| `vkGetImageSubresourceLayout2` | Extensible form of `vkGetImageSubresourceLayout` |
| `vkGetDeviceImageSubresourceLayout` | Returns a subresource layout from an image's create info, without creating it |
| **Host image copy** | |
| `vkCopyMemoryToImage` | Copies pixels from CPU memory straight into an image, with no staging buffer or command buffer |
| `vkCopyImageToMemory` | Copies pixels from an image straight into CPU memory |
| `vkCopyImageToImage` | Copies between images on the CPU |
| `vkTransitionImageLayout` | Changes an image's layout on the CPU, for use with host image copies |
| **Binding and descriptors** | |
| `vkCmdBindIndexBuffer2` | Binds an index buffer with an explicit size |
| `vkCmdBindDescriptorSets2` | Extensible form of `vkCmdBindDescriptorSets`, binding for several stages at once |
| `vkCmdPushConstants2` | Extensible form of `vkCmdPushConstants` |
| `vkCmdPushDescriptorSet` | Writes descriptors straight into the command buffer, with no pool or set |
| `vkCmdPushDescriptorSet2` | Extensible form of `vkCmdPushDescriptorSet` |
| `vkCmdPushDescriptorSetWithTemplate` | Pushes descriptors from a packed struct using an update template |
| `vkCmdPushDescriptorSetWithTemplate2` | Extensible form of `vkCmdPushDescriptorSetWithTemplate` |
| **Dynamic rendering** | |
| `vkCmdSetRenderingAttachmentLocations` | Remaps colour attachments to fragment shader output locations within a rendering pass |
| `vkCmdSetRenderingInputAttachmentIndices` | Maps attachments to input attachment indices within a rendering pass, for tile-local reads |
| `vkGetRenderingAreaGranularity` | Returns the render-area alignment that is fastest for a dynamic rendering pass |
| **Dynamic state** | |
| `vkCmdSetLineStipple` | Sets the line stipple pattern dynamically |
