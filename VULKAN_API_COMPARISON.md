# Vulkan API Coverage vs. howtovulkan.com

Comparison of the raw Vulkan entry points taught in the
[howtovulkan.com](https://howtovulkan.com) tutorial against the functions this
repository actually wraps (`grep -rhoE 'C\.vk[A-Za-z0-9]+' --include='*.go'`).

## Method note

- **Site list** was extracted from the tutorial's chapters (Instance Setup ->
  Cleaning Up). The tutorial uses **VMA (Vulkan Memory Allocator)** for all
  device-memory work, so raw allocation calls (`vkAllocateMemory`,
  `vkMapMemory`, `vkBind*Memory`, `vkGet*MemoryRequirements`, `vkFreeMemory`)
  are hidden behind VMA on the site and do **not** appear as explicit `vk*`
  calls there.
- This repo has **no VMA C dependency**. Instead `vk/vma.go` is a pure-Go
  **VMA substitute** exposing the same API shape (`VmaCreateAllocator`,
  `VmaCreateBuffer`, `VmaCreateImage`, `VmaDestroy*`) built on the raw
  allocation wrappers below, so `cmd/howto/main.go` mirrors the reference's
  VMA call sites 1:1.
- The site teaches `vkQueueSubmit`; this repo uses the newer **`vkQueueSubmit2`**
  (synchronization2). Same for barriers: both use `vkCmdPipelineBarrier2`.

Legend: Yes recreated here | No not present here | NEW in repo, not on site.

---

## 1. Functions taught on the site - recreated here?

| Vulkan function | On site | In repo | Status |
|---|---|---|---|
| vkCreateInstance | Yes | Yes | Yes |
| vkEnumeratePhysicalDevices | Yes | Yes | Yes |
| vkGetPhysicalDeviceProperties2 | Yes | Yes | Yes |
| vkGetPhysicalDeviceQueueFamilyProperties | Yes | Yes | Yes |
| vkGetPhysicalDeviceFormatProperties2 | Yes | Yes | Yes |
| vkCreateDevice | Yes | Yes | Yes |
| vkGetDeviceQueue | Yes | Yes | Yes |
| vkGetBufferDeviceAddress | Yes | Yes | Yes |
| vkCreateImageView | Yes | Yes | Yes |
| vkGetPhysicalDeviceSurfaceCapabilitiesKHR | Yes | Yes | Yes |
| vkCreateSwapchainKHR | Yes | Yes | Yes |
| vkGetSwapchainImagesKHR | Yes | Yes | Yes |
| vkAcquireNextImageKHR | Yes | Yes | Yes |
| vkCreateShaderModule | Yes | Yes | Yes |
| vkCreatePipelineLayout | Yes | Yes | Yes |
| vkCreateGraphicsPipelines | Yes | Yes | Yes |
| vkCreateDescriptorSetLayout | Yes | Yes | Yes |
| vkCreateDescriptorPool | Yes | Yes | Yes |
| vkAllocateDescriptorSets | Yes | Yes | Yes |
| vkUpdateDescriptorSets | Yes | Yes | Yes |
| vkCreateCommandPool | Yes | Yes | Yes |
| vkAllocateCommandBuffers | Yes | Yes | Yes |
| vkBeginCommandBuffer | Yes | Yes | Yes |
| vkResetCommandBuffer | Yes | Yes | Yes |
| vkEndCommandBuffer | Yes | Yes | Yes |
| vkCmdPipelineBarrier2 | Yes | Yes | Yes |
| vkCmdCopyBufferToImage | Yes | Yes | Yes |
| vkCreateFence | Yes | Yes | Yes |
| vkCreateSemaphore | Yes | Yes | Yes |
| vkWaitForFences | Yes | Yes | Yes |
| vkResetFences | Yes | Yes | Yes |
| vkQueuePresentKHR | Yes | Yes | Yes |
| vkCreateSampler | Yes | Yes | Yes |

---

## 2. On the site, NOT recreated here (No)

| Vulkan function | Why it's missing here |
|---|---|
| vkQueueSubmit | Superseded here by **`vkQueueSubmit2`** (synchronization2). Functionally the site's submit path exists in the repo - just the newer entry point. Not a gap, a version difference. |
| vkAllocateMemory / vkMapMemory / vkUnmapMemory / vkBindBufferMemory / vkBindImageMemory / vkGetBufferMemoryRequirements / vkGetImageMemoryRequirements / vkFreeMemory | On the site these are **hidden inside VMA**, so they aren't taught as raw calls. The repo implements them directly (see Sec 3) and layers the `vk/vma.go` VMA substitute on top. So: present here, just categorized under the site as "VMA-abstracted." |

> Net truly-absent-and-needed: **nothing**. Everything the site "uses" is
> recreated, a newer variant, or VMA-hidden (and covered by the substitute).

---

## 3. In this repo, NOT taught as raw calls on the site (NEW) - and why

These exist here because this repo wraps Vulkan **without VMA** and covers the
full draw/present/teardown surface the tutorial narrates but delegates.

### Raw memory management (the layer `vk/vma.go` is built on)
| Function | Why needed here |
|---|---|
| vkAllocateMemory NEW | `VmaCreateBuffer`/`VmaCreateImage` allocate `VkDeviceMemory` through this. |
| vkFreeMemory NEW | Symmetric teardown of the above (`VmaDestroy*`). |
| vkMapMemory / vkUnmapMemory NEW | Persistent mapping behind `VmaAllocationCreateMapped`. |
| vkBindBufferMemory / vkBindImageMemory NEW | Bind allocations to buffers/images (real VMA does this internally). |
| vkGetBufferMemoryRequirements / vkGetImageMemoryRequirements NEW | Size/alignment/memory-type bits for allocation, normally answered by VMA. |
| vkGetPhysicalDeviceMemoryProperties2 NEW | The memory-type heap walk `VmaAllocator` caches at creation. |

### Buffer/image lifecycle (site delegates create+alloc to VMA)
| Function | Why needed here |
|---|---|
| vkCreateBuffer / vkDestroyBuffer NEW | First/last step inside `VmaCreateBuffer`/`VmaDestroyBuffer`; also usable raw. |
| vkCreateImage / vkDestroyImage NEW | Same, inside `VmaCreateImage`/`VmaDestroyImage`. |

### Draw-time command recording (narrated on site, enumerated here)
| Function | Why needed here |
|---|---|
| vkCmdBeginRendering / vkCmdEndRendering NEW | Dynamic rendering (no render pass); the pipeline wrapper is built around this. |
| vkCmdBindPipeline NEW | Bind the graphics pipeline before drawing. |
| vkCmdBindDescriptorSets NEW | Bind textures/UBOs for the draw. |
| vkCmdBindVertexBuffers / vkCmdBindIndexBuffer NEW | Bind mesh geometry. |
| vkCmdDrawIndexed NEW | The actual indexed draw call. |
| vkCmdPushConstants NEW | Fast per-draw uniforms (matches `PushConstantRange` in `vk/pipeline.go`). |
| vkCmdSetViewport / vkCmdSetScissor NEW | Dynamic viewport+scissor (the pipeline declares them dynamic). |

### Synchronization / submission completeness
| Function | Why needed here |
|---|---|
| vkQueueSubmit2 NEW | Modern submit (synchronization2) - the repo's chosen submit path vs the site's `vkQueueSubmit`. |
| vkDeviceWaitIdle NEW | Full-device drain before teardown / swapchain recreate. |
| vkQueueWaitIdle NEW | Coarse per-queue sync (e.g. one-shot upload transfers). |

### Surface + swapchain query surface
| Function | Why needed here |
|---|---|
| vkGetPhysicalDeviceSurfaceSupportKHR NEW | Confirm a queue family can present to the surface. |
| vkGetPhysicalDeviceSurfaceFormatsKHR NEW | Choose swapchain format/colorspace. |
| vkGetPhysicalDeviceSurfacePresentModesKHR NEW | Choose present mode (FIFO/mailbox). |
| vkDestroySurfaceKHR / vkDestroySwapchainKHR NEW | Teardown/recreate on resize. |

### Full destroy set (the site's "Cleaning Up" chapter, made explicit)
`vkDestroyInstance`, `vkDestroyDevice`, `vkDestroyImageView`,
`vkDestroyShaderModule`, `vkDestroyPipeline`, `vkDestroyPipelineLayout`,
`vkDestroyDescriptorSetLayout`, `vkDestroyDescriptorPool`,
`vkDestroyCommandPool`, `vkDestroyFence`, `vkDestroySemaphore`,
`vkDestroySampler` - all NEW as explicit wrappers. The tutorial performs cleanup
in prose; a reusable library must expose each teardown symmetric to its create.

---

## 4. Summary

- **Recreated:** every core creation/binding/sync/present function the tutorial
  teaches (Sec 1), including runtime format probing - full
  triangle->mesh->texture->present path is covered.
- **Genuinely not recreated:** nothing. `vkQueueSubmit` is covered via
  `vkQueueSubmit2`.
- **Extra here:** the entire **raw memory layer** the site hands to VMA -
  wrapped again by the pure-Go **VMA substitute** in `vk/vma.go` so the demo
  keeps the reference's `vmaCreateBuffer`/`vmaCreateImage` call shape - plus
  **explicit destroy wrappers** and **draw-time `vkCmd*` recording**. Needed
  because this is a reusable Go binding, not a linear tutorial - every create
  needs a matching destroy.

_Updated 2026-07-18. Repo function list from source grep; site list from
howtovulkan.com chapter extraction (VMA-abstracted memory calls noted)._
