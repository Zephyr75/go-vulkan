package vk

/*
#include <stdlib.h>
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

// ComputePipelineCreateInfo is the whole of a compute pipeline: one stage and a
// layout. No vertex input, no blend state, no rendering info
type ComputePipelineCreateInfo struct {
	Layout PipelineLayout
	Stage  PipelineShaderStageCreateInfo
}

// Creates a single compute pipeline; DestroyPipeline tears it down like a graphics one
func CreateComputePipeline(d Device, ci ComputePipelineCreateInfo) (Pipeline, error) {
	var a arena
	defer a.free()

	info := C.VkComputePipelineCreateInfo{
		sType:             C.VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO,
		layout:            C.VkPipelineLayout(unsafe.Pointer(ci.Layout)),
		basePipelineIndex: -1,
	}
	info.stage = C.VkPipelineShaderStageCreateInfo{
		sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
		stage:  C.VkShaderStageFlagBits(ci.Stage.Stage),
		module: C.VkShaderModule(unsafe.Pointer(ci.Stage.Module)),
		pName:  a.cstr(ci.Stage.Name),
	}

	var out C.VkPipeline
	if err := check(C.vkCreateComputePipelines(C.VkDevice(unsafe.Pointer(d)),
		nil, 1, &info, nil, &out)); err != nil {
		return 0, err
	}
	return Pipeline(unsafe.Pointer(out)), nil
}
