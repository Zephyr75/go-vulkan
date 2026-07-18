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

// DescriptorSetLayoutBinding is one binding slot. BindingFlags is consumed only
// when the create info sets UseBindingFlags (descriptor indexing).
type DescriptorSetLayoutBinding struct {
	Binding         uint32
	DescriptorType  DescriptorType
	DescriptorCount uint32
	StageFlags      ShaderStageFlags
	BindingFlags    DescriptorBindingFlags
}

// DescriptorSetLayoutCreateInfo is the Go-facing input to
// CreateDescriptorSetLayout. Set UseBindingFlags to chain
// VkDescriptorSetLayoutBindingFlagsCreateInfo (variable descriptor count,
// partially bound, update-after-bind).
type DescriptorSetLayoutCreateInfo struct {
	Flags           DescriptorSetLayoutCreateFlags
	Bindings        []DescriptorSetLayoutBinding
	UseBindingFlags bool
}

// Marshals the binding slice into the arena
func bindingsC(a *arena, in []DescriptorSetLayoutBinding) *C.VkDescriptorSetLayoutBinding {
	n := len(in)
	if n == 0 {
		return nil
	}
	p := (*C.VkDescriptorSetLayoutBinding)(a.alloc(n, unsafe.Sizeof(C.VkDescriptorSetLayoutBinding{})))
	bs := unsafe.Slice(p, n)
	for i, b := range in {
		bs[i].binding = C.uint32_t(b.Binding)
		bs[i].descriptorType = C.VkDescriptorType(b.DescriptorType)
		bs[i].descriptorCount = C.uint32_t(b.DescriptorCount)
		bs[i].stageFlags = C.VkShaderStageFlags(b.StageFlags)
	}
	return p
}

// Builds the VkDescriptorSetLayoutBindingFlagsCreateInfo pNext (descriptor indexing) from each binding's BindingFlags
func bindingFlagsC(a *arena, in []DescriptorSetLayoutBinding) *C.VkDescriptorSetLayoutBindingFlagsCreateInfo {
	n := len(in)
	bf := (*C.VkDescriptorBindingFlags)(a.alloc(n, unsafe.Sizeof(C.VkDescriptorBindingFlags(0))))
	fs := unsafe.Slice(bf, n)
	for i, b := range in {
		fs[i] = C.VkDescriptorBindingFlags(b.BindingFlags)
	}
	out := (*C.VkDescriptorSetLayoutBindingFlagsCreateInfo)(a.alloc(1, unsafe.Sizeof(C.VkDescriptorSetLayoutBindingFlagsCreateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_BINDING_FLAGS_CREATE_INFO
	out.bindingCount = C.uint32_t(n)
	out.pBindingFlags = bf
	return out
}

// Creates a descriptor set layout, chaining per-binding flags when UseBindingFlags is set
func CreateDescriptorSetLayout(d Device, ci DescriptorSetLayoutCreateInfo) (DescriptorSetLayout, error) {
	var a arena
	defer a.free()

	info := C.VkDescriptorSetLayoutCreateInfo{
		sType:        C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
		flags:        C.VkDescriptorSetLayoutCreateFlags(ci.Flags),
		bindingCount: C.uint32_t(len(ci.Bindings)),
		pBindings:    bindingsC(&a, ci.Bindings),
	}
	if ci.UseBindingFlags {
		info.pNext = unsafe.Pointer(bindingFlagsC(&a, ci.Bindings))
	}

	var out C.VkDescriptorSetLayout
	if err := check(C.vkCreateDescriptorSetLayout(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return DescriptorSetLayout(unsafe.Pointer(out)), nil
}

func DestroyDescriptorSetLayout(d Device, l DescriptorSetLayout) {
	C.vkDestroyDescriptorSetLayout(C.VkDevice(unsafe.Pointer(d)), C.VkDescriptorSetLayout(unsafe.Pointer(l)), nil)
}

// DescriptorPoolSize reserves DescriptorCount descriptors of a type.
type DescriptorPoolSize struct {
	Type            DescriptorType
	DescriptorCount uint32
}

type DescriptorPoolCreateInfo struct {
	Flags     DescriptorPoolCreateFlags
	MaxSets   uint32
	PoolSizes []DescriptorPoolSize
}

// Creates a descriptor pool sized by the pool sizes
func CreateDescriptorPool(d Device, ci DescriptorPoolCreateInfo) (DescriptorPool, error) {
	n := len(ci.PoolSizes)
	sizes := (*C.VkDescriptorPoolSize)(C.calloc(C.size_t(n), C.size_t(unsafe.Sizeof(C.VkDescriptorPoolSize{}))))
	defer C.free(unsafe.Pointer(sizes))
	ss := unsafe.Slice(sizes, n)
	for i, s := range ci.PoolSizes {
		ss[i]._type = C.VkDescriptorType(s.Type)
		ss[i].descriptorCount = C.uint32_t(s.DescriptorCount)
	}
	info := C.VkDescriptorPoolCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
		flags:         C.VkDescriptorPoolCreateFlags(ci.Flags),
		maxSets:       C.uint32_t(ci.MaxSets),
		poolSizeCount: C.uint32_t(n),
		pPoolSizes:    sizes,
	}
	var out C.VkDescriptorPool
	if err := check(C.vkCreateDescriptorPool(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return DescriptorPool(unsafe.Pointer(out)), nil
}

func DestroyDescriptorPool(d Device, p DescriptorPool) {
	C.vkDestroyDescriptorPool(C.VkDevice(unsafe.Pointer(d)), C.VkDescriptorPool(unsafe.Pointer(p)), nil)
}

// DescriptorSetAllocateInfo allocates one set per layout. VariableCounts, when
// non-nil, must have the same length as Layouts and chains
// VkDescriptorSetVariableDescriptorCountAllocateInfo (the final size of a
// variable-count binding, e.g. the number of bindless textures).
type DescriptorSetAllocateInfo struct {
	Pool           DescriptorPool
	Layouts        []DescriptorSetLayout
	VariableCounts []uint32
}

// Builds the VkDescriptorSetVariableDescriptorCountAllocateInfo pNext (final size of each variable-count binding)
func variableCountsC(a *arena, counts []uint32) *C.VkDescriptorSetVariableDescriptorCountAllocateInfo {
	n := len(counts)
	vc := (*C.uint32_t)(a.alloc(n, unsafe.Sizeof(C.uint32_t(0))))
	vs := unsafe.Slice(vc, n)
	for i, c := range counts {
		vs[i] = C.uint32_t(c)
	}
	out := (*C.VkDescriptorSetVariableDescriptorCountAllocateInfo)(a.alloc(1, unsafe.Sizeof(C.VkDescriptorSetVariableDescriptorCountAllocateInfo{})))
	out.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_VARIABLE_DESCRIPTOR_COUNT_ALLOCATE_INFO
	out.descriptorSetCount = C.uint32_t(n)
	out.pDescriptorCounts = vc
	return out
}

// Allocates one descriptor set per layout, sizing variable-count bindings from VariableCounts
func AllocateDescriptorSets(d Device, ai DescriptorSetAllocateInfo) ([]DescriptorSet, error) {
	n := len(ai.Layouts)
	if n == 0 {
		return nil, nil
	}
	var a arena
	defer a.free()

	// pSetLayouts points at the Go-owned Layouts slice; pin it for the call so
	// cgo's "Go pointer in C-passed struct" check is satisfied.
	var pin runtime.Pinner
	defer pin.Unpin()
	pin.Pin(&ai.Layouts[0])

	info := C.VkDescriptorSetAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
		descriptorPool:     C.VkDescriptorPool(unsafe.Pointer(ai.Pool)),
		descriptorSetCount: C.uint32_t(n),
		pSetLayouts:        (*C.VkDescriptorSetLayout)(unsafe.Pointer(&ai.Layouts[0])),
	}
	if len(ai.VariableCounts) > 0 {
		info.pNext = unsafe.Pointer(variableCountsC(&a, ai.VariableCounts))
	}

	out := make([]DescriptorSet, n)
	if err := check(C.vkAllocateDescriptorSets(C.VkDevice(unsafe.Pointer(d)), &info,
		(*C.VkDescriptorSet)(unsafe.Pointer(&out[0])))); err != nil {
		return nil, err
	}
	return out, nil
}

// DescriptorImageInfo binds a sampler+view+layout into a combined-image-sampler
// (or sampled-image) descriptor.
type DescriptorImageInfo struct {
	Sampler     Sampler
	ImageView   ImageView
	ImageLayout ImageLayout
}

// WriteDescriptorSet updates one (or a contiguous run of) descriptors from
// ImageInfo. Buffer descriptors are not modeled: the tutorial passes buffers by
// device address through push constants instead.
type WriteDescriptorSet struct {
	DstSet          DescriptorSet
	DstBinding      uint32
	DstArrayElement uint32
	DescriptorType  DescriptorType
	ImageInfo       []DescriptorImageInfo
}

// Marshals the image descriptor array into the arena
func imageInfoC(a *arena, in []DescriptorImageInfo) *C.VkDescriptorImageInfo {
	n := len(in)
	if n == 0 {
		return nil
	}
	p := (*C.VkDescriptorImageInfo)(a.alloc(n, unsafe.Sizeof(C.VkDescriptorImageInfo{})))
	is := unsafe.Slice(p, n)
	for k, im := range in {
		is[k].sampler = C.VkSampler(unsafe.Pointer(im.Sampler))
		is[k].imageView = C.VkImageView(unsafe.Pointer(im.ImageView))
		is[k].imageLayout = C.VkImageLayout(im.ImageLayout)
	}
	return p
}

// Writes image descriptors into their sets in one vkUpdateDescriptorSets call
func UpdateDescriptorSets(d Device, writes []WriteDescriptorSet) {
	if len(writes) == 0 {
		return
	}
	var a arena
	defer a.free()

	n := len(writes)
	wArr := (*C.VkWriteDescriptorSet)(a.alloc(n, unsafe.Sizeof(C.VkWriteDescriptorSet{})))
	ws := unsafe.Slice(wArr, n)
	for i, w := range writes {
		ws[i].sType = C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET
		ws[i].dstSet = C.VkDescriptorSet(unsafe.Pointer(w.DstSet))
		ws[i].dstBinding = C.uint32_t(w.DstBinding)
		ws[i].dstArrayElement = C.uint32_t(w.DstArrayElement)
		ws[i].descriptorType = C.VkDescriptorType(w.DescriptorType)
		ws[i].descriptorCount = C.uint32_t(len(w.ImageInfo))
		ws[i].pImageInfo = imageInfoC(&a, w.ImageInfo)
	}
	C.vkUpdateDescriptorSets(C.VkDevice(unsafe.Pointer(d)), C.uint32_t(n), wArr, 0, nil)
}
