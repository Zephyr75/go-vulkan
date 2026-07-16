# govk Library Reference

The `vk` package is a hand-written cgo binding for the subset of Vulkan 1.3 used
by the "How to Vulkan 2026" tutorial (dynamic rendering, buffer device address,
descriptor indexing, synchronization2). This document lists what each file
provides and which files can be deleted to keep only the reusable library.

---

## Function inventory by file

### `vk/vk.go` — core machinery (cgo preamble + shared helpers)
Package doc, `#cgo LDFLAGS: -lvulkan`, and the cross-cutting plumbing.

| Symbol | Kind | Purpose |
|---|---|---|
| `Result` | type | Wraps `VkResult`, implements `error`. |
| `Success`, `Timeout`, `ErrOutOfDateKHR`, `SuboptimalKHR`, `ErrDeviceLost`, … | consts | Sentinel result codes the app branches on. |
| `(Result).Error` / `(Result).String` | method | Stringify a result code. |
| `check` | func (internal) | `VkResult` → Go `error`. |
| `Instance`, `Device`, `Buffer`, `Image`, … | types | Handle types (all `uintptr`). |
| `cStrings` | func (internal) | `[]string` → `**C.char` + free. |
| `enumerate` / `enumerateVoid` | func (internal) | Generic two-call enumeration. |
| `MemCopy[T]` | func | Copy a Go slice into a mapped device pointer. |

### `vk/types.go` — enums, flag bits, small value structs
No functions. Defines `Extent2D/3D`, `Offset2D`, `Rect2D`, `Viewport`, and the
enums/flags (`Format`, `ImageLayout`, `BufferUsageFlags`, `PipelineStageFlags2`,
`AccessFlags2`, descriptor/pipeline enums, …) plus constants like `WholeSize`,
`ApiVersion13`, `QueueFamilyIgnored`.

### `vk/instance.go` — instance + physical device queries (Milestone 1)
`CreateInstance`, `DestroyInstance`, `EnumeratePhysicalDevices`,
`GetPhysicalDeviceProperties2`, `GetPhysicalDeviceQueueFamilyProperties`,
`GetPhysicalDeviceMemoryProperties2`.

### `vk/device.go` — logical device (Milestone 1)
`CreateDevice` (builds the Features2 → Vulkan12 → Vulkan13 pNext chain),
`DestroyDevice`, `GetDeviceQueue`, `DeviceWaitIdle`. Config: `Features`,
`DeviceCreateInfo`, `DeviceQueueCreateInfo`.

### `vk/swapchain.go` — surface, swapchain, image views, present (Milestones 2 & 7)
`GetPhysicalDeviceSurfaceCapabilitiesKHR`, `GetPhysicalDeviceSurfaceFormatsKHR`,
`GetPhysicalDeviceSurfacePresentModesKHR`, `GetPhysicalDeviceSurfaceSupportKHR`,
`CreateSwapchainKHR`, `GetSwapchainImagesKHR`, `DestroySwapchainKHR`,
`DestroySurfaceKHR`, `CreateImageView`, `DestroyImageView`,
`AcquireNextImageKHR`, `QueuePresentKHR`.

### `vk/memory.go` — allocation, mapping, BDA (Milestone 3)
`GetBufferMemoryRequirements`, `GetImageMemoryRequirements`, `AllocateMemory`
(chains `VkMemoryAllocateFlagsInfo` for BDA), `FreeMemory`, `BindBufferMemory`,
`BindImageMemory`, `MapMemory`, `UnmapMemory`, `GetBufferDeviceAddress`,
`FindMemoryType` (pure-Go memory-type selection).

### `vk/resources.go` — buffers, images, samplers (Milestones 3 & 5)
`CreateBuffer`, `DestroyBuffer`, `CreateImage`, `DestroyImage`, `CreateSampler`,
`DestroySampler`.

### `vk/sync.go` — fences and semaphores (Milestone 4)
`CreateFence`, `DestroyFence`, `CreateSemaphore`, `DestroySemaphore`,
`WaitForFences`, `ResetFences`.

### `vk/command.go` — pools, buffers, sync2 submit (Milestones 4 & 5)
`CreateCommandPool`, `DestroyCommandPool`, `AllocateCommandBuffers`,
`ResetCommandBuffer`, `BeginCommandBuffer`, `EndCommandBuffer`,
`QueueSubmit2` (sync2 `VkSubmitInfo2`), `QueueWaitIdle`. Config:
`SemaphoreSubmitInfo`, `SubmitInfo2`.

### `vk/cmd.go` — command recording (Milestones 5 & 7)
`ClearColor`, `ClearDepthStencil` (the `VkClearValue` union as `[16]byte`),
`CmdPipelineBarrier2`, `CmdCopyBufferToImage`, `CmdBeginRendering`,
`CmdEndRendering`, `CmdSetViewport`, `CmdSetScissor`, `CmdBindPipeline`,
`CmdBindDescriptorSets`, `CmdBindVertexBuffer`, `CmdBindIndexBuffer`,
`CmdPushConstants`, `CmdDrawIndexed`. Config: `ImageMemoryBarrier2`,
`BufferImageCopy`, `RenderingInfo`, `RenderingAttachmentInfo`.

### `vk/descriptor.go` — descriptor indexing (Milestone 5)
`CreateDescriptorSetLayout` (chains binding-flags for variable count),
`DestroyDescriptorSetLayout`, `CreateDescriptorPool`, `DestroyDescriptorPool`,
`AllocateDescriptorSets` (chains variable-descriptor-count), `UpdateDescriptorSets`.

### `vk/pipeline.go` — shaders, layout, graphics pipeline (Milestone 6)
`CreateShaderModule`, `DestroyShaderModule`, `CreatePipelineLayout`,
`DestroyPipelineLayout`, `CreateGraphicsPipeline` (~10 substates +
`VkPipelineRenderingCreateInfo` pNext for dynamic rendering), `DestroyPipeline`.

### `vk/smoke_test.go` — Milestone 1 headless test
Not part of the API. `TestInstanceDevice` creates instance + device + queue and
prints the GPU. Run: `go test ./vk/ -run Instance -v`.

---

## Minimum library: what to keep vs remove

The `vk` package is self-contained: its only dependencies are `vulkan.h` + the
Vulkan loader (`-lvulkan`) and the Go standard library. Nothing in `vk/` imports
GLFW, mathgl, or the `internal/` helpers — those belong to the demo only.

### Keep (the reusable library)
```
vk/                 # all *.go except the test, if you want zero test deps
├── vk.go
├── types.go
├── instance.go
├── device.go
├── swapchain.go
├── memory.go
├── resources.go
├── sync.go
├── command.go
├── cmd.go
├── descriptor.go
└── pipeline.go
go.mod              # trim to just: module + go version (see below)
```

### Safe to delete
| Path | Why it's removable |
|---|---|
| `cmd/demo/` | The example application. Only consumer of GLFW + mathgl. |
| `shaders/` | GLSL/SPIR-V + embed — used only by the demo. |
| `internal/obj/` | OBJ parser — demo asset loading. |
| `internal/texture/` | PNG/solid-color loader — demo asset loading. |
| `vk/smoke_test.go` | Optional. Delete if you want the package to build with no test file. |
| `go-vulkan-bindings-plan.md` | The planning doc. |
| `go.sum` | Regenerate/remove once the external requires are gone. |

### After removing the demo, trim `go.mod` to:
```
module govk

go 1.26.3
```
The `github.com/go-gl/glfw` and `github.com/go-gl/mathgl` requires exist only for
`cmd/demo`; drop them, then run `go mod tidy`. `go build ./vk/` needs no module
dependencies at all.

### One-liner
```sh
rm -rf cmd shaders internal go-vulkan-bindings-plan.md go.sum vk/smoke_test.go
# edit go.mod to remove the go-gl requires, then:
go mod tidy && go build ./vk/
```
