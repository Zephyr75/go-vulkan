package vk

/*
#include <vulkan/vulkan.h>
*/
import "C"

// ---- small value structs ------------------------------------------------

type Extent2D struct{ Width, Height uint32 }
type Extent3D struct{ Width, Height, Depth uint32 }
type Offset2D struct{ X, Y int32 }
type Rect2D struct {
	Offset Offset2D
	Extent Extent2D
}
type Viewport struct {
	X, Y, Width, Height, MinDepth, MaxDepth float32
}

// ---- enums (int32 to match C enum width) --------------------------------

type Format int32

const (
	FormatUndefined          = Format(C.VK_FORMAT_UNDEFINED)
	FormatR8G8B8A8Unorm      = Format(C.VK_FORMAT_R8G8B8A8_UNORM)
	FormatR8G8B8A8Srgb       = Format(C.VK_FORMAT_R8G8B8A8_SRGB)
	FormatB8G8R8A8Unorm      = Format(C.VK_FORMAT_B8G8R8A8_UNORM)
	FormatB8G8R8A8Srgb       = Format(C.VK_FORMAT_B8G8R8A8_SRGB)
	FormatD32Sfloat          = Format(C.VK_FORMAT_D32_SFLOAT)
	FormatD32SfloatS8Uint    = Format(C.VK_FORMAT_D32_SFLOAT_S8_UINT)
	FormatD24UnormS8Uint     = Format(C.VK_FORMAT_D24_UNORM_S8_UINT)
	FormatR32G32Sfloat       = Format(C.VK_FORMAT_R32G32_SFLOAT)
	FormatR32G32B32Sfloat    = Format(C.VK_FORMAT_R32G32B32_SFLOAT)
	FormatR32G32B32A32Sfloat = Format(C.VK_FORMAT_R32G32B32A32_SFLOAT)
)

type ImageLayout int32

const (
	ImageLayoutUndefined              = ImageLayout(C.VK_IMAGE_LAYOUT_UNDEFINED)
	ImageLayoutGeneral                = ImageLayout(C.VK_IMAGE_LAYOUT_GENERAL)
	ImageLayoutColorAttachmentOptimal = ImageLayout(C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL)
	ImageLayoutDepthAttachmentOptimal = ImageLayout(C.VK_IMAGE_LAYOUT_DEPTH_ATTACHMENT_OPTIMAL)
	ImageLayoutShaderReadOnlyOptimal  = ImageLayout(C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL)
	ImageLayoutTransferSrcOptimal     = ImageLayout(C.VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL)
	ImageLayoutTransferDstOptimal     = ImageLayout(C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL)
	ImageLayoutPresentSrcKHR          = ImageLayout(C.VK_IMAGE_LAYOUT_PRESENT_SRC_KHR)
)

type SharingMode int32

const (
	SharingModeExclusive  = SharingMode(C.VK_SHARING_MODE_EXCLUSIVE)
	SharingModeConcurrent = SharingMode(C.VK_SHARING_MODE_CONCURRENT)
)

type ImageType int32

const (
	ImageType1D = ImageType(C.VK_IMAGE_TYPE_1D)
	ImageType2D = ImageType(C.VK_IMAGE_TYPE_2D)
	ImageType3D = ImageType(C.VK_IMAGE_TYPE_3D)
)

type ImageViewType int32

const (
	ImageViewType2D      = ImageViewType(C.VK_IMAGE_VIEW_TYPE_2D)
	ImageViewTypeCube    = ImageViewType(C.VK_IMAGE_VIEW_TYPE_CUBE)
	ImageViewType2DArray = ImageViewType(C.VK_IMAGE_VIEW_TYPE_2D_ARRAY)
)

type ImageCreateFlags uint32

// Required on images sampled through a cube (or cube-array) view.
const ImageCreateCubeCompatible = ImageCreateFlags(C.VK_IMAGE_CREATE_CUBE_COMPATIBLE_BIT)

type ColorSpace int32

const ColorSpaceSrgbNonlinearKHR = ColorSpace(C.VK_COLOR_SPACE_SRGB_NONLINEAR_KHR)

type PresentMode int32

const (
	PresentModeImmediateKHR = PresentMode(C.VK_PRESENT_MODE_IMMEDIATE_KHR)
	PresentModeMailboxKHR   = PresentMode(C.VK_PRESENT_MODE_MAILBOX_KHR)
	PresentModeFifoKHR      = PresentMode(C.VK_PRESENT_MODE_FIFO_KHR)
)

type PhysicalDeviceType int32

const (
	PhysicalDeviceTypeIntegratedGPU = PhysicalDeviceType(C.VK_PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU)
	PhysicalDeviceTypeDiscreteGPU   = PhysicalDeviceType(C.VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU)
)

type DescriptorType int32

const (
	DescriptorTypeCombinedImageSampler = DescriptorType(C.VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER)
	DescriptorTypeUniformBuffer        = DescriptorType(C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER)
	DescriptorTypeStorageBuffer        = DescriptorType(C.VK_DESCRIPTOR_TYPE_STORAGE_BUFFER)
)

type PipelineBindPoint int32

const PipelineBindPointGraphics = PipelineBindPoint(C.VK_PIPELINE_BIND_POINT_GRAPHICS)

type Filter int32

const (
	FilterNearest = Filter(C.VK_FILTER_NEAREST)
	FilterLinear  = Filter(C.VK_FILTER_LINEAR)
)

type SamplerAddressMode int32

const (
	SamplerAddressModeRepeat        = SamplerAddressMode(C.VK_SAMPLER_ADDRESS_MODE_REPEAT)
	SamplerAddressModeClampToEdge   = SamplerAddressMode(C.VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE)
	SamplerAddressModeClampToBorder = SamplerAddressMode(C.VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_BORDER)
)

// Border color used by ClampToBorder address modes. The zero value keeps the
// package's historical default (opaque black); see CreateSampler.
type BorderColor int32

const (
	BorderColorOpaqueBlackFloat = BorderColor(C.VK_BORDER_COLOR_FLOAT_OPAQUE_BLACK)
	BorderColorOpaqueWhiteFloat = BorderColor(C.VK_BORDER_COLOR_FLOAT_OPAQUE_WHITE)
)

type SamplerMipmapMode int32

const (
	SamplerMipmapModeNearest = SamplerMipmapMode(C.VK_SAMPLER_MIPMAP_MODE_NEAREST)
	SamplerMipmapModeLinear  = SamplerMipmapMode(C.VK_SAMPLER_MIPMAP_MODE_LINEAR)
)

type PrimitiveTopology int32

const PrimitiveTopologyTriangleList = PrimitiveTopology(C.VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST)

type PolygonMode int32

const PolygonModeFill = PolygonMode(C.VK_POLYGON_MODE_FILL)

type FrontFace int32

const (
	FrontFaceCounterClockwise = FrontFace(C.VK_FRONT_FACE_COUNTER_CLOCKWISE)
	FrontFaceClockwise        = FrontFace(C.VK_FRONT_FACE_CLOCKWISE)
)

type CompareOp int32

const (
	CompareOpLess        = CompareOp(C.VK_COMPARE_OP_LESS)
	CompareOpLessOrEqual = CompareOp(C.VK_COMPARE_OP_LESS_OR_EQUAL)
	CompareOpEqual       = CompareOp(C.VK_COMPARE_OP_EQUAL)
	CompareOpAlways      = CompareOp(C.VK_COMPARE_OP_ALWAYS)
)

type VertexInputRate int32

const (
	VertexInputRateVertex   = VertexInputRate(C.VK_VERTEX_INPUT_RATE_VERTEX)
	VertexInputRateInstance = VertexInputRate(C.VK_VERTEX_INPUT_RATE_INSTANCE)
)

type AttachmentLoadOp int32

const (
	AttachmentLoadOpLoad     = AttachmentLoadOp(C.VK_ATTACHMENT_LOAD_OP_LOAD)
	AttachmentLoadOpClear    = AttachmentLoadOp(C.VK_ATTACHMENT_LOAD_OP_CLEAR)
	AttachmentLoadOpDontCare = AttachmentLoadOp(C.VK_ATTACHMENT_LOAD_OP_DONT_CARE)
)

type AttachmentStoreOp int32

const (
	AttachmentStoreOpStore    = AttachmentStoreOp(C.VK_ATTACHMENT_STORE_OP_STORE)
	AttachmentStoreOpDontCare = AttachmentStoreOp(C.VK_ATTACHMENT_STORE_OP_DONT_CARE)
)

type IndexType int32

const (
	IndexTypeUint16 = IndexType(C.VK_INDEX_TYPE_UINT16)
	IndexTypeUint32 = IndexType(C.VK_INDEX_TYPE_UINT32)
)

type DynamicState int32

const (
	DynamicStateViewport = DynamicState(C.VK_DYNAMIC_STATE_VIEWPORT)
	DynamicStateScissor  = DynamicState(C.VK_DYNAMIC_STATE_SCISSOR)
)

type BlendFactor int32

const (
	BlendFactorZero             = BlendFactor(C.VK_BLEND_FACTOR_ZERO)
	BlendFactorOne              = BlendFactor(C.VK_BLEND_FACTOR_ONE)
	BlendFactorSrcAlpha         = BlendFactor(C.VK_BLEND_FACTOR_SRC_ALPHA)
	BlendFactorOneMinusSrcAlpha = BlendFactor(C.VK_BLEND_FACTOR_ONE_MINUS_SRC_ALPHA)
)

type BlendOp int32

const BlendOpAdd = BlendOp(C.VK_BLEND_OP_ADD)

type LogicOp int32

const LogicOpCopy = LogicOp(C.VK_LOGIC_OP_COPY)

type StencilOp int32

const StencilOpKeep = StencilOp(C.VK_STENCIL_OP_KEEP)

// ---- flag bits (uint32) --------------------------------------------------

type ImageUsageFlags uint32

const (
	ImageUsageTransferSrc            = ImageUsageFlags(C.VK_IMAGE_USAGE_TRANSFER_SRC_BIT)
	ImageUsageTransferDst            = ImageUsageFlags(C.VK_IMAGE_USAGE_TRANSFER_DST_BIT)
	ImageUsageSampled                = ImageUsageFlags(C.VK_IMAGE_USAGE_SAMPLED_BIT)
	ImageUsageColorAttachment        = ImageUsageFlags(C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT)
	ImageUsageDepthStencilAttachment = ImageUsageFlags(C.VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT)
	// An attachment that lives only inside one pass, letting a tiler keep it in
	// on-chip memory: MSAA colour targets that are resolved, never stored
	ImageUsageTransientAttachment = ImageUsageFlags(C.VK_IMAGE_USAGE_TRANSIENT_ATTACHMENT_BIT)
)

type BufferUsageFlags uint32

const (
	BufferUsageTransferSrc         = BufferUsageFlags(C.VK_BUFFER_USAGE_TRANSFER_SRC_BIT)
	BufferUsageTransferDst         = BufferUsageFlags(C.VK_BUFFER_USAGE_TRANSFER_DST_BIT)
	BufferUsageUniformBuffer       = BufferUsageFlags(C.VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT)
	BufferUsageStorageBuffer       = BufferUsageFlags(C.VK_BUFFER_USAGE_STORAGE_BUFFER_BIT)
	BufferUsageIndexBuffer         = BufferUsageFlags(C.VK_BUFFER_USAGE_INDEX_BUFFER_BIT)
	BufferUsageVertexBuffer        = BufferUsageFlags(C.VK_BUFFER_USAGE_VERTEX_BUFFER_BIT)
	BufferUsageShaderDeviceAddress = BufferUsageFlags(C.VK_BUFFER_USAGE_SHADER_DEVICE_ADDRESS_BIT)
)

type MemoryPropertyFlags uint32

const (
	MemoryPropertyDeviceLocal  = MemoryPropertyFlags(C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT)
	MemoryPropertyHostVisible  = MemoryPropertyFlags(C.VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT)
	MemoryPropertyHostCoherent = MemoryPropertyFlags(C.VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)
	MemoryPropertyHostCached   = MemoryPropertyFlags(C.VK_MEMORY_PROPERTY_HOST_CACHED_BIT)
)

type ImageAspectFlags uint32

const (
	ImageAspectColor   = ImageAspectFlags(C.VK_IMAGE_ASPECT_COLOR_BIT)
	ImageAspectDepth   = ImageAspectFlags(C.VK_IMAGE_ASPECT_DEPTH_BIT)
	ImageAspectStencil = ImageAspectFlags(C.VK_IMAGE_ASPECT_STENCIL_BIT)
)

type FormatFeatureFlags uint32

const FormatFeatureDepthStencilAttachment = FormatFeatureFlags(C.VK_FORMAT_FEATURE_DEPTH_STENCIL_ATTACHMENT_BIT)

type QueueFlags uint32

const (
	QueueGraphics = QueueFlags(C.VK_QUEUE_GRAPHICS_BIT)
	QueueCompute  = QueueFlags(C.VK_QUEUE_COMPUTE_BIT)
	QueueTransfer = QueueFlags(C.VK_QUEUE_TRANSFER_BIT)
)

type SampleCountFlags uint32

// The spec guarantees framebufferColor/DepthSampleCounts contain 1, 2 and 4, so
// only counts above that need a limits query before use
const (
	SampleCount1Bit = SampleCountFlags(C.VK_SAMPLE_COUNT_1_BIT)
	SampleCount2Bit = SampleCountFlags(C.VK_SAMPLE_COUNT_2_BIT)
	SampleCount4Bit = SampleCountFlags(C.VK_SAMPLE_COUNT_4_BIT)
	SampleCount8Bit = SampleCountFlags(C.VK_SAMPLE_COUNT_8_BIT)
)

type ResolveModeFlags uint32

const (
	ResolveModeNone       = ResolveModeFlags(C.VK_RESOLVE_MODE_NONE)
	ResolveModeAverage    = ResolveModeFlags(C.VK_RESOLVE_MODE_AVERAGE_BIT)
	ResolveModeSampleZero = ResolveModeFlags(C.VK_RESOLVE_MODE_SAMPLE_ZERO_BIT)
)

type ColorComponentFlags uint32

const (
	ColorComponentR = ColorComponentFlags(C.VK_COLOR_COMPONENT_R_BIT)
	ColorComponentG = ColorComponentFlags(C.VK_COLOR_COMPONENT_G_BIT)
	ColorComponentB = ColorComponentFlags(C.VK_COLOR_COMPONENT_B_BIT)
	ColorComponentA = ColorComponentFlags(C.VK_COLOR_COMPONENT_A_BIT)
)

type ShaderStageFlags uint32

const (
	ShaderStageVertex      = ShaderStageFlags(C.VK_SHADER_STAGE_VERTEX_BIT)
	ShaderStageGeometry    = ShaderStageFlags(C.VK_SHADER_STAGE_GEOMETRY_BIT)
	ShaderStageFragment    = ShaderStageFlags(C.VK_SHADER_STAGE_FRAGMENT_BIT)
	ShaderStageAllGraphics = ShaderStageFlags(C.VK_SHADER_STAGE_ALL_GRAPHICS)
)

type CullModeFlags uint32

const (
	CullModeNone  = CullModeFlags(C.VK_CULL_MODE_NONE)
	CullModeFront = CullModeFlags(C.VK_CULL_MODE_FRONT_BIT)
	CullModeBack  = CullModeFlags(C.VK_CULL_MODE_BACK_BIT)
)

type CommandBufferUsageFlags uint32

const CommandBufferUsageOneTimeSubmit = CommandBufferUsageFlags(C.VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT)

type CommandPoolCreateFlags uint32

const (
	CommandPoolCreateResetCommandBuffer = CommandPoolCreateFlags(C.VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT)
	CommandPoolCreateTransient          = CommandPoolCreateFlags(C.VK_COMMAND_POOL_CREATE_TRANSIENT_BIT)
)

type FenceCreateFlags uint32

const FenceCreateSignaled = FenceCreateFlags(C.VK_FENCE_CREATE_SIGNALED_BIT)

type CompositeAlphaFlags uint32

const CompositeAlphaOpaqueKHR = CompositeAlphaFlags(C.VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR)

type SurfaceTransformFlags uint32

const SurfaceTransformIdentityKHR = SurfaceTransformFlags(C.VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR)

type DescriptorBindingFlags uint32

const (
	DescriptorBindingUpdateAfterBind         = DescriptorBindingFlags(C.VK_DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT)
	DescriptorBindingPartiallyBound          = DescriptorBindingFlags(C.VK_DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT)
	DescriptorBindingVariableDescriptorCount = DescriptorBindingFlags(C.VK_DESCRIPTOR_BINDING_VARIABLE_DESCRIPTOR_COUNT_BIT)
)

type DescriptorSetLayoutCreateFlags uint32

const DescriptorSetLayoutCreateUpdateAfterBindPool = DescriptorSetLayoutCreateFlags(C.VK_DESCRIPTOR_SET_LAYOUT_CREATE_UPDATE_AFTER_BIND_POOL_BIT)

type DescriptorPoolCreateFlags uint32

const (
	DescriptorPoolCreateUpdateAfterBind   = DescriptorPoolCreateFlags(C.VK_DESCRIPTOR_POOL_CREATE_UPDATE_AFTER_BIND_BIT)
	DescriptorPoolCreateFreeDescriptorSet = DescriptorPoolCreateFlags(C.VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT)
)

// ---- sync2 flags (uint64) ------------------------------------------------

// VkFlags64 (sync2) values are static-const in the header — cgo emits an
// unresolved external ref for each, so they are hardcoded from vulkan_core.h.
type PipelineStageFlags2 uint64

const (
	PipelineStage2None                  = PipelineStageFlags2(0)
	PipelineStage2TopOfPipe             = PipelineStageFlags2(0x00000001)
	PipelineStage2FragmentShader        = PipelineStageFlags2(0x00000080)
	PipelineStage2EarlyFragmentTests    = PipelineStageFlags2(0x00000100)
	PipelineStage2LateFragmentTests     = PipelineStageFlags2(0x00000200)
	PipelineStage2ColorAttachmentOutput = PipelineStageFlags2(0x00000400)
	PipelineStage2Transfer              = PipelineStageFlags2(0x00001000) // ALL_TRANSFER
	PipelineStage2BottomOfPipe          = PipelineStageFlags2(0x00002000)
	PipelineStage2AllCommands           = PipelineStageFlags2(0x00010000)
	PipelineStage2Copy                  = PipelineStageFlags2(0x100000000)
)

type AccessFlags2 uint64

const (
	Access2None                        = AccessFlags2(0)
	Access2ShaderRead                  = AccessFlags2(0x00000020)
	Access2ColorAttachmentWrite        = AccessFlags2(0x00000100)
	Access2DepthStencilAttachmentWrite = AccessFlags2(0x00000400)
	Access2TransferRead                = AccessFlags2(0x00000800)
	Access2TransferWrite               = AccessFlags2(0x00001000)
	Access2MemoryRead                  = AccessFlags2(0x00008000)
	Access2MemoryWrite                 = AccessFlags2(0x00010000)
	Access2ShaderSampledRead           = AccessFlags2(0x100000000)
)

// WholeSize maps VK_WHOLE_SIZE for map/flush ranges
const WholeSize = ^uint64(0)

// ApiVersion13 is the Vulkan 1.3 version number
const ApiVersion13 = uint32(C.VK_API_VERSION_1_3)

// QueueFamilyIgnored for barrier ownership fields
const QueueFamilyIgnored = uint32(C.VK_QUEUE_FAMILY_IGNORED)

// ---- additions for compute, storage images, HDR, timing and labels -------
//
// Every constant below was added for Overdrive (BINDINGS_OVERDRIVE.md §2). They
// are grouped by the batch that added them rather than merged into the sections
// above, so what a batch added stays readable.

// Batch 1: formats. HDR colour, single- and two-channel float targets, and the
// block-compressed families a loader hands over already compressed
const (
	FormatR8Unorm               = Format(C.VK_FORMAT_R8_UNORM)
	FormatR16Sfloat             = Format(C.VK_FORMAT_R16_SFLOAT)
	FormatR16G16Sfloat          = Format(C.VK_FORMAT_R16G16_SFLOAT)
	FormatR16G16B16A16Sfloat    = Format(C.VK_FORMAT_R16G16B16A16_SFLOAT)
	FormatR11G11B10UfloatPack32 = Format(C.VK_FORMAT_B10G11R11_UFLOAT_PACK32)
	FormatR32Sfloat             = Format(C.VK_FORMAT_R32_SFLOAT)
	FormatR32Uint               = Format(C.VK_FORMAT_R32_UINT)
	FormatBC5UnormBlock         = Format(C.VK_FORMAT_BC5_UNORM_BLOCK)
	FormatBC6HUfloatBlock       = Format(C.VK_FORMAT_BC6H_UFLOAT_BLOCK)
	FormatBC7UnormBlock         = Format(C.VK_FORMAT_BC7_UNORM_BLOCK)
	FormatBC7SrgbBlock          = Format(C.VK_FORMAT_BC7_SRGB_BLOCK)
)

// Batch 4: the view types a 3D froxel grid and a probe array need
const (
	ImageViewType1D        = ImageViewType(C.VK_IMAGE_VIEW_TYPE_1D)
	ImageViewType3D        = ImageViewType(C.VK_IMAGE_VIEW_TYPE_3D)
	ImageViewTypeCubeArray = ImageViewType(C.VK_IMAGE_VIEW_TYPE_CUBE_ARRAY)
)

// Batch 4: an image a compute shader writes is bound as a storage image, in
// GENERAL layout, with the sampler field of its descriptor left zero
const (
	ImageUsageStorage          = ImageUsageFlags(C.VK_IMAGE_USAGE_STORAGE_BIT)
	DescriptorTypeStorageImage = DescriptorType(C.VK_DESCRIPTOR_TYPE_STORAGE_IMAGE)
)

// Batch 9: the usage an indirect draw or dispatch reads its arguments from
const BufferUsageIndirectBuffer = BufferUsageFlags(C.VK_BUFFER_USAGE_INDIRECT_BUFFER_BIT)

// Batch 3: compute
const (
	PipelineBindPointCompute = PipelineBindPoint(C.VK_PIPELINE_BIND_POINT_COMPUTE)
	ShaderStageCompute       = ShaderStageFlags(C.VK_SHADER_STAGE_COMPUTE_BIT)
	ShaderStageAll           = ShaderStageFlags(C.VK_SHADER_STAGE_ALL)
)

// Batch 2/3: the stages and accesses a compute hazard is expressed in
const (
	PipelineStage2VertexShader  = PipelineStageFlags2(0x00000008)
	PipelineStage2ComputeShader = PipelineStageFlags2(0x00000800)
	PipelineStage2DrawIndirect  = PipelineStageFlags2(0x00000002)
	PipelineStage2AllGraphics   = PipelineStageFlags2(0x00008000)
	PipelineStage2Blit          = PipelineStageFlags2(0x400000000)
	PipelineStage2Resolve       = PipelineStageFlags2(0x200000000)
	PipelineStage2Clear         = PipelineStageFlags2(0x800000000)
)

const (
	Access2IndirectCommandRead        = AccessFlags2(0x00000001)
	Access2ShaderWrite                = AccessFlags2(0x00000040)
	Access2ColorAttachmentRead        = AccessFlags2(0x00000080)
	Access2DepthStencilAttachmentRead = AccessFlags2(0x00000200)
	Access2ShaderStorageRead          = AccessFlags2(0x200000000)
	Access2ShaderStorageWrite         = AccessFlags2(0x400000000)
)

// Batch 1: the bits a format probe tests, beside the depth one already here
const (
	FormatFeatureSampledImage             = FormatFeatureFlags(C.VK_FORMAT_FEATURE_SAMPLED_IMAGE_BIT)
	FormatFeatureStorageImage             = FormatFeatureFlags(C.VK_FORMAT_FEATURE_STORAGE_IMAGE_BIT)
	FormatFeatureColorAttachment          = FormatFeatureFlags(C.VK_FORMAT_FEATURE_COLOR_ATTACHMENT_BIT)
	FormatFeatureColorAttachmentBlend     = FormatFeatureFlags(C.VK_FORMAT_FEATURE_COLOR_ATTACHMENT_BLEND_BIT)
	FormatFeatureSampledImageFilterLinear = FormatFeatureFlags(C.VK_FORMAT_FEATURE_SAMPLED_IMAGE_FILTER_LINEAR_BIT)
	FormatFeatureBlitSrc                  = FormatFeatureFlags(C.VK_FORMAT_FEATURE_BLIT_SRC_BIT)
	FormatFeatureBlitDst                  = FormatFeatureFlags(C.VK_FORMAT_FEATURE_BLIT_DST_BIT)
)

// Batch 6: the blend factors and ops glass, water and additive glow need
const (
	BlendFactorSrcColor         = BlendFactor(C.VK_BLEND_FACTOR_SRC_COLOR)
	BlendFactorOneMinusSrcColor = BlendFactor(C.VK_BLEND_FACTOR_ONE_MINUS_SRC_COLOR)
	BlendFactorDstColor         = BlendFactor(C.VK_BLEND_FACTOR_DST_COLOR)
	BlendFactorOneMinusDstColor = BlendFactor(C.VK_BLEND_FACTOR_ONE_MINUS_DST_COLOR)
	BlendFactorDstAlpha         = BlendFactor(C.VK_BLEND_FACTOR_DST_ALPHA)
	BlendFactorOneMinusDstAlpha = BlendFactor(C.VK_BLEND_FACTOR_ONE_MINUS_DST_ALPHA)
	BlendFactorConstantColor    = BlendFactor(C.VK_BLEND_FACTOR_CONSTANT_COLOR)
	BlendFactorConstantAlpha    = BlendFactor(C.VK_BLEND_FACTOR_CONSTANT_ALPHA)
)

const (
	BlendOpSubtract        = BlendOp(C.VK_BLEND_OP_SUBTRACT)
	BlendOpReverseSubtract = BlendOp(C.VK_BLEND_OP_REVERSE_SUBTRACT)
	BlendOpMin             = BlendOp(C.VK_BLEND_OP_MIN)
	BlendOpMax             = BlendOp(C.VK_BLEND_OP_MAX)
)

// Batch 6: reverse-Z wants Greater, a stencil-free depth equality test Never
const (
	CompareOpNever          = CompareOp(C.VK_COMPARE_OP_NEVER)
	CompareOpGreater        = CompareOp(C.VK_COMPARE_OP_GREATER)
	CompareOpNotEqual       = CompareOp(C.VK_COMPARE_OP_NOT_EQUAL)
	CompareOpGreaterOrEqual = CompareOp(C.VK_COMPARE_OP_GREATER_OR_EQUAL)
)

// Batch 6: misc
const (
	SamplerAddressModeMirroredRepeat = SamplerAddressMode(C.VK_SAMPLER_ADDRESS_MODE_MIRRORED_REPEAT)
	PolygonModeLine                  = PolygonMode(C.VK_POLYGON_MODE_LINE)
)

const (
	DynamicStateDepthTestEnable  = DynamicState(C.VK_DYNAMIC_STATE_DEPTH_TEST_ENABLE)
	DynamicStateDepthWriteEnable = DynamicState(C.VK_DYNAMIC_STATE_DEPTH_WRITE_ENABLE)
	DynamicStateDepthBias        = DynamicState(C.VK_DYNAMIC_STATE_DEPTH_BIAS)
)
