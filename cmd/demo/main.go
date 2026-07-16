// Command demo recreates the "How to Vulkan 2026" scene with the hand-written
// vk bindings: three lit, textured model instances drawn with dynamic
// rendering, synchronization2 barriers, a push-constant buffer device address
// feeding per-frame matrices, and a bindless (descriptor-indexed) texture array
// indexed by instance. Mouse drag orbits, scroll zooms, window resize drives
// swapchain recreation.
package main

import (
	"log"
	"math"
	"runtime"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"govk/internal/obj"
	"govk/internal/texture"
	"govk/shaders"
	"govk/vk"
)

const (
	framesInFlight = 2
	instanceCount  = 3
	maxTextures    = 16
	initialWidth   = 1280
	initialHeight  = 720
)

// ShaderData mirrors the Slang/GLSL SceneData (scalar layout) byte for byte.
type ShaderData struct {
	Projection [16]float32
	View       [16]float32
	Model      [instanceCount][16]float32
	LightPos   [4]float32
	Selected   uint32
}

func init() { runtime.LockOSThread() }

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// ---- managed buffer ------------------------------------------------------

type mappedBuffer struct {
	buf  vk.Buffer
	mem  vk.DeviceMemory
	ptr  unsafe.Pointer
	size uint64
}

func (r *Renderer) createMappedBuffer(size uint64, usage vk.BufferUsageFlags, bda bool) mappedBuffer {
	buf, err := vk.CreateBuffer(r.device, vk.BufferCreateInfo{Size: size, Usage: usage})
	must(err)
	req := vk.GetBufferMemoryRequirements(r.device, buf)
	idx := r.hostVisibleType(req.MemoryTypeBits)
	mem, err := vk.AllocateMemory(r.device, vk.MemoryAllocateInfo{
		AllocationSize: req.Size, MemoryTypeIndex: idx, DeviceAddress: bda,
	})
	must(err)
	must(vk.BindBufferMemory(r.device, buf, mem, 0))
	ptr, err := vk.MapMemory(r.device, mem, 0, vk.WholeSize)
	must(err)
	return mappedBuffer{buf: buf, mem: mem, ptr: ptr, size: size}
}

func (b mappedBuffer) destroy(d vk.Device) {
	vk.UnmapMemory(d, b.mem)
	vk.DestroyBuffer(d, b.buf)
	vk.FreeMemory(d, b.mem)
}

// hostVisibleType picks a ReBAR-style DEVICE_LOCAL|HOST_VISIBLE|HOST_COHERENT
// type, falling back to plain HOST_VISIBLE|HOST_COHERENT.
func (r *Renderer) hostVisibleType(bits uint32) uint32 {
	if idx, err := vk.FindMemoryType(r.memProps, bits,
		vk.MemoryPropertyDeviceLocal|vk.MemoryPropertyHostVisible|vk.MemoryPropertyHostCoherent); err == nil {
		return idx
	}
	idx, err := vk.FindMemoryType(r.memProps, bits, vk.MemoryPropertyHostVisible|vk.MemoryPropertyHostCoherent)
	must(err)
	return idx
}

func (r *Renderer) deviceLocalType(bits uint32) uint32 {
	idx, err := vk.FindMemoryType(r.memProps, bits, vk.MemoryPropertyDeviceLocal)
	must(err)
	return idx
}

// ---- texture -------------------------------------------------------------

type gpuTexture struct {
	img  vk.Image
	mem  vk.DeviceMemory
	view vk.ImageView
}

// ---- renderer ------------------------------------------------------------

type Renderer struct {
	window *glfw.Window

	instance vk.Instance
	surface  vk.SurfaceKHR
	physical vk.PhysicalDevice
	device   vk.Device
	queue    vk.Queue
	gfxFam   uint32
	memProps vk.PhysicalDeviceMemoryProperties

	surfaceFormat vk.SurfaceFormat
	depthFormat   vk.Format

	swapchain  vk.SwapchainKHR
	swapExtent vk.Extent2D
	swapImages []vk.Image
	swapViews  []vk.ImageView
	depth      gpuTexture

	cmdPool vk.CommandPool
	cmdBufs []vk.CommandBuffer

	frameFences []vk.Fence
	acquireSems []vk.Semaphore
	renderSems  []vk.Semaphore // one per swapchain image

	descLayout vk.DescriptorSetLayout
	descPool   vk.DescriptorPool
	descSet    vk.DescriptorSet
	sampler    vk.Sampler

	pipeLayout vk.PipelineLayout
	pipeline   vk.Pipeline
	vertMod    vk.ShaderModule
	fragMod    vk.ShaderModule

	mesh      *obj.Mesh
	vertexBuf mappedBuffer
	indexBuf  mappedBuffer
	scene     [framesInFlight]mappedBuffer
	sceneAddr [framesInFlight]uint64
	textures  []gpuTexture

	frame        int
	needResize   bool
	yaw, pitch   float32
	zoom         float32
	dragging     bool
	lastX, lastY float64
}

func main() {
	must(glfw.Init())
	defer glfw.Terminate()
	if !glfw.VulkanSupported() {
		log.Fatal("glfw: Vulkan not supported")
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	win, err := glfw.CreateWindow(initialWidth, initialHeight, "How to Vulkan 2026 (Go)", nil, nil)
	must(err)

	r := &Renderer{window: win, zoom: 6}
	r.initInput()
	r.initVulkan()
	r.loadScene()
	r.buildPipeline()
	defer r.cleanup()

	r.renderLoop()
}

func (r *Renderer) initInput() {
	r.window.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) { r.needResize = true })
	r.window.SetMouseButtonCallback(func(_ *glfw.Window, b glfw.MouseButton, a glfw.Action, _ glfw.ModifierKey) {
		if b == glfw.MouseButtonLeft {
			r.dragging = a == glfw.Press
			r.lastX, r.lastY = r.window.GetCursorPos()
		}
	})
	r.window.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) {
		if !r.dragging {
			return
		}
		r.yaw += float32(x-r.lastX) * 0.01
		r.pitch += float32(y-r.lastY) * 0.01
		if r.pitch > 1.5 {
			r.pitch = 1.5
		}
		if r.pitch < -1.5 {
			r.pitch = -1.5
		}
		r.lastX, r.lastY = x, y
	})
	r.window.SetScrollCallback(func(_ *glfw.Window, _, dy float64) {
		r.zoom -= float32(dy) * 0.5
		if r.zoom < 2 {
			r.zoom = 2
		}
		if r.zoom > 30 {
			r.zoom = 30
		}
	})
}

func (r *Renderer) initVulkan() {
	// Instance with the surface extensions GLFW needs plus validation.
	exts := r.window.GetRequiredInstanceExtensions()
	inst, err := vk.CreateInstance(vk.InstanceCreateInfo{
		AppName:    "howtovulkan-go",
		EngineName: "govk",
		APIVersion: vk.ApiVersion13,
		Extensions: exts,
		Layers:     []string{"VK_LAYER_KHRONOS_validation"},
	})
	must(err)
	r.instance = inst

	// Surface. GLFW wants an instance value of pointer kind and hands back a
	// pointer to the surface handle, so both ends go through unsafe.Pointer.
	instPtr := (*byte)(unsafe.Pointer(inst))
	surfRaw, err := r.window.CreateWindowSurface(instPtr, nil)
	must(err)
	r.surface = vk.SurfaceKHR(*(*uintptr)(unsafe.Pointer(surfRaw)))

	// Physical device + graphics/present queue family.
	gpus, err := vk.EnumeratePhysicalDevices(inst)
	must(err)
	if len(gpus) == 0 {
		log.Fatal("no Vulkan GPUs")
	}
	r.physical = r.pickGPU(gpus)
	props := vk.GetPhysicalDeviceProperties2(r.physical)
	log.Printf("GPU: %s", props.DeviceName)

	fams := vk.GetPhysicalDeviceQueueFamilyProperties(r.physical)
	r.gfxFam = ^uint32(0)
	for i, f := range fams {
		if f.QueueFlags&vk.QueueGraphics == 0 {
			continue
		}
		ok, err := vk.GetPhysicalDeviceSurfaceSupportKHR(r.physical, uint32(i), r.surface)
		must(err)
		if ok {
			r.gfxFam = uint32(i)
			break
		}
	}
	if r.gfxFam == ^uint32(0) {
		log.Fatal("no graphics+present queue family")
	}
	r.memProps = vk.GetPhysicalDeviceMemoryProperties2(r.physical)

	dev, err := vk.CreateDevice(r.physical, vk.DeviceCreateInfo{
		QueueCreateInfos: []vk.DeviceQueueCreateInfo{{QueueFamilyIndex: r.gfxFam, Priorities: []float32{1}}},
		Extensions:       []string{"VK_KHR_swapchain"},
		Features: vk.Features{
			SamplerAnisotropy:                            true,
			DynamicRendering:                             true,
			Synchronization2:                             true,
			BufferDeviceAddress:                          true,
			DescriptorIndexing:                           true,
			RuntimeDescriptorArray:                       true,
			DescriptorBindingPartiallyBound:              true,
			DescriptorBindingVariableDescriptorCount:     true,
			ShaderSampledImageArrayNonUniformIndexing:    true,
			DescriptorBindingSampledImageUpdateAfterBind: true,
		},
	})
	must(err)
	r.device = dev
	r.queue = vk.GetDeviceQueue(dev, r.gfxFam, 0)
	r.depthFormat = vk.FormatD32Sfloat

	r.chooseSurfaceFormat()
	r.createSwapchain(vk.SwapchainKHR(0))

	// Command pool + per-frame buffers, sync objects.
	pool, err := vk.CreateCommandPool(dev, r.gfxFam, vk.CommandPoolCreateResetCommandBuffer)
	must(err)
	r.cmdPool = pool
	r.cmdBufs, err = vk.AllocateCommandBuffers(dev, pool, framesInFlight)
	must(err)
	for i := 0; i < framesInFlight; i++ {
		f, err := vk.CreateFence(dev, vk.FenceCreateSignaled)
		must(err)
		r.frameFences = append(r.frameFences, f)
		s, err := vk.CreateSemaphore(dev)
		must(err)
		r.acquireSems = append(r.acquireSems, s)
	}
}

func (r *Renderer) pickGPU(gpus []vk.PhysicalDevice) vk.PhysicalDevice {
	for _, g := range gpus {
		if vk.GetPhysicalDeviceProperties2(g).DeviceType == vk.PhysicalDeviceTypeDiscreteGPU {
			return g
		}
	}
	return gpus[0]
}

func (r *Renderer) chooseSurfaceFormat() {
	formats, err := vk.GetPhysicalDeviceSurfaceFormatsKHR(r.physical, r.surface)
	must(err)
	r.surfaceFormat = formats[0]
	for _, f := range formats {
		if f.Format == vk.FormatB8G8R8A8Srgb && f.ColorSpace == vk.ColorSpaceSrgbNonlinearKHR {
			r.surfaceFormat = f
			break
		}
	}
}

// ---- swapchain -----------------------------------------------------------

func (r *Renderer) createSwapchain(old vk.SwapchainKHR) {
	caps, err := vk.GetPhysicalDeviceSurfaceCapabilitiesKHR(r.physical, r.surface)
	must(err)

	extent := caps.CurrentExtent
	if extent.Width == 0xFFFFFFFF { // undefined: use framebuffer size
		w, h := r.window.GetFramebufferSize()
		extent = vk.Extent2D{Width: uint32(w), Height: uint32(h)}
	}
	r.swapExtent = extent

	imgCount := caps.MinImageCount + 1
	if caps.MaxImageCount != 0 && imgCount > caps.MaxImageCount {
		imgCount = caps.MaxImageCount
	}

	sc, err := vk.CreateSwapchainKHR(r.device, vk.SwapchainCreateInfo{
		Surface:         r.surface,
		MinImageCount:   imgCount,
		ImageFormat:     r.surfaceFormat.Format,
		ImageColorSpace: r.surfaceFormat.ColorSpace,
		ImageExtent:     extent,
		ImageUsage:      vk.ImageUsageColorAttachment,
		PreTransform:    caps.CurrentTransform,
		CompositeAlpha:  vk.CompositeAlphaOpaqueKHR,
		PresentMode:     vk.PresentModeFifoKHR,
		Clipped:         true,
		OldSwapchain:    old,
	})
	must(err)
	r.swapchain = sc

	r.swapImages, err = vk.GetSwapchainImagesKHR(r.device, sc)
	must(err)
	r.swapViews = make([]vk.ImageView, len(r.swapImages))
	for i, img := range r.swapImages {
		r.swapViews[i] = r.createView(img, r.surfaceFormat.Format, vk.ImageAspectColor)
	}

	// Render-complete semaphores: one per swapchain image.
	for _, s := range r.renderSems {
		vk.DestroySemaphore(r.device, s)
	}
	r.renderSems = make([]vk.Semaphore, len(r.swapImages))
	for i := range r.renderSems {
		r.renderSems[i], err = vk.CreateSemaphore(r.device)
		must(err)
	}

	r.createDepth(extent)
}

func (r *Renderer) createView(img vk.Image, format vk.Format, aspect vk.ImageAspectFlags) vk.ImageView {
	v, err := vk.CreateImageView(r.device, vk.ImageViewCreateInfo{
		Image:    img,
		ViewType: vk.ImageViewType2D,
		Format:   format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask: aspect, LevelCount: 1, LayerCount: 1,
		},
	})
	must(err)
	return v
}

func (r *Renderer) createDepth(extent vk.Extent2D) {
	img, err := vk.CreateImage(r.device, vk.ImageCreateInfo{
		ImageType: vk.ImageType2D,
		Format:    r.depthFormat,
		Extent:    vk.Extent3D{Width: extent.Width, Height: extent.Height, Depth: 1},
		Usage:     vk.ImageUsageDepthStencilAttachment,
	})
	must(err)
	req := vk.GetImageMemoryRequirements(r.device, img)
	mem, err := vk.AllocateMemory(r.device, vk.MemoryAllocateInfo{
		AllocationSize: req.Size, MemoryTypeIndex: r.deviceLocalType(req.MemoryTypeBits),
	})
	must(err)
	must(vk.BindImageMemory(r.device, img, mem, 0))
	r.depth = gpuTexture{img: img, mem: mem, view: r.createView(img, r.depthFormat, vk.ImageAspectDepth)}
}

func (r *Renderer) destroySwapchain() {
	vk.DestroyImageView(r.device, r.depth.view)
	vk.DestroyImage(r.device, r.depth.img)
	vk.FreeMemory(r.device, r.depth.mem)
	for _, v := range r.swapViews {
		vk.DestroyImageView(r.device, v)
	}
}

func (r *Renderer) recreateSwapchain() {
	// Block while minimized.
	for {
		w, h := r.window.GetFramebufferSize()
		if w != 0 && h != 0 {
			break
		}
		glfw.WaitEvents()
	}
	must(vk.DeviceWaitIdle(r.device))

	old := r.swapchain
	r.destroySwapchain()
	r.createSwapchain(old)
	vk.DestroySwapchainKHR(r.device, old)
	r.needResize = false
}

// ---- scene: mesh, buffers, textures, descriptors -------------------------

func (r *Renderer) loadScene() {
	// Model: try Suzanne, else a procedural cube so the demo is self-contained.
	if m, err := obj.Load("models/suzanne.obj"); err == nil {
		r.mesh = m
	} else {
		r.mesh = cubeMesh()
	}

	vsize := uint64(len(r.mesh.Vertices)) * uint64(unsafe.Sizeof(obj.Vertex{}))
	r.vertexBuf = r.createMappedBuffer(vsize, vk.BufferUsageVertexBuffer, false)
	vk.MemCopy(r.vertexBuf.ptr, r.mesh.Vertices)

	isize := uint64(len(r.mesh.Indices)) * 4
	r.indexBuf = r.createMappedBuffer(isize, vk.BufferUsageIndexBuffer, false)
	vk.MemCopy(r.indexBuf.ptr, r.mesh.Indices)

	// Per-frame scene buffers reached by device address (BDA).
	for i := 0; i < framesInFlight; i++ {
		b := r.createMappedBuffer(uint64(unsafe.Sizeof(ShaderData{})),
			vk.BufferUsageStorageBuffer|vk.BufferUsageShaderDeviceAddress, true)
		r.scene[i] = b
		r.sceneAddr[i] = vk.GetBufferDeviceAddress(r.device, b.buf)
	}

	r.uploadTextures()
	r.buildDescriptors()
}

func (r *Renderer) uploadTextures() {
	// One solid-color texture per instance (no external assets required).
	colors := [instanceCount][3]byte{{220, 80, 80}, {80, 200, 120}, {90, 120, 230}}
	sampler, err := vk.CreateSampler(r.device, vk.SamplerCreateInfo{
		MagFilter: vk.FilterLinear, MinFilter: vk.FilterLinear,
		MipmapMode:   vk.SamplerMipmapModeLinear,
		AddressModeU: vk.SamplerAddressModeRepeat,
		AddressModeV: vk.SamplerAddressModeRepeat,
		AddressModeW: vk.SamplerAddressModeRepeat,
		MaxLod:       1,
	})
	must(err)
	r.sampler = sampler

	// One-time upload command buffer.
	oneShot, err := vk.AllocateCommandBuffers(r.device, r.cmdPool, 1)
	must(err)
	cb := oneShot[0]
	must(vk.BeginCommandBuffer(cb, vk.CommandBufferUsageOneTimeSubmit))

	var staging []mappedBuffer
	for i := 0; i < instanceCount; i++ {
		c := colors[i]
		src := texture.Solid(256, 256, c[0], c[1], c[2], 255)
		tex := r.createTextureImage(src.Width, src.Height)

		stage := r.createMappedBuffer(uint64(len(src.Pixels)), vk.BufferUsageTransferSrc, false)
		vk.MemCopy(stage.ptr, src.Pixels)
		staging = append(staging, stage)

		r.recordTextureUpload(cb, stage.buf, tex.img, uint32(src.Width), uint32(src.Height))
		r.textures = append(r.textures, tex)
	}

	must(vk.EndCommandBuffer(cb))
	must(vk.QueueSubmit2(r.queue, []vk.SubmitInfo2{{CommandBuffers: []vk.CommandBuffer{cb}}}, vk.Fence(0)))
	must(vk.QueueWaitIdle(r.queue))
	for _, s := range staging {
		s.destroy(r.device)
	}
}

func (r *Renderer) createTextureImage(w, h int) gpuTexture {
	img, err := vk.CreateImage(r.device, vk.ImageCreateInfo{
		ImageType: vk.ImageType2D,
		Format:    vk.FormatR8G8B8A8Unorm,
		Extent:    vk.Extent3D{Width: uint32(w), Height: uint32(h), Depth: 1},
		Usage:     vk.ImageUsageTransferDst | vk.ImageUsageSampled,
	})
	must(err)
	req := vk.GetImageMemoryRequirements(r.device, img)
	mem, err := vk.AllocateMemory(r.device, vk.MemoryAllocateInfo{
		AllocationSize: req.Size, MemoryTypeIndex: r.deviceLocalType(req.MemoryTypeBits),
	})
	must(err)
	must(vk.BindImageMemory(r.device, img, mem, 0))
	return gpuTexture{img: img, mem: mem, view: r.createView(img, vk.FormatR8G8B8A8Unorm, vk.ImageAspectColor)}
}

func (r *Renderer) recordTextureUpload(cb vk.CommandBuffer, staging vk.Buffer, img vk.Image, w, h uint32) {
	full := vk.ImageSubresourceRange{AspectMask: vk.ImageAspectColor, LevelCount: 1, LayerCount: 1}

	vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{{
		SrcStageMask: vk.PipelineStage2TopOfPipe, SrcAccessMask: vk.Access2None,
		DstStageMask: vk.PipelineStage2Copy, DstAccessMask: vk.Access2TransferWrite,
		OldLayout: vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutTransferDstOptimal,
		SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		Image: img, SubresourceRange: full,
	}})

	vk.CmdCopyBufferToImage(cb, staging, img, vk.ImageLayoutTransferDstOptimal, []vk.BufferImageCopy{{
		AspectMask: vk.ImageAspectColor, LayerCount: 1,
		ImageExtent: vk.Extent3D{Width: w, Height: h, Depth: 1},
	}})

	vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{{
		SrcStageMask: vk.PipelineStage2Copy, SrcAccessMask: vk.Access2TransferWrite,
		DstStageMask: vk.PipelineStage2FragmentShader, DstAccessMask: vk.Access2ShaderSampledRead,
		OldLayout: vk.ImageLayoutTransferDstOptimal, NewLayout: vk.ImageLayoutShaderReadOnlyOptimal,
		SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		Image: img, SubresourceRange: full,
	}})
}

func (r *Renderer) buildDescriptors() {
	layout, err := vk.CreateDescriptorSetLayout(r.device, vk.DescriptorSetLayoutCreateInfo{
		Flags:           vk.DescriptorSetLayoutCreateUpdateAfterBindPool,
		UseBindingFlags: true,
		Bindings: []vk.DescriptorSetLayoutBinding{{
			Binding:         0,
			DescriptorType:  vk.DescriptorTypeCombinedImageSampler,
			DescriptorCount: maxTextures,
			StageFlags:      vk.ShaderStageFragment,
			BindingFlags: vk.DescriptorBindingPartiallyBound |
				vk.DescriptorBindingVariableDescriptorCount |
				vk.DescriptorBindingUpdateAfterBind,
		}},
	})
	must(err)
	r.descLayout = layout

	pool, err := vk.CreateDescriptorPool(r.device, vk.DescriptorPoolCreateInfo{
		Flags:     vk.DescriptorPoolCreateUpdateAfterBind | vk.DescriptorPoolCreateFreeDescriptorSet,
		MaxSets:   1,
		PoolSizes: []vk.DescriptorPoolSize{{Type: vk.DescriptorTypeCombinedImageSampler, DescriptorCount: maxTextures}},
	})
	must(err)
	r.descPool = pool

	sets, err := vk.AllocateDescriptorSets(r.device, vk.DescriptorSetAllocateInfo{
		Pool:           pool,
		Layouts:        []vk.DescriptorSetLayout{layout},
		VariableCounts: []uint32{instanceCount},
	})
	must(err)
	r.descSet = sets[0]

	infos := make([]vk.DescriptorImageInfo, len(r.textures))
	for i, t := range r.textures {
		infos[i] = vk.DescriptorImageInfo{
			Sampler: r.sampler, ImageView: t.view, ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
		}
	}
	vk.UpdateDescriptorSets(r.device, []vk.WriteDescriptorSet{{
		DstSet: r.descSet, DstBinding: 0, DescriptorType: vk.DescriptorTypeCombinedImageSampler, ImageInfo: infos,
	}})
}

// ---- pipeline ------------------------------------------------------------

func (r *Renderer) buildPipeline() {
	var err error
	r.vertMod, err = vk.CreateShaderModule(r.device, shaders.Vert)
	must(err)
	r.fragMod, err = vk.CreateShaderModule(r.device, shaders.Frag)
	must(err)

	r.pipeLayout, err = vk.CreatePipelineLayout(r.device, vk.PipelineLayoutCreateInfo{
		SetLayouts:         []vk.DescriptorSetLayout{r.descLayout},
		PushConstantRanges: []vk.PushConstantRange{{StageFlags: vk.ShaderStageVertex, Offset: 0, Size: 8}},
	})
	must(err)

	stride := uint32(unsafe.Sizeof(obj.Vertex{}))
	r.pipeline, err = vk.CreateGraphicsPipeline(r.device, vk.GraphicsPipelineCreateInfo{
		Layout: r.pipeLayout,
		Stages: []vk.PipelineShaderStageCreateInfo{
			{Stage: vk.ShaderStageVertex, Module: r.vertMod, Name: "main"},
			{Stage: vk.ShaderStageFragment, Module: r.fragMod, Name: "main"},
		},
		VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
			Bindings: []vk.VertexInputBinding{{Binding: 0, Stride: stride, InputRate: vk.VertexInputRateVertex}},
			Attributes: []vk.VertexInputAttribute{
				{Location: 0, Binding: 0, Format: vk.FormatR32G32B32Sfloat, Offset: 0},
				{Location: 1, Binding: 0, Format: vk.FormatR32G32B32Sfloat, Offset: 12},
				{Location: 2, Binding: 0, Format: vk.FormatR32G32Sfloat, Offset: 24},
			},
		},
		InputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
			Topology: vk.PrimitiveTopologyTriangleList,
		},
		ViewportState: &vk.PipelineViewportStateCreateInfo{
			ViewportCount: 1,
			ScissorCount:  1,
		},
		RasterizationState: &vk.PipelineRasterizationStateCreateInfo{
			PolygonMode: vk.PolygonModeFill,
			CullMode:    vk.CullModeBack,
			FrontFace:   vk.FrontFaceCounterClockwise,
			LineWidth:   1.0,
		},
		MultisampleState: &vk.PipelineMultisampleStateCreateInfo{
			RasterizationSamples: vk.SampleCount1Bit,
		},
		DepthStencilState: &vk.PipelineDepthStencilStateCreateInfo{
			DepthTestEnable:  true,
			DepthWriteEnable: true,
			DepthCompareOp:   vk.CompareOpLessOrEqual,
			MaxDepthBounds:   1.0,
		},
		ColorBlendState: &vk.PipelineColorBlendStateCreateInfo{
			Attachments: []vk.PipelineColorBlendAttachmentState{
				{ColorWriteMask: vk.ColorComponentR | vk.ColorComponentG | vk.ColorComponentB | vk.ColorComponentA},
			},
		},
		DynamicState: &vk.PipelineDynamicStateCreateInfo{
			DynamicStates: []vk.DynamicState{vk.DynamicStateViewport, vk.DynamicStateScissor},
		},
		Rendering: &vk.PipelineRenderingCreateInfo{
			ColorAttachmentFormats: []vk.Format{r.surfaceFormat.Format},
			DepthAttachmentFormat:  r.depthFormat,
		},
	})
	must(err)
}

// ---- render loop ---------------------------------------------------------

func (r *Renderer) renderLoop() {
	for !r.window.ShouldClose() {
		glfw.PollEvents()
		r.drawFrame()
	}
	must(vk.DeviceWaitIdle(r.device))
}

func (r *Renderer) drawFrame() {
	fence := r.frameFences[r.frame]
	must(vk.WaitForFences(r.device, []vk.Fence{fence}, true, math.MaxUint64))

	imageIndex, err := vk.AcquireNextImageKHR(r.device, r.swapchain, math.MaxUint64, r.acquireSems[r.frame], vk.Fence(0))
	if err == vk.ErrOutOfDateKHR {
		r.recreateSwapchain()
		return
	} else if err != nil && err != vk.SuboptimalKHR {
		must(err)
	}

	must(vk.ResetFences(r.device, []vk.Fence{fence}))
	r.updateScene(r.frame)

	cb := r.cmdBufs[r.frame]
	must(vk.ResetCommandBuffer(cb))
	must(vk.BeginCommandBuffer(cb, vk.CommandBufferUsageOneTimeSubmit))
	r.recordFrame(cb, int(imageIndex))
	must(vk.EndCommandBuffer(cb))

	must(vk.QueueSubmit2(r.queue, []vk.SubmitInfo2{{
		WaitSemaphores: []vk.SemaphoreSubmitInfo{{
			Semaphore: r.acquireSems[r.frame], StageMask: vk.PipelineStage2ColorAttachmentOutput,
		}},
		CommandBuffers: []vk.CommandBuffer{cb},
		SignalSemaphores: []vk.SemaphoreSubmitInfo{{
			Semaphore: r.renderSems[imageIndex], StageMask: vk.PipelineStage2AllCommands,
		}},
	}}, fence))

	err = vk.QueuePresentKHR(r.queue, r.renderSems[imageIndex], r.swapchain, imageIndex)
	if err == vk.ErrOutOfDateKHR || err == vk.SuboptimalKHR || r.needResize {
		r.recreateSwapchain()
	} else if err != nil {
		must(err)
	}

	r.frame = (r.frame + 1) % framesInFlight
}

func (r *Renderer) recordFrame(cb vk.CommandBuffer, imageIndex int) {
	colorRange := vk.ImageSubresourceRange{AspectMask: vk.ImageAspectColor, LevelCount: 1, LayerCount: 1}
	depthRange := vk.ImageSubresourceRange{AspectMask: vk.ImageAspectDepth, LevelCount: 1, LayerCount: 1}

	// Transition swapchain image -> color attachment, depth -> depth attachment.
	vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{
		{
			SrcStageMask: vk.PipelineStage2TopOfPipe, SrcAccessMask: vk.Access2None,
			DstStageMask: vk.PipelineStage2ColorAttachmentOutput, DstAccessMask: vk.Access2ColorAttachmentWrite,
			OldLayout: vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutColorAttachmentOptimal,
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
			Image: r.swapImages[imageIndex], SubresourceRange: colorRange,
		},
		{
			SrcStageMask: vk.PipelineStage2TopOfPipe, SrcAccessMask: vk.Access2None,
			DstStageMask:  vk.PipelineStage2EarlyFragmentTests | vk.PipelineStage2LateFragmentTests,
			DstAccessMask: vk.Access2DepthStencilAttachmentWrite,
			OldLayout:     vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutDepthAttachmentOptimal,
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
			Image: r.depth.img, SubresourceRange: depthRange,
		},
	})

	area := vk.Rect2D{Extent: r.swapExtent}
	vk.CmdBeginRendering(cb, vk.RenderingInfo{
		RenderArea: area,
		LayerCount: 1,
		ColorAttachments: []vk.RenderingAttachmentInfo{{
			ImageView:   r.swapViews[imageIndex],
			ImageLayout: vk.ImageLayoutColorAttachmentOptimal,
			LoadOp:      vk.AttachmentLoadOpClear,
			StoreOp:     vk.AttachmentStoreOpStore,
			ClearValue:  vk.ClearColor(0.02, 0.02, 0.05, 1),
		}},
		DepthAttachment: &vk.RenderingAttachmentInfo{
			ImageView:   r.depth.view,
			ImageLayout: vk.ImageLayoutDepthAttachmentOptimal,
			LoadOp:      vk.AttachmentLoadOpClear,
			StoreOp:     vk.AttachmentStoreOpDontCare,
			ClearValue:  vk.ClearDepthStencil(1, 0),
		},
	})

	vk.CmdSetViewport(cb, vk.Viewport{
		Width: float32(r.swapExtent.Width), Height: float32(r.swapExtent.Height), MaxDepth: 1,
	})
	vk.CmdSetScissor(cb, area)
	vk.CmdBindPipeline(cb, vk.PipelineBindPointGraphics, r.pipeline)
	vk.CmdBindDescriptorSets(cb, vk.PipelineBindPointGraphics, r.pipeLayout, 0, []vk.DescriptorSet{r.descSet})
	vk.CmdBindVertexBuffer(cb, 0, r.vertexBuf.buf, 0)
	vk.CmdBindIndexBuffer(cb, r.indexBuf.buf, 0, vk.IndexTypeUint32)

	addr := r.sceneAddr[r.frame]
	vk.CmdPushConstants(cb, r.pipeLayout, vk.ShaderStageVertex, 0, 8, unsafe.Pointer(&addr))
	vk.CmdDrawIndexed(cb, uint32(len(r.mesh.Indices)), instanceCount, 0, 0, 0)

	vk.CmdEndRendering(cb)

	// Transition swapchain image -> present.
	vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{{
		SrcStageMask: vk.PipelineStage2ColorAttachmentOutput, SrcAccessMask: vk.Access2ColorAttachmentWrite,
		DstStageMask: vk.PipelineStage2BottomOfPipe, DstAccessMask: vk.Access2None,
		OldLayout: vk.ImageLayoutColorAttachmentOptimal, NewLayout: vk.ImageLayoutPresentSrcKHR,
		SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		Image: r.swapImages[imageIndex], SubresourceRange: colorRange,
	}})
}

func (r *Renderer) updateScene(frame int) {
	aspect := float32(r.swapExtent.Width) / float32(r.swapExtent.Height)
	if aspect == 0 || math.IsInf(float64(aspect), 0) {
		aspect = 1
	}
	// GL -> Vulkan clip correction (flip Y, remap Z to 0..1).
	clip := mgl32.Mat4{
		1, 0, 0, 0,
		0, -1, 0, 0,
		0, 0, 0.5, 0,
		0, 0, 0.5, 1,
	}
	proj := clip.Mul4(mgl32.Perspective(mgl32.DegToRad(60), aspect, 0.1, 100))

	eye := mgl32.Vec3{
		r.zoom * float32(math.Cos(float64(r.pitch))) * float32(math.Sin(float64(r.yaw))),
		r.zoom * float32(math.Sin(float64(r.pitch))),
		r.zoom * float32(math.Cos(float64(r.pitch))) * float32(math.Cos(float64(r.yaw))),
	}
	view := mgl32.LookAtV(eye, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})

	var data ShaderData
	data.Projection = proj
	data.View = view
	for i := 0; i < instanceCount; i++ {
		x := float32(i-1) * 2.5
		model := mgl32.Translate3D(x, 0, 0)
		data.Model[i] = model
	}
	data.LightPos = [4]float32{4, 4, 4, 1}
	data.Selected = 0

	*(*ShaderData)(r.scene[frame].ptr) = data
}

// ---- cleanup -------------------------------------------------------------

func (r *Renderer) cleanup() {
	vk.DeviceWaitIdle(r.device)

	vk.DestroyPipeline(r.device, r.pipeline)
	vk.DestroyPipelineLayout(r.device, r.pipeLayout)
	vk.DestroyShaderModule(r.device, r.vertMod)
	vk.DestroyShaderModule(r.device, r.fragMod)

	vk.DestroyDescriptorPool(r.device, r.descPool)
	vk.DestroyDescriptorSetLayout(r.device, r.descLayout)
	vk.DestroySampler(r.device, r.sampler)
	for _, t := range r.textures {
		vk.DestroyImageView(r.device, t.view)
		vk.DestroyImage(r.device, t.img)
		vk.FreeMemory(r.device, t.mem)
	}

	for i := 0; i < framesInFlight; i++ {
		r.scene[i].destroy(r.device)
	}
	r.vertexBuf.destroy(r.device)
	r.indexBuf.destroy(r.device)

	for _, s := range r.renderSems {
		vk.DestroySemaphore(r.device, s)
	}
	for _, s := range r.acquireSems {
		vk.DestroySemaphore(r.device, s)
	}
	for _, f := range r.frameFences {
		vk.DestroyFence(r.device, f)
	}
	vk.DestroyCommandPool(r.device, r.cmdPool)

	r.destroySwapchain()
	vk.DestroySwapchainKHR(r.device, r.swapchain)
	vk.DestroyDevice(r.device)
	vk.DestroySurfaceKHR(r.instance, r.surface)
	vk.DestroyInstance(r.instance)
}

// cubeMesh builds a unit cube with per-face normals and UVs as a fallback when
// no OBJ asset is present.
func cubeMesh() *obj.Mesh {
	// 6 faces, each: 4 verts + 2 tris.
	type face struct {
		normal     [3]float32
		a, b, c, d [3]float32
	}
	faces := []face{
		{[3]float32{0, 0, 1}, [3]float32{-1, -1, 1}, [3]float32{1, -1, 1}, [3]float32{1, 1, 1}, [3]float32{-1, 1, 1}},
		{[3]float32{0, 0, -1}, [3]float32{1, -1, -1}, [3]float32{-1, -1, -1}, [3]float32{-1, 1, -1}, [3]float32{1, 1, -1}},
		{[3]float32{1, 0, 0}, [3]float32{1, -1, 1}, [3]float32{1, -1, -1}, [3]float32{1, 1, -1}, [3]float32{1, 1, 1}},
		{[3]float32{-1, 0, 0}, [3]float32{-1, -1, -1}, [3]float32{-1, -1, 1}, [3]float32{-1, 1, 1}, [3]float32{-1, 1, -1}},
		{[3]float32{0, 1, 0}, [3]float32{-1, 1, 1}, [3]float32{1, 1, 1}, [3]float32{1, 1, -1}, [3]float32{-1, 1, -1}},
		{[3]float32{0, -1, 0}, [3]float32{-1, -1, -1}, [3]float32{1, -1, -1}, [3]float32{1, -1, 1}, [3]float32{-1, -1, 1}},
	}
	m := &obj.Mesh{}
	uv := [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}
	for _, f := range faces {
		base := uint32(len(m.Vertices))
		for k, p := range [4][3]float32{f.a, f.b, f.c, f.d} {
			m.Vertices = append(m.Vertices, obj.Vertex{Pos: p, Normal: f.normal, UV: uv[k]})
		}
		m.Indices = append(m.Indices, base, base+1, base+2, base, base+2, base+3)
	}
	return m
}
