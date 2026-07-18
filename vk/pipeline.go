package vk

/*
#include <stdlib.h>
#include <vulkan/vulkan.h>
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// Wraps SPIR-V bytes (must be a multiple of 4) in a shader module
func CreateShaderModule(d Device, code []byte) (ShaderModule, error) {
	if len(code) == 0 || len(code)%4 != 0 {
		return 0, Result(int32(C.VK_ERROR_INITIALIZATION_FAILED))
	}
	// Copy into C memory so pCode has no Go-pointer aliasing across the call.
	buf := C.CBytes(code)
	defer C.free(buf)
	info := C.VkShaderModuleCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
		codeSize: C.size_t(len(code)),
		pCode:    (*C.uint32_t)(buf),
	}
	var out C.VkShaderModule
	if err := check(C.vkCreateShaderModule(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return ShaderModule(unsafe.Pointer(out)), nil
}

func DestroyShaderModule(d Device, m ShaderModule) {
	C.vkDestroyShaderModule(C.VkDevice(unsafe.Pointer(d)), C.VkShaderModule(unsafe.Pointer(m)), nil)
}

// PushConstantRange exposes a byte range of push constants to a set of stages.
type PushConstantRange struct {
	StageFlags ShaderStageFlags
	Offset     uint32
	Size       uint32
}

type PipelineLayoutCreateInfo struct {
	SetLayouts         []DescriptorSetLayout
	PushConstantRanges []PushConstantRange
}

// Creates a pipeline layout from set layouts and push constant ranges
func CreatePipelineLayout(d Device, ci PipelineLayoutCreateInfo) (PipelineLayout, error) {
	var frees []unsafe.Pointer
	defer func() {
		for _, p := range frees {
			C.free(p)
		}
	}()

	var pin runtime.Pinner
	defer pin.Unpin()

	info := C.VkPipelineLayoutCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
	}
	if n := len(ci.SetLayouts); n > 0 {
		pin.Pin(&ci.SetLayouts[0])
		info.setLayoutCount = C.uint32_t(n)
		info.pSetLayouts = (*C.VkDescriptorSetLayout)(unsafe.Pointer(&ci.SetLayouts[0]))
	}
	if n := len(ci.PushConstantRanges); n > 0 {
		pc := (*C.VkPushConstantRange)(C.calloc(C.size_t(n), C.size_t(unsafe.Sizeof(C.VkPushConstantRange{}))))
		frees = append(frees, unsafe.Pointer(pc))
		ps := unsafe.Slice(pc, n)
		for i, r := range ci.PushConstantRanges {
			ps[i].stageFlags = C.VkShaderStageFlags(r.StageFlags)
			ps[i].offset = C.uint32_t(r.Offset)
			ps[i].size = C.uint32_t(r.Size)
		}
		info.pushConstantRangeCount = C.uint32_t(n)
		info.pPushConstantRanges = pc
	}

	var out C.VkPipelineLayout
	if err := check(C.vkCreatePipelineLayout(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return PipelineLayout(unsafe.Pointer(out)), nil
}

func DestroyPipelineLayout(d Device, l PipelineLayout) {
	C.vkDestroyPipelineLayout(C.VkDevice(unsafe.Pointer(d)), C.VkPipelineLayout(unsafe.Pointer(l)), nil)
}

// ---- C arena ------------------------------------------------------------
//
// cgo forbids passing C a Go pointer that itself contains Go pointers, so every
// nested Vulkan sub-struct is built in C heap. arena tracks the allocations and
// frees them together after the create call returns (Vulkan deep-copies the
// create info, so freeing immediately is safe).

type arena struct{ frees []unsafe.Pointer }

// Returns zeroed heap C memory for n elements of the given size
func (a *arena) alloc(n int, size uintptr) unsafe.Pointer {
	p := C.calloc(C.size_t(n), C.size_t(size))
	a.frees = append(a.frees, p)
	return p
}

// Copies a Go string into C heap
func (a *arena) cstr(s string) *C.char {
	p := C.CString(s)
	a.frees = append(a.frees, unsafe.Pointer(p))
	return p
}

func (a *arena) free() {
	for _, p := range a.frees {
		C.free(p)
	}
}

// ---- graphics pipeline sub-state (1:1 mirrors of the Vk* structs) --------
//
// Each mirror maps a Vulkan VkPipeline*StateCreateInfo. Fields are 1:1; nothing
// is defaulted here, so Go zero values are your defaults (set LineWidth: 1.0,
// RasterizationSamples: SampleCount1Bit, etc. yourself, as you would in C).
// sType/pNext are filled by the marshal step, not by the caller.

type PipelineShaderStageCreateInfo struct {
	Stage  ShaderStageFlags
	Module ShaderModule
	Name   string // entry point, e.g. "main"
}

type VertexInputBinding struct {
	Binding   uint32
	Stride    uint32
	InputRate VertexInputRate
}

type VertexInputAttribute struct {
	Location uint32
	Binding  uint32
	Format   Format
	Offset   uint32
}

type PipelineVertexInputStateCreateInfo struct {
	Bindings   []VertexInputBinding
	Attributes []VertexInputAttribute
}

type PipelineInputAssemblyStateCreateInfo struct {
	Topology               PrimitiveTopology
	PrimitiveRestartEnable bool
}

// PipelineViewportStateCreateInfo carries counts and optional data. Leave the
// slices nil (with counts set) when viewport/scissor are dynamic.
type PipelineViewportStateCreateInfo struct {
	ViewportCount uint32
	Viewports     []Viewport
	ScissorCount  uint32
	Scissors      []Rect2D
}

type PipelineRasterizationStateCreateInfo struct {
	DepthClampEnable        bool
	RasterizerDiscardEnable bool
	PolygonMode             PolygonMode
	CullMode                CullModeFlags
	FrontFace               FrontFace
	DepthBiasEnable         bool
	DepthBiasConstantFactor float32
	DepthBiasClamp          float32
	DepthBiasSlopeFactor    float32
	LineWidth               float32
}

type PipelineMultisampleStateCreateInfo struct {
	RasterizationSamples  SampleCountFlags
	SampleShadingEnable   bool
	MinSampleShading      float32
	AlphaToCoverageEnable bool
	AlphaToOneEnable      bool
}

type StencilOpState struct {
	FailOp      StencilOp
	PassOp      StencilOp
	DepthFailOp StencilOp
	CompareOp   CompareOp
	CompareMask uint32
	WriteMask   uint32
	Reference   uint32
}

type PipelineDepthStencilStateCreateInfo struct {
	DepthTestEnable       bool
	DepthWriteEnable      bool
	DepthCompareOp        CompareOp
	DepthBoundsTestEnable bool
	StencilTestEnable     bool
	Front                 StencilOpState
	Back                  StencilOpState
	MinDepthBounds        float32
	MaxDepthBounds        float32
}

type PipelineColorBlendAttachmentState struct {
	BlendEnable         bool
	SrcColorBlendFactor BlendFactor
	DstColorBlendFactor BlendFactor
	ColorBlendOp        BlendOp
	SrcAlphaBlendFactor BlendFactor
	DstAlphaBlendFactor BlendFactor
	AlphaBlendOp        BlendOp
	ColorWriteMask      ColorComponentFlags
}

type PipelineColorBlendStateCreateInfo struct {
	LogicOpEnable  bool
	LogicOp        LogicOp
	Attachments    []PipelineColorBlendAttachmentState
	BlendConstants [4]float32
}

type PipelineDynamicStateCreateInfo struct {
	DynamicStates []DynamicState
}

// PipelineRenderingCreateInfo is chained via pNext for dynamic rendering (no
// render pass). FormatUndefined disables the depth/stencil attachment.
type PipelineRenderingCreateInfo struct {
	ColorAttachmentFormats  []Format
	DepthAttachmentFormat   Format
	StencilAttachmentFormat Format
}

// Nil sub-state pointers marshal to NULL, matching Vulkan's optionality
type GraphicsPipelineCreateInfo struct {
	Stages             []PipelineShaderStageCreateInfo
	VertexInputState   *PipelineVertexInputStateCreateInfo
	InputAssemblyState *PipelineInputAssemblyStateCreateInfo
	ViewportState      *PipelineViewportStateCreateInfo
	RasterizationState *PipelineRasterizationStateCreateInfo
	MultisampleState   *PipelineMultisampleStateCreateInfo
	DepthStencilState  *PipelineDepthStencilStateCreateInfo
	ColorBlendState    *PipelineColorBlendStateCreateInfo
	DynamicState       *PipelineDynamicStateCreateInfo
	Rendering          *PipelineRenderingCreateInfo // pNext
	Layout             PipelineLayout
}

// ---- marshal methods (one per sub-state, each trivial and 1:1) -----------

func (s *PipelineVertexInputStateCreateInfo) c(a *arena) *C.VkPipelineVertexInputStateCreateInfo {
	out := (*C.VkPipelineVertexInputStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineVertexInputStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO
	if n := len(s.Bindings); n > 0 {
		p := (*C.VkVertexInputBindingDescription)(a.alloc(n, unsafe.Sizeof(C.VkVertexInputBindingDescription{})))
		for i, b := range s.Bindings {
			e := &unsafe.Slice(p, n)[i]
			e.binding = C.uint32_t(b.Binding)
			e.stride = C.uint32_t(b.Stride)
			e.inputRate = C.VkVertexInputRate(b.InputRate)
		}
		out.vertexBindingDescriptionCount = C.uint32_t(n)
		out.pVertexBindingDescriptions = p
	}
	if n := len(s.Attributes); n > 0 {
		p := (*C.VkVertexInputAttributeDescription)(a.alloc(n, unsafe.Sizeof(C.VkVertexInputAttributeDescription{})))
		for i, at := range s.Attributes {
			e := &unsafe.Slice(p, n)[i]
			e.location = C.uint32_t(at.Location)
			e.binding = C.uint32_t(at.Binding)
			e.format = C.VkFormat(at.Format)
			e.offset = C.uint32_t(at.Offset)
		}
		out.vertexAttributeDescriptionCount = C.uint32_t(n)
		out.pVertexAttributeDescriptions = p
	}
	return out
}

func (s *PipelineInputAssemblyStateCreateInfo) c(a *arena) *C.VkPipelineInputAssemblyStateCreateInfo {
	out := (*C.VkPipelineInputAssemblyStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineInputAssemblyStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO
	out.topology = C.VkPrimitiveTopology(s.Topology)
	out.primitiveRestartEnable = vkBool(s.PrimitiveRestartEnable)
	return out
}

func (s *PipelineViewportStateCreateInfo) c(a *arena) *C.VkPipelineViewportStateCreateInfo {
	out := (*C.VkPipelineViewportStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineViewportStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO
	out.viewportCount = C.uint32_t(s.ViewportCount)
	out.scissorCount = C.uint32_t(s.ScissorCount)
	if n := len(s.Viewports); n > 0 {
		p := (*C.VkViewport)(a.alloc(n, unsafe.Sizeof(C.VkViewport{})))
		for i, v := range s.Viewports {
			e := &unsafe.Slice(p, n)[i]
			e.x, e.y = C.float(v.X), C.float(v.Y)
			e.width, e.height = C.float(v.Width), C.float(v.Height)
			e.minDepth, e.maxDepth = C.float(v.MinDepth), C.float(v.MaxDepth)
		}
		out.pViewports = p
	}
	if n := len(s.Scissors); n > 0 {
		p := (*C.VkRect2D)(a.alloc(n, unsafe.Sizeof(C.VkRect2D{})))
		for i, r := range s.Scissors {
			e := &unsafe.Slice(p, n)[i]
			e.offset.x, e.offset.y = C.int32_t(r.Offset.X), C.int32_t(r.Offset.Y)
			e.extent.width, e.extent.height = C.uint32_t(r.Extent.Width), C.uint32_t(r.Extent.Height)
		}
		out.pScissors = p
	}
	return out
}

func (s *PipelineRasterizationStateCreateInfo) c(a *arena) *C.VkPipelineRasterizationStateCreateInfo {
	out := (*C.VkPipelineRasterizationStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineRasterizationStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO
	out.depthClampEnable = vkBool(s.DepthClampEnable)
	out.rasterizerDiscardEnable = vkBool(s.RasterizerDiscardEnable)
	out.polygonMode = C.VkPolygonMode(s.PolygonMode)
	out.cullMode = C.VkCullModeFlags(s.CullMode)
	out.frontFace = C.VkFrontFace(s.FrontFace)
	out.depthBiasEnable = vkBool(s.DepthBiasEnable)
	out.depthBiasConstantFactor = C.float(s.DepthBiasConstantFactor)
	out.depthBiasClamp = C.float(s.DepthBiasClamp)
	out.depthBiasSlopeFactor = C.float(s.DepthBiasSlopeFactor)
	out.lineWidth = C.float(s.LineWidth)
	return out
}

func (s *PipelineMultisampleStateCreateInfo) c(a *arena) *C.VkPipelineMultisampleStateCreateInfo {
	out := (*C.VkPipelineMultisampleStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineMultisampleStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO
	out.rasterizationSamples = C.VkSampleCountFlagBits(s.RasterizationSamples)
	out.sampleShadingEnable = vkBool(s.SampleShadingEnable)
	out.minSampleShading = C.float(s.MinSampleShading)
	out.alphaToCoverageEnable = vkBool(s.AlphaToCoverageEnable)
	out.alphaToOneEnable = vkBool(s.AlphaToOneEnable)
	return out
}

func (s *PipelineDepthStencilStateCreateInfo) c(a *arena) *C.VkPipelineDepthStencilStateCreateInfo {
	out := (*C.VkPipelineDepthStencilStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineDepthStencilStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO
	out.depthTestEnable = vkBool(s.DepthTestEnable)
	out.depthWriteEnable = vkBool(s.DepthWriteEnable)
	out.depthCompareOp = C.VkCompareOp(s.DepthCompareOp)
	out.depthBoundsTestEnable = vkBool(s.DepthBoundsTestEnable)
	out.stencilTestEnable = vkBool(s.StencilTestEnable)
	out.front = s.Front.c()
	out.back = s.Back.c()
	out.minDepthBounds = C.float(s.MinDepthBounds)
	out.maxDepthBounds = C.float(s.MaxDepthBounds)
	return out
}

func (s StencilOpState) c() C.VkStencilOpState {
	return C.VkStencilOpState{
		failOp:      C.VkStencilOp(s.FailOp),
		passOp:      C.VkStencilOp(s.PassOp),
		depthFailOp: C.VkStencilOp(s.DepthFailOp),
		compareOp:   C.VkCompareOp(s.CompareOp),
		compareMask: C.uint32_t(s.CompareMask),
		writeMask:   C.uint32_t(s.WriteMask),
		reference:   C.uint32_t(s.Reference),
	}
}

func (s *PipelineColorBlendStateCreateInfo) c(a *arena) *C.VkPipelineColorBlendStateCreateInfo {
	out := (*C.VkPipelineColorBlendStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineColorBlendStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO
	out.logicOpEnable = vkBool(s.LogicOpEnable)
	out.logicOp = C.VkLogicOp(s.LogicOp)
	for i := 0; i < 4; i++ {
		out.blendConstants[i] = C.float(s.BlendConstants[i])
	}
	if n := len(s.Attachments); n > 0 {
		p := (*C.VkPipelineColorBlendAttachmentState)(a.alloc(n, unsafe.Sizeof(C.VkPipelineColorBlendAttachmentState{})))
		for i, at := range s.Attachments {
			e := &unsafe.Slice(p, n)[i]
			e.blendEnable = vkBool(at.BlendEnable)
			e.srcColorBlendFactor = C.VkBlendFactor(at.SrcColorBlendFactor)
			e.dstColorBlendFactor = C.VkBlendFactor(at.DstColorBlendFactor)
			e.colorBlendOp = C.VkBlendOp(at.ColorBlendOp)
			e.srcAlphaBlendFactor = C.VkBlendFactor(at.SrcAlphaBlendFactor)
			e.dstAlphaBlendFactor = C.VkBlendFactor(at.DstAlphaBlendFactor)
			e.alphaBlendOp = C.VkBlendOp(at.AlphaBlendOp)
			e.colorWriteMask = C.VkColorComponentFlags(at.ColorWriteMask)
		}
		out.attachmentCount = C.uint32_t(n)
		out.pAttachments = p
	}
	return out
}

func (s *PipelineDynamicStateCreateInfo) c(a *arena) *C.VkPipelineDynamicStateCreateInfo {
	out := (*C.VkPipelineDynamicStateCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineDynamicStateCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO
	if n := len(s.DynamicStates); n > 0 {
		p := (*C.VkDynamicState)(a.alloc(n, unsafe.Sizeof(C.VkDynamicState(0))))
		ds := unsafe.Slice(p, n)
		for i, v := range s.DynamicStates {
			ds[i] = C.VkDynamicState(v)
		}
		out.dynamicStateCount = C.uint32_t(n)
		out.pDynamicStates = p
	}
	return out
}

func (s *PipelineRenderingCreateInfo) c(a *arena) *C.VkPipelineRenderingCreateInfo {
	out := (*C.VkPipelineRenderingCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkPipelineRenderingCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_PIPELINE_RENDERING_CREATE_INFO
	out.depthAttachmentFormat = C.VkFormat(s.DepthAttachmentFormat)
	out.stencilAttachmentFormat = C.VkFormat(s.StencilAttachmentFormat)
	if n := len(s.ColorAttachmentFormats); n > 0 {
		p := (*C.VkFormat)(a.alloc(n, unsafe.Sizeof(C.VkFormat(0))))
		cf := unsafe.Slice(p, n)
		for i, f := range s.ColorAttachmentFormats {
			cf[i] = C.VkFormat(f)
		}
		out.colorAttachmentCount = C.uint32_t(n)
		out.pColorAttachmentFormats = p
	}
	return out
}

// Marshals the shader stage array into the arena
func stagesC(a *arena, in []PipelineShaderStageCreateInfo) (*C.VkPipelineShaderStageCreateInfo, C.uint32_t) {
	n := len(in)
	if n == 0 {
		return nil, 0
	}
	p := (*C.VkPipelineShaderStageCreateInfo)(a.alloc(n, unsafe.Sizeof(C.VkPipelineShaderStageCreateInfo{})))
	st := unsafe.Slice(p, n)
	for i, s := range in {
		st[i].sType = C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO
		st[i].stage = C.VkShaderStageFlagBits(s.Stage)
		st[i].module = C.VkShaderModule(unsafe.Pointer(s.Module))
		st[i].pName = a.cstr(s.Name)
	}
	return p, C.uint32_t(n)
}

// Marshals ci 1:1 into a VkGraphicsPipelineCreateInfo and creates a single pipeline; nothing is defaulted
func CreateGraphicsPipeline(d Device, ci GraphicsPipelineCreateInfo) (Pipeline, error) {
	var a arena
	defer a.free()

	info := C.VkGraphicsPipelineCreateInfo{
		sType:             C.VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		layout:            C.VkPipelineLayout(unsafe.Pointer(ci.Layout)),
		basePipelineIndex: -1,
	}
	info.pStages, info.stageCount = stagesC(&a, ci.Stages)
	if ci.Rendering != nil {
		info.pNext = unsafe.Pointer(ci.Rendering.c(&a))
	}
	if ci.VertexInputState != nil {
		info.pVertexInputState = ci.VertexInputState.c(&a)
	}
	if ci.InputAssemblyState != nil {
		info.pInputAssemblyState = ci.InputAssemblyState.c(&a)
	}
	if ci.ViewportState != nil {
		info.pViewportState = ci.ViewportState.c(&a)
	}
	if ci.RasterizationState != nil {
		info.pRasterizationState = ci.RasterizationState.c(&a)
	}
	if ci.MultisampleState != nil {
		info.pMultisampleState = ci.MultisampleState.c(&a)
	}
	if ci.DepthStencilState != nil {
		info.pDepthStencilState = ci.DepthStencilState.c(&a)
	}
	if ci.ColorBlendState != nil {
		info.pColorBlendState = ci.ColorBlendState.c(&a)
	}
	if ci.DynamicState != nil {
		info.pDynamicState = ci.DynamicState.c(&a)
	}

	var out C.VkPipeline
	if err := check(C.vkCreateGraphicsPipelines(C.VkDevice(unsafe.Pointer(d)),
		nil, 1, &info, nil, &out)); err != nil {
		return 0, err
	}
	return Pipeline(unsafe.Pointer(out)), nil
}

func DestroyPipeline(d Device, p Pipeline) {
	C.vkDestroyPipeline(C.VkDevice(unsafe.Pointer(d)), C.VkPipeline(unsafe.Pointer(p)), nil)
}
