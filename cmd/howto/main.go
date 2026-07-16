// Command howto is a close Go port of the single-file reference renderer from
// howtovulkan.com (Sascha Willems, MIT-licensed). It deliberately keeps the
// reference's flat shape: package-level globals plus one long main() that walks
// the same sections in the same order, calling the hand-written vk bindings.
//
// The reference depends on several C++-only libraries; each is swapped for its
// Go equivalent. These swaps change a handful of lines but NOT the Vulkan call
// sequence, which is what the tutorial is really about:
//
//	SDL3            -> GLFW                         (window + surface + input)
//	VMA             -> vkx.CreateBuffer/CreateImage (raw allocate + bind + map)
//	Slang (runtime) -> pre-compiled embedded SPIR-V (shaders.Vert / shaders.Frag)
//	KTX textures    -> texture.Solid               (one solid color per instance)
//	tinyobjloader   -> internal/obj                (with a cube fallback)
//	GLM             -> mathgl                       (matrix math)
//
// The Go-specific scaffolding (the VMA-substitute allocators, the GLFW->vk
// surface bridge, the cube fallback) lives in the imported internal/vkx package
// so this file stays a readable mirror of the original.
package main

import (
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"govk/internal/obj"
	"govk/internal/texture"
	"govk/internal/vkx"
	"govk/shaders"
	"govk/vk"
)

// maxFramesInFlight is how many frames the CPU may record ahead of the GPU. Two
// gives double buffering: while the GPU draws frame N the CPU records frame N+1.
const maxFramesInFlight = 2

// ShaderData mirrors the reference's ShaderData struct byte-for-byte (scalar
// layout). The vertex shader reads it through a buffer device address passed as
// a push constant, so the CPU and GPU must agree on the layout exactly.
type ShaderData struct {
	Projection [16]float32    // camera projection matrix
	View       [16]float32    // camera view matrix
	Model      [3][16]float32 // per-instance model matrix (three Suzannes)
	LightPos   [4]float32     // world-space light position
	Selected   uint32         // which instance the mouse currently rotates
}

// Texture bundles a sampled image, its memory, view, and sampler. Equivalent to
// the reference's Texture struct (VmaAllocation replaced by vk.DeviceMemory).
type Texture struct {
	image   vk.Image
	memory  vk.DeviceMemory
	view    vk.ImageView
	sampler vk.Sampler
}

// ---- globals (matching the reference's file-scope state) -----------------

var (
	frameIndex      int    // ping-pongs 0..maxFramesInFlight-1
	imageIndex      uint32 // swapchain image acquired this frame
	instance        vk.Instance
	physicalDevice  vk.PhysicalDevice
	device          vk.Device
	queue           vk.Queue
	queueFamily     uint32
	memProps        vk.PhysicalDeviceMemoryProperties
	surface         vk.SurfaceKHR
	updateSwapchain bool // set when the surface is out of date / resized
	swapchain       vk.SwapchainKHR
	commandPool     vk.CommandPool
	pipeline        vk.Pipeline
	pipelineLayout  vk.PipelineLayout
	depthFormat     vk.Format
	depthImage      vk.Image
	depthImageMem   vk.DeviceMemory
	depthImageView  vk.ImageView

	swapchainImages          []vk.Image
	swapchainImageViews      []vk.ImageView
	commandBuffers           []vk.CommandBuffer
	fences                   [maxFramesInFlight]vk.Fence     // one per in-flight frame
	imageAcquiredSemaphores  [maxFramesInFlight]vk.Semaphore // signalled by AcquireNextImage
	renderCompleteSemaphores []vk.Semaphore                  // one per swapchain image, waited on by present

	vBuffer  vkx.AllocBuffer // interleaved vertices followed by indices in one buffer
	vBufSize uint64          // byte size of the vertex region (= index buffer offset)
	indexCnt uint32          // number of indices to draw

	shaderData        ShaderData
	shaderDataBuffers [maxFramesInFlight]vkx.AllocBuffer // per-frame uniform data
	shaderDataAddrs   [maxFramesInFlight]uint64          // their device addresses

	textures               [3]Texture
	descriptorPool         vk.DescriptorPool
	descriptorSetLayoutTex vk.DescriptorSetLayout
	descriptorSetTex       vk.DescriptorSet

	window     *glfw.Window
	windowSize [2]int

	// Camera/scene state driven by input, matching the reference.
	camPos          = mgl32.Vec3{0, 0, -6}
	objectRotations [3]mgl32.Vec3

	// The reference hardcodes an sRGB BGRA swapchain format + colorspace.
	imageFormat = vk.FormatB8G8R8A8Srgb
)

// init locks main() to one OS thread. Vulkan command submission and the windowing
// system both expect a stable thread; Go may otherwise move a goroutine between
// threads at any scheduling point.
func init() { runtime.LockOSThread() }

// chk aborts on a Vulkan error, like the reference's chk(). Fatal-on-error keeps
// the linear tutorial code readable (no error plumbing at every call site).
func chk(err error) {
	if err != nil {
		log.Fatalf("Vulkan call returned an error: %v", err)
	}
}

// chkSwapchain treats OUT_OF_DATE / SUBOPTIMAL as "recreate the swapchain" rather
// than fatal, mirroring the reference's chkSwapchain().
func chkSwapchain(err error) {
	if err == vk.ErrOutOfDateKHR || err == vk.SuboptimalKHR {
		updateSwapchain = true
		return
	}
	chk(err)
}

func main() {
	// --- Window -----------------------------------------------------------
	// GLFW must create the window before the instance, because the list of
	// required instance extensions is queried from the window (SDL can query it
	// without one). ClientAPI=NoAPI tells GLFW not to create an OpenGL context.
	chk(glfw.Init())
	if !glfw.VulkanSupported() {
		log.Fatal("GLFW: Vulkan not supported")
	}
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	var err error
	window, err = glfw.CreateWindow(1280, 720, "How to Vulkan", nil, nil)
	chk(err)
	windowSize[0], windowSize[1] = window.GetSize()

	// --- Instance ---------------------------------------------------------
	// The instance is the connection to the Vulkan loader. It needs the
	// surface-related extensions the windowing system requires.
	instance, err = vk.CreateInstance(vk.InstanceCreateInfo{
		AppName:    "How to Vulkan",
		APIVersion: vk.ApiVersion13,
		Extensions: window.GetRequiredInstanceExtensions(),
	})
	chk(err)

	// --- Physical device + queue family -----------------------------------
	devices, err := vk.EnumeratePhysicalDevices(instance)
	chk(err)
	deviceIndex := 0
	if len(os.Args) > 1 { // optional GPU index argument, like the reference
		deviceIndex, _ = strconv.Atoi(os.Args[1])
	}
	physicalDevice = devices[deviceIndex]
	log.Printf("Selected device: %s", vk.GetPhysicalDeviceProperties2(physicalDevice).DeviceName)

	// Pick the first queue family that supports graphics. On desktop GPUs this
	// same family also supports present (verified below after surface creation).
	for i, qf := range vk.GetPhysicalDeviceQueueFamilyProperties(physicalDevice) {
		if qf.QueueFlags&vk.QueueGraphics != 0 {
			queueFamily = uint32(i)
			break
		}
	}

	// --- Logical device ---------------------------------------------------
	// Enable exactly the 1.2/1.3 features this renderer uses: descriptor
	// indexing (bindless texture array), buffer device address (pointer to the
	// uniform buffer), synchronization2 (the *2 barriers/submit), and dynamic
	// rendering (no VkRenderPass object).
	device, err = vk.CreateDevice(physicalDevice, vk.DeviceCreateInfo{
		QueueCreateInfos: []vk.DeviceQueueCreateInfo{{QueueFamilyIndex: queueFamily, Priorities: []float32{1}}},
		Extensions:       []string{"VK_KHR_swapchain"},
		Features: vk.Features{
			DescriptorIndexing:                        true,
			ShaderSampledImageArrayNonUniformIndexing: true,
			DescriptorBindingVariableDescriptorCount:  true,
			RuntimeDescriptorArray:                    true,
			BufferDeviceAddress:                       true,
			SamplerAnisotropy:                         true,
			Synchronization2:                          true,
			DynamicRendering:                          true,
		},
	})
	chk(err)
	queue = vk.GetDeviceQueue(device, queueFamily, 0)
	memProps = vk.GetPhysicalDeviceMemoryProperties2(physicalDevice)

	// --- Surface ----------------------------------------------------------
	surface = vkx.CreateSurface(instance, window)
	presentOK, err := vk.GetPhysicalDeviceSurfaceSupportKHR(physicalDevice, queueFamily, surface)
	chk(err)
	if !presentOK {
		log.Fatal("selected queue family cannot present to the surface")
	}
	surfaceCaps, err := vk.GetPhysicalDeviceSurfaceCapabilitiesKHR(physicalDevice, surface)
	chk(err)
	// A currentExtent of 0xFFFFFFFF means "surface size is defined by the
	// swapchain", so fall back to the window's own size.
	swapchainExtent := surfaceCaps.CurrentExtent
	if surfaceCaps.CurrentExtent.Width == 0xFFFFFFFF {
		swapchainExtent = vk.Extent2D{Width: uint32(windowSize[0]), Height: uint32(windowSize[1])}
	}

	// --- Swapchain --------------------------------------------------------
	// The swapchain owns the images shown on screen. swapchainCI is kept around
	// because swapchain recreation on resize reuses it with a new extent.
	swapchainCI := vk.SwapchainCreateInfo{
		Surface:         surface,
		MinImageCount:   surfaceCaps.MinImageCount,
		ImageFormat:     imageFormat,
		ImageColorSpace: vk.ColorSpaceSrgbNonlinearKHR,
		ImageExtent:     swapchainExtent,
		ImageUsage:      vk.ImageUsageColorAttachment,
		PreTransform:    vk.SurfaceTransformIdentityKHR,
		CompositeAlpha:  vk.CompositeAlphaOpaqueKHR,
		PresentMode:     vk.PresentModeFifoKHR, // FIFO = vsync, always supported
	}
	swapchain, err = vk.CreateSwapchainKHR(device, swapchainCI)
	chk(err)
	swapchainImages, err = vk.GetSwapchainImagesKHR(device, swapchain)
	chk(err)
	// One image view per swapchain image, used as the color attachment.
	swapchainImageViews = make([]vk.ImageView, len(swapchainImages))
	for i := range swapchainImages {
		swapchainImageViews[i] = createImageView(swapchainImages[i], imageFormat, vk.ImageAspectColor)
	}

	// --- Depth attachment -------------------------------------------------
	// Probe the GPU for a supported depth/stencil format at runtime (the one
	// place the reference needs vkGetPhysicalDeviceFormatProperties2).
	depthFormat = vk.FormatUndefined
	for _, format := range []vk.Format{vk.FormatD32SfloatS8Uint, vk.FormatD24UnormS8Uint} {
		fp := vk.GetPhysicalDeviceFormatProperties2(physicalDevice, format)
		if fp.OptimalTilingFeatures&vk.FormatFeatureDepthStencilAttachment != 0 {
			depthFormat = format
			break
		}
	}
	if depthFormat == vk.FormatUndefined {
		log.Fatal("no supported depth/stencil format")
	}
	// vkx.CreateImage supplies the fixed defaults the reference sets explicitly
	// (samples=1, tiling=OPTIMAL, initialLayout=UNDEFINED, mip/array=1).
	depthImage, depthImageMem = vkx.CreateImage(device, memProps, vk.ImageCreateInfo{
		ImageType: vk.ImageType2D,
		Format:    depthFormat,
		Extent:    vk.Extent3D{Width: uint32(windowSize[0]), Height: uint32(windowSize[1]), Depth: 1},
		Usage:     vk.ImageUsageDepthStencilAttachment,
	})
	depthImageView = createImageView(depthImage, depthFormat, vk.ImageAspectDepth)

	// --- Mesh data --------------------------------------------------------
	// Load Suzanne, or fall back to a procedural cube so the demo needs no
	// external assets. Vertices and indices are packed into ONE buffer (vertices
	// first, indices after), exactly like the reference.
	mesh, err := obj.Load("assets/suzanne.obj")
	if err != nil {
		log.Printf("mesh load failed (%v); using procedural cube", err)
		mesh = vkx.CubeMesh()
	}
	indexCnt = uint32(len(mesh.Indices))
	// The reference uses 16-bit indices; convert (safe while unique vertex count
	// stays below 65536, true for Suzanne and the cube).
	indices16 := make([]uint16, len(mesh.Indices))
	for i, v := range mesh.Indices {
		indices16[i] = uint16(v)
	}
	vBufSize = uint64(len(mesh.Vertices)) * uint64(unsafe.Sizeof(obj.Vertex{}))
	iBufSize := uint64(len(indices16)) * 2
	vBuffer = vkx.CreateBuffer(device, memProps,
		vk.BufferCreateInfo{Size: vBufSize + iBufSize, Usage: vk.BufferUsageVertexBuffer | vk.BufferUsageIndexBuffer},
		true, false)
	vk.MemCopy(vBuffer.Mapped, mesh.Vertices)                   // vertices at offset 0
	vk.MemCopy(unsafe.Add(vBuffer.Mapped, vBufSize), indices16) // indices right after

	// --- Shader data buffers ----------------------------------------------
	// One uniform buffer per in-flight frame, each reached in the shader by its
	// device address (buffer_device_address), passed as a push constant.
	for i := 0; i < maxFramesInFlight; i++ {
		shaderDataBuffers[i] = vkx.CreateBuffer(device, memProps,
			vk.BufferCreateInfo{Size: uint64(unsafe.Sizeof(ShaderData{})),
				Usage: vk.BufferUsageStorageBuffer | vk.BufferUsageShaderDeviceAddress},
			true, true)
		shaderDataAddrs[i] = vk.GetBufferDeviceAddress(device, shaderDataBuffers[i].Buffer)
	}

	// --- Sync objects -----------------------------------------------------
	// Fences let the CPU wait for a frame's GPU work to finish; the acquire
	// semaphores order acquire->render on the GPU. Render-complete semaphores are
	// per swapchain image (present waits on them), created after the swapchain.
	for i := 0; i < maxFramesInFlight; i++ {
		fences[i], err = vk.CreateFence(device, vk.FenceCreateSignaled) // signalled so frame 0 doesn't block
		chk(err)
		imageAcquiredSemaphores[i], err = vk.CreateSemaphore(device)
		chk(err)
	}
	renderCompleteSemaphores = make([]vk.Semaphore, len(swapchainImages))
	for i := range renderCompleteSemaphores {
		renderCompleteSemaphores[i], err = vk.CreateSemaphore(device)
		chk(err)
	}

	// --- Command pool -----------------------------------------------------
	commandPool, err = vk.CreateCommandPool(device, queueFamily, vk.CommandPoolCreateResetCommandBuffer)
	chk(err)
	commandBuffers, err = vk.AllocateCommandBuffers(device, commandPool, maxFramesInFlight)
	chk(err)

	// --- Texture images ---------------------------------------------------
	// One solid-color texture per instance (the reference loads three mipmapped
	// KTX files). Each is created, then uploaded via a staging buffer and a
	// one-time command buffer with two layout transitions.
	colors := [3][3]byte{{220, 80, 80}, {80, 200, 120}, {90, 120, 230}}
	var textureDescriptors []vk.DescriptorImageInfo
	for i := 0; i < len(textures); i++ {
		src := texture.Solid(256, 256, colors[i][0], colors[i][1], colors[i][2], 255)
		textures[i].image, textures[i].memory = vkx.CreateImage(device, memProps, vk.ImageCreateInfo{
			ImageType: vk.ImageType2D,
			Format:    vk.FormatR8G8B8A8Unorm,
			Extent:    vk.Extent3D{Width: uint32(src.Width), Height: uint32(src.Height), Depth: 1},
			Usage:     vk.ImageUsageTransferDst | vk.ImageUsageSampled,
		})
		textures[i].view = createImageView(textures[i].image, vk.FormatR8G8B8A8Unorm, vk.ImageAspectColor)

		// Staging buffer holds the pixels on the host side for the copy.
		imgSrc := vkx.CreateBuffer(device, memProps,
			vk.BufferCreateInfo{Size: uint64(len(src.Pixels)), Usage: vk.BufferUsageTransferSrc}, true, false)
		vk.MemCopy(imgSrc.Mapped, src.Pixels)

		// Record and run a one-time upload: UNDEFINED -> TRANSFER_DST, copy,
		// TRANSFER_DST -> SHADER_READ_ONLY. Wait on a fence, then clean up.
		fenceOneTime, err := vk.CreateFence(device, 0)
		chk(err)
		oneTime, err := vk.AllocateCommandBuffers(device, commandPool, 1)
		chk(err)
		cb := oneTime[0]
		chk(vk.BeginCommandBuffer(cb, vk.CommandBufferUsageOneTimeSubmit))
		full := vk.ImageSubresourceRange{AspectMask: vk.ImageAspectColor, LevelCount: 1, LayerCount: 1}
		vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{{
			SrcStageMask: vk.PipelineStage2None, SrcAccessMask: vk.Access2None,
			DstStageMask: vk.PipelineStage2Transfer, DstAccessMask: vk.Access2TransferWrite,
			OldLayout: vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutTransferDstOptimal,
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
			Image: textures[i].image, SubresourceRange: full,
		}})
		vk.CmdCopyBufferToImage(cb, imgSrc.Buffer, textures[i].image, vk.ImageLayoutTransferDstOptimal,
			[]vk.BufferImageCopy{{
				AspectMask: vk.ImageAspectColor, LayerCount: 1,
				ImageExtent: vk.Extent3D{Width: uint32(src.Width), Height: uint32(src.Height), Depth: 1},
			}})
		vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{{
			SrcStageMask: vk.PipelineStage2Transfer, SrcAccessMask: vk.Access2TransferWrite,
			DstStageMask: vk.PipelineStage2FragmentShader, DstAccessMask: vk.Access2ShaderRead,
			OldLayout: vk.ImageLayoutTransferDstOptimal, NewLayout: vk.ImageLayoutShaderReadOnlyOptimal,
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
			Image: textures[i].image, SubresourceRange: full,
		}})
		chk(vk.EndCommandBuffer(cb))
		// Reference uses vkQueueSubmit (v1); this binding is synchronization2, so
		// the equivalent one-batch submit goes through vkQueueSubmit2.
		chk(vk.QueueSubmit2(queue, []vk.SubmitInfo2{{CommandBuffers: []vk.CommandBuffer{cb}}}, fenceOneTime))
		chk(vk.WaitForFences(device, []vk.Fence{fenceOneTime}, true, math.MaxUint64))
		vk.DestroyFence(device, fenceOneTime)
		imgSrc.Destroy(device)

		// Sampler: linear filtering + anisotropy, one per texture.
		textures[i].sampler, err = vk.CreateSampler(device, vk.SamplerCreateInfo{
			MagFilter: vk.FilterLinear, MinFilter: vk.FilterLinear,
			MipmapMode: vk.SamplerMipmapModeLinear, AnisotropyEnable: true, MaxAnisotropy: 8, MaxLod: 1,
		})
		chk(err)
		textureDescriptors = append(textureDescriptors, vk.DescriptorImageInfo{
			Sampler: textures[i].sampler, ImageView: textures[i].view, ImageLayout: vk.ImageLayoutShaderReadOnlyOptimal,
		})
	}

	// --- Descriptor set (descriptor indexing) -----------------------------
	// A single binding holding a variable-length array of combined image
	// samplers. The variable-count binding flag lets the shader index the array
	// by instance (bindless), sized at allocation time to the texture count.
	descriptorSetLayoutTex, err = vk.CreateDescriptorSetLayout(device, vk.DescriptorSetLayoutCreateInfo{
		UseBindingFlags: true,
		Bindings: []vk.DescriptorSetLayoutBinding{{
			Binding: 0, DescriptorType: vk.DescriptorTypeCombinedImageSampler,
			DescriptorCount: uint32(len(textures)), StageFlags: vk.ShaderStageFragment,
			BindingFlags: vk.DescriptorBindingVariableDescriptorCount,
		}},
	})
	chk(err)
	descriptorPool, err = vk.CreateDescriptorPool(device, vk.DescriptorPoolCreateInfo{
		MaxSets:   1,
		PoolSizes: []vk.DescriptorPoolSize{{Type: vk.DescriptorTypeCombinedImageSampler, DescriptorCount: uint32(len(textures))}},
	})
	chk(err)
	sets, err := vk.AllocateDescriptorSets(device, vk.DescriptorSetAllocateInfo{
		Pool:           descriptorPool,
		Layouts:        []vk.DescriptorSetLayout{descriptorSetLayoutTex},
		VariableCounts: []uint32{uint32(len(textures))}, // final size of the variable-count binding
	})
	chk(err)
	descriptorSetTex = sets[0]
	// Write all texture descriptors into the set in one update.
	vk.UpdateDescriptorSets(device, []vk.WriteDescriptorSet{{
		DstSet: descriptorSetTex, DstBinding: 0,
		DescriptorType: vk.DescriptorTypeCombinedImageSampler, ImageInfo: textureDescriptors,
	}})

	// --- Shader modules ---------------------------------------------------
	// The reference compiles assets/shader.slang at runtime with Slang; there is
	// no Go Slang binding, so pre-compiled SPIR-V (built offline by glslc) is
	// embedded and loaded as two modules instead of one multi-entry module.
	vertModule, err := vk.CreateShaderModule(device, shaders.Vert)
	chk(err)
	fragModule, err := vk.CreateShaderModule(device, shaders.Frag)
	chk(err)

	// --- Pipeline ---------------------------------------------------------
	// The push constant carries the 8-byte device address of this frame's
	// uniform buffer to the vertex shader.
	pipelineLayout, err = vk.CreatePipelineLayout(device, vk.PipelineLayoutCreateInfo{
		SetLayouts:         []vk.DescriptorSetLayout{descriptorSetLayoutTex},
		PushConstantRanges: []vk.PushConstantRange{{StageFlags: vk.ShaderStageVertex, Size: 8}},
	})
	chk(err)
	stride := uint32(unsafe.Sizeof(obj.Vertex{}))
	pipeline, err = vk.CreateGraphicsPipeline(device, vk.GraphicsPipelineCreateInfo{
		Layout: pipelineLayout,
		Stages: []vk.PipelineShaderStageCreateInfo{
			{Stage: vk.ShaderStageVertex, Module: vertModule, Name: "main"},
			{Stage: vk.ShaderStageFragment, Module: fragModule, Name: "main"},
		},
		// Vertex layout: position (vec3) @0, normal (vec3) @12, uv (vec2) @24.
		VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
			Bindings: []vk.VertexInputBinding{{Binding: 0, Stride: stride, InputRate: vk.VertexInputRateVertex}},
			Attributes: []vk.VertexInputAttribute{
				{Location: 0, Binding: 0, Format: vk.FormatR32G32B32Sfloat, Offset: 0},
				{Location: 1, Binding: 0, Format: vk.FormatR32G32B32Sfloat, Offset: 12},
				{Location: 2, Binding: 0, Format: vk.FormatR32G32Sfloat, Offset: 24},
			},
		},
		InputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{Topology: vk.PrimitiveTopologyTriangleList},
		// Viewport/scissor are dynamic (set each frame); only the counts matter here.
		ViewportState:      &vk.PipelineViewportStateCreateInfo{ViewportCount: 1, ScissorCount: 1},
		RasterizationState: &vk.PipelineRasterizationStateCreateInfo{PolygonMode: vk.PolygonModeFill, LineWidth: 1.0},
		MultisampleState:   &vk.PipelineMultisampleStateCreateInfo{RasterizationSamples: vk.SampleCount1Bit},
		DepthStencilState: &vk.PipelineDepthStencilStateCreateInfo{
			DepthTestEnable: true, DepthWriteEnable: true, DepthCompareOp: vk.CompareOpLessOrEqual,
		},
		// Single color attachment, blending off, write all RGBA channels (0xF).
		ColorBlendState: &vk.PipelineColorBlendStateCreateInfo{
			Attachments: []vk.PipelineColorBlendAttachmentState{{ColorWriteMask: 0xF}},
		},
		DynamicState: &vk.PipelineDynamicStateCreateInfo{
			DynamicStates: []vk.DynamicState{vk.DynamicStateViewport, vk.DynamicStateScissor},
		},
		// Dynamic rendering: declare the attachment formats instead of a render pass.
		Rendering: &vk.PipelineRenderingCreateInfo{
			ColorAttachmentFormats: []vk.Format{imageFormat},
			DepthAttachmentFormat:  depthFormat,
		},
	})
	chk(err)

	// GLFW splits events into callbacks (scroll, resize) and polled state (mouse
	// buttons, keys), unlike SDL's single event queue; installInput wires the
	// callbacks and the loop polls the rest.
	installInput()

	// --- Render loop ------------------------------------------------------
	lastTime := glfw.GetTime()
	for !window.ShouldClose() {
		// Wait until this frame's slot is free, then reset its fence.
		chk(vk.WaitForFences(device, []vk.Fence{fences[frameIndex]}, true, math.MaxUint64))
		chk(vk.ResetFences(device, []vk.Fence{fences[frameIndex]}))
		// Acquire the next image to draw into; the semaphore is signalled when it's ready.
		imageIndex, err = vk.AcquireNextImageKHR(device, swapchain, math.MaxUint64, imageAcquiredSemaphores[frameIndex], vk.Fence(0))
		chkSwapchain(err)

		updateSceneData()

		// Record this frame's command buffer, then submit and present.
		recordCommandBuffer(commandBuffers[frameIndex])
		chk(vk.QueueSubmit2(queue, []vk.SubmitInfo2{{
			WaitSemaphores:   []vk.SemaphoreSubmitInfo{{Semaphore: imageAcquiredSemaphores[frameIndex], StageMask: vk.PipelineStage2ColorAttachmentOutput}},
			CommandBuffers:   []vk.CommandBuffer{commandBuffers[frameIndex]},
			SignalSemaphores: []vk.SemaphoreSubmitInfo{{Semaphore: renderCompleteSemaphores[imageIndex], StageMask: vk.PipelineStage2AllCommands}},
		}}, fences[frameIndex]))
		// Advance the frame slot before present (reference does the same).
		frameIndex = (frameIndex + 1) % maxFramesInFlight
		chkSwapchain(vk.QueuePresentKHR(queue, renderCompleteSemaphores[imageIndex], swapchain, imageIndex))

		// Input: poll mouse-drag + keys, tracking the frame delta for speed.
		elapsed := float32(glfw.GetTime() - lastTime)
		lastTime = glfw.GetTime()
		pollInput(elapsed)

		if updateSwapchain {
			recreateSwapchain(&swapchainCI)
		}
	}

	// --- Tear down --------------------------------------------------------
	// Wait for the GPU to go idle, then destroy everything in roughly reverse
	// creation order. Every create above has its matching destroy here.
	chk(vk.DeviceWaitIdle(device))
	for i := 0; i < maxFramesInFlight; i++ {
		vk.DestroyFence(device, fences[i])
		vk.DestroySemaphore(device, imageAcquiredSemaphores[i])
		shaderDataBuffers[i].Destroy(device)
	}
	for _, s := range renderCompleteSemaphores {
		vk.DestroySemaphore(device, s)
	}
	vk.DestroyImageView(device, depthImageView)
	vk.DestroyImage(device, depthImage)
	vk.FreeMemory(device, depthImageMem)
	vBuffer.Destroy(device)
	for i := 0; i < len(textures); i++ {
		vk.DestroyImageView(device, textures[i].view)
		vk.DestroySampler(device, textures[i].sampler)
		vk.DestroyImage(device, textures[i].image)
		vk.FreeMemory(device, textures[i].memory)
	}
	for _, v := range swapchainImageViews {
		vk.DestroyImageView(device, v)
	}
	vk.DestroyDescriptorSetLayout(device, descriptorSetLayoutTex)
	vk.DestroyDescriptorPool(device, descriptorPool)
	vk.DestroyPipelineLayout(device, pipelineLayout)
	vk.DestroyPipeline(device, pipeline)
	vk.DestroySwapchainKHR(device, swapchain)
	vk.DestroySurfaceKHR(instance, surface)
	vk.DestroyCommandPool(device, commandPool)
	vk.DestroyShaderModule(device, vertModule)
	vk.DestroyShaderModule(device, fragModule)
	window.Destroy()
	glfw.Terminate()
	vk.DestroyDevice(device)
	vk.DestroyInstance(instance)
}

// updateSceneData recomputes the per-frame matrices and writes them straight
// into the mapped uniform buffer for the current frame.
func updateSceneData() {
	aspect := float32(windowSize[0]) / float32(windowSize[1])
	// mathgl's Perspective is OpenGL-style (Z in -1..1, Y up). This clip matrix
	// applies GLM_FORCE_DEPTH_ZERO_TO_ONE (remap Z to 0..1) and flips Y for
	// Vulkan's coordinate system, matching the reference's projection result.
	clip := mgl32.Mat4{1, 0, 0, 0, 0, -1, 0, 0, 0, 0, 0.5, 0, 0, 0, 0.5, 1}
	shaderData.Projection = clip.Mul4(mgl32.Perspective(mgl32.DegToRad(45), aspect, 0.1, 32))
	shaderData.View = mgl32.Translate3D(camPos[0], camPos[1], camPos[2])
	for i := 0; i < 3; i++ {
		// Three instances spaced along X, each rotated by its accumulated euler angles.
		pos := mgl32.Vec3{float32(i-1) * 3.0, 0, 0}
		shaderData.Model[i] = mgl32.Translate3D(pos[0], pos[1], pos[2]).Mul4(
			mgl32.AnglesToQuat(objectRotations[i][0], objectRotations[i][1], objectRotations[i][2], mgl32.XYZ).Mat4())
	}
	shaderData.LightPos = [4]float32{0, -10, 10, 0}
	// The buffer is persistently mapped, so a struct assignment is the upload.
	*(*ShaderData)(shaderDataBuffers[frameIndex].Mapped) = shaderData
}

// recordCommandBuffer records one frame: barrier the attachments into their
// rendering layouts, draw the three instances with dynamic rendering, then
// barrier the swapchain image into PRESENT layout.
func recordCommandBuffer(cb vk.CommandBuffer) {
	chk(vk.ResetCommandBuffer(cb))
	chk(vk.BeginCommandBuffer(cb, vk.CommandBufferUsageOneTimeSubmit))

	colorRange := vk.ImageSubresourceRange{AspectMask: vk.ImageAspectColor, LevelCount: 1, LayerCount: 1}
	// Transition color image UNDEFINED->ATTACHMENT and depth image into its
	// attachment layout before rendering begins.
	vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{
		{
			SrcStageMask: vk.PipelineStage2ColorAttachmentOutput, SrcAccessMask: vk.Access2None,
			DstStageMask: vk.PipelineStage2ColorAttachmentOutput, DstAccessMask: vk.Access2ColorAttachmentWrite,
			OldLayout: vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutColorAttachmentOptimal,
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
			Image: swapchainImages[imageIndex], SubresourceRange: colorRange,
		},
		{
			SrcStageMask: vk.PipelineStage2LateFragmentTests, SrcAccessMask: vk.Access2DepthStencilAttachmentWrite,
			DstStageMask: vk.PipelineStage2EarlyFragmentTests, DstAccessMask: vk.Access2DepthStencilAttachmentWrite,
			OldLayout: vk.ImageLayoutUndefined, NewLayout: vk.ImageLayoutDepthAttachmentOptimal,
			SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
			// The chosen depth format carries stencil, so the barrier covers both aspects.
			Image: depthImage, SubresourceRange: vk.ImageSubresourceRange{AspectMask: vk.ImageAspectDepth | vk.ImageAspectStencil, LevelCount: 1, LayerCount: 1},
		},
	})

	extent := vk.Extent2D{Width: uint32(windowSize[0]), Height: uint32(windowSize[1])}
	vk.CmdBeginRendering(cb, vk.RenderingInfo{
		RenderArea: vk.Rect2D{Extent: extent},
		LayerCount: 1,
		ColorAttachments: []vk.RenderingAttachmentInfo{{
			ImageView: swapchainImageViews[imageIndex], ImageLayout: vk.ImageLayoutColorAttachmentOptimal,
			LoadOp: vk.AttachmentLoadOpClear, StoreOp: vk.AttachmentStoreOpStore,
			ClearValue: vk.ClearColor(0, 0, 0, 1),
		}},
		DepthAttachment: &vk.RenderingAttachmentInfo{
			ImageView: depthImageView, ImageLayout: vk.ImageLayoutDepthAttachmentOptimal,
			LoadOp: vk.AttachmentLoadOpClear, StoreOp: vk.AttachmentStoreOpDontCare,
			ClearValue: vk.ClearDepthStencil(1, 0),
		},
	})

	// Dynamic viewport + scissor cover the whole window.
	vk.CmdSetViewport(cb, vk.Viewport{Width: float32(windowSize[0]), Height: float32(windowSize[1]), MaxDepth: 1})
	vk.CmdSetScissor(cb, vk.Rect2D{Extent: extent})
	vk.CmdBindPipeline(cb, vk.PipelineBindPointGraphics, pipeline)
	vk.CmdBindDescriptorSets(cb, vk.PipelineBindPointGraphics, pipelineLayout, 0, []vk.DescriptorSet{descriptorSetTex})
	// Vertices at offset 0, indices at offset vBufSize inside the same buffer.
	vk.CmdBindVertexBuffer(cb, 0, vBuffer.Buffer, 0)
	vk.CmdBindIndexBuffer(cb, vBuffer.Buffer, vBufSize, vk.IndexTypeUint16)
	// Push this frame's uniform-buffer address, then draw three instances.
	addr := shaderDataAddrs[frameIndex]
	vk.CmdPushConstants(cb, pipelineLayout, vk.ShaderStageVertex, 0, 8, unsafe.Pointer(&addr))
	vk.CmdDrawIndexed(cb, indexCnt, 3, 0, 0, 0)

	vk.CmdEndRendering(cb)

	// Transition the swapchain image to PRESENT layout for the presentation engine.
	vk.CmdPipelineBarrier2(cb, []vk.ImageMemoryBarrier2{{
		SrcStageMask: vk.PipelineStage2ColorAttachmentOutput, SrcAccessMask: vk.Access2ColorAttachmentWrite,
		DstStageMask: vk.PipelineStage2ColorAttachmentOutput, DstAccessMask: vk.Access2None,
		OldLayout: vk.ImageLayoutColorAttachmentOptimal, NewLayout: vk.ImageLayoutPresentSrcKHR,
		SrcQueueFamilyIndex: vk.QueueFamilyIgnored, DstQueueFamilyIndex: vk.QueueFamilyIgnored,
		Image: swapchainImages[imageIndex], SubresourceRange: colorRange,
	}})
	chk(vk.EndCommandBuffer(cb))
}

// recreateSwapchain rebuilds the swapchain, its views, the render-complete
// semaphores, and the depth buffer after a resize. Mirrors the reference's
// inline updateSwapchain block.
func recreateSwapchain(ci *vk.SwapchainCreateInfo) {
	windowSize[0], windowSize[1] = window.GetSize()
	for windowSize[0] == 0 || windowSize[1] == 0 { // block while minimized
		glfw.WaitEvents()
		windowSize[0], windowSize[1] = window.GetSize()
	}
	updateSwapchain = false
	chk(vk.DeviceWaitIdle(device))

	old := swapchain
	ci.OldSwapchain = old
	ci.ImageExtent = vk.Extent2D{Width: uint32(windowSize[0]), Height: uint32(windowSize[1])}
	var err error
	swapchain, err = vk.CreateSwapchainKHR(device, *ci)
	chk(err)

	for _, v := range swapchainImageViews {
		vk.DestroyImageView(device, v)
	}
	swapchainImages, err = vk.GetSwapchainImagesKHR(device, swapchain)
	chk(err)
	swapchainImageViews = make([]vk.ImageView, len(swapchainImages))
	for i := range swapchainImages {
		swapchainImageViews[i] = createImageView(swapchainImages[i], imageFormat, vk.ImageAspectColor)
	}

	// Render-complete semaphores are per image, so re-make them for the new count.
	for _, s := range renderCompleteSemaphores {
		vk.DestroySemaphore(device, s)
	}
	renderCompleteSemaphores = make([]vk.Semaphore, len(swapchainImages))
	for i := range renderCompleteSemaphores {
		renderCompleteSemaphores[i], err = vk.CreateSemaphore(device)
		chk(err)
	}
	vk.DestroySwapchainKHR(device, old)

	// Depth buffer must match the new size.
	vk.DestroyImageView(device, depthImageView)
	vk.DestroyImage(device, depthImage)
	vk.FreeMemory(device, depthImageMem)
	depthImage, depthImageMem = vkx.CreateImage(device, memProps, vk.ImageCreateInfo{
		ImageType: vk.ImageType2D,
		Format:    depthFormat,
		Extent:    vk.Extent3D{Width: uint32(windowSize[0]), Height: uint32(windowSize[1]), Depth: 1},
		Usage:     vk.ImageUsageDepthStencilAttachment,
	})
	depthImageView = createImageView(depthImage, depthFormat, vk.ImageAspectDepth)
}

// createImageView is a tiny inline helper (the reference builds each
// VkImageViewCreateInfo by hand); every view here is a 2D single-mip view.
func createImageView(img vk.Image, format vk.Format, aspect vk.ImageAspectFlags) vk.ImageView {
	v, err := vk.CreateImageView(device, vk.ImageViewCreateInfo{
		Image: img, ViewType: vk.ImageViewType2D, Format: format,
		SubresourceRange: vk.ImageSubresourceRange{AspectMask: aspect, LevelCount: 1, LayerCount: 1},
	})
	chk(err)
	return v
}

// ---- input (SDL's event queue -> GLFW callbacks + polling) ---------------

var (
	dragging   bool       // left mouse button held
	lastCursor [2]float64 // cursor position on the previous polled frame
	prevPlus   bool       // edge-detect state for the +/- keys
	prevMinus  bool
)

// installInput wires the GLFW callbacks for events that can only arrive that way
// (scroll wheel, framebuffer resize).
func installInput() {
	window.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) { updateSwapchain = true })
	window.SetScrollCallback(func(_ *glfw.Window, _, dy float64) {
		camPos[2] += float32(dy) * 0.5 // wheel dollies the camera along Z
	})
}

// pollInput reads mouse-drag and the +/- keys each frame. Left-drag rotates the
// currently selected instance; +/- cycle the selection (edge-detected so one
// press advances one step).
func pollInput(elapsed float32) {
	// Mouse drag -> rotate the selected object.
	if window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press {
		x, y := window.GetCursorPos()
		if dragging {
			sel := shaderData.Selected
			objectRotations[sel][0] -= float32(y-lastCursor[1]) * elapsed
			objectRotations[sel][1] += float32(x-lastCursor[0]) * elapsed
		}
		lastCursor[0], lastCursor[1] = x, y
		dragging = true
	} else {
		dragging = false
	}

	// + key: select the next instance (wrapping 2 -> 0).
	plus := window.GetKey(glfw.KeyEqual) == glfw.Press || window.GetKey(glfw.KeyKPAdd) == glfw.Press
	if plus && !prevPlus {
		if shaderData.Selected < 2 {
			shaderData.Selected++
		} else {
			shaderData.Selected = 0
		}
	}
	prevPlus = plus

	// - key: select the previous instance (wrapping 0 -> 2).
	minus := window.GetKey(glfw.KeyMinus) == glfw.Press || window.GetKey(glfw.KeyKPSubtract) == glfw.Press
	if minus && !prevMinus {
		if shaderData.Selected > 0 {
			shaderData.Selected--
		} else {
			shaderData.Selected = 2
		}
	}
	prevMinus = minus
}
