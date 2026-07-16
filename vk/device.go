package vk

/*
#include <stdlib.h>
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

// Features toggles the feature bits the tutorial needs, spread across the
// Vulkan 1.0 / 1.2 / 1.3 feature structs. CreateDevice builds the pNext chain.
type Features struct {
	// 1.0
	SamplerAnisotropy bool
	// 1.3
	DynamicRendering bool
	Synchronization2 bool
	// 1.2
	BufferDeviceAddress                          bool
	DescriptorIndexing                           bool
	RuntimeDescriptorArray                       bool
	DescriptorBindingPartiallyBound              bool
	DescriptorBindingVariableDescriptorCount     bool
	ShaderSampledImageArrayNonUniformIndexing    bool
	DescriptorBindingSampledImageUpdateAfterBind bool
}

type DeviceQueueCreateInfo struct {
	QueueFamilyIndex uint32
	Priorities       []float32
}

type DeviceCreateInfo struct {
	QueueCreateInfos []DeviceQueueCreateInfo
	Extensions       []string
	Features         Features
}

func vkBool(b bool) C.VkBool32 {
	if b {
		return C.VK_TRUE
	}
	return C.VK_FALSE
}

// c builds the feature pNext chain (Features2 -> Vulkan12 -> Vulkan13) in the
// arena and returns the head to hang off VkDeviceCreateInfo.pNext.
func (f Features) c(a *arena) *C.VkPhysicalDeviceFeatures2 {
	v13 := (*C.VkPhysicalDeviceVulkan13Features)(a.alloc(1, unsafe.Sizeof(C.VkPhysicalDeviceVulkan13Features{})))
	v13.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_3_FEATURES
	v13.dynamicRendering = vkBool(f.DynamicRendering)
	v13.synchronization2 = vkBool(f.Synchronization2)

	v12 := (*C.VkPhysicalDeviceVulkan12Features)(a.alloc(1, unsafe.Sizeof(C.VkPhysicalDeviceVulkan12Features{})))
	v12.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_2_FEATURES
	v12.pNext = unsafe.Pointer(v13)
	v12.bufferDeviceAddress = vkBool(f.BufferDeviceAddress)
	v12.descriptorIndexing = vkBool(f.DescriptorIndexing)
	v12.runtimeDescriptorArray = vkBool(f.RuntimeDescriptorArray)
	v12.descriptorBindingPartiallyBound = vkBool(f.DescriptorBindingPartiallyBound)
	v12.descriptorBindingVariableDescriptorCount = vkBool(f.DescriptorBindingVariableDescriptorCount)
	v12.shaderSampledImageArrayNonUniformIndexing = vkBool(f.ShaderSampledImageArrayNonUniformIndexing)
	v12.descriptorBindingSampledImageUpdateAfterBind = vkBool(f.DescriptorBindingSampledImageUpdateAfterBind)

	feat2 := (*C.VkPhysicalDeviceFeatures2)(a.alloc(1, unsafe.Sizeof(C.VkPhysicalDeviceFeatures2{})))
	feat2.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2
	feat2.pNext = unsafe.Pointer(v12)
	feat2.features.samplerAnisotropy = vkBool(f.SamplerAnisotropy)
	return feat2
}

// queueInfosC marshals the queue create infos (with their priority arrays) into
// the arena.
func queueInfosC(a *arena, in []DeviceQueueCreateInfo) (*C.VkDeviceQueueCreateInfo, C.uint32_t) {
	n := len(in)
	if n == 0 {
		return nil, 0
	}
	p := (*C.VkDeviceQueueCreateInfo)(a.alloc(n, unsafe.Sizeof(C.VkDeviceQueueCreateInfo{})))
	qs := unsafe.Slice(p, n)
	for i, q := range in {
		pc := len(q.Priorities)
		pr := (*C.float)(a.alloc(pc, unsafe.Sizeof(C.float(0))))
		prs := unsafe.Slice(pr, pc)
		for k, v := range q.Priorities {
			prs[k] = C.float(v)
		}
		qs[i].sType = C.VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO
		qs[i].queueFamilyIndex = C.uint32_t(q.QueueFamilyIndex)
		qs[i].queueCount = C.uint32_t(pc)
		qs[i].pQueuePriorities = pr
	}
	return p, C.uint32_t(n)
}

// CreateDevice builds the logical device. The feature pNext chain and queue
// infos are marshaled by the helpers above; all C-visible memory lives in the
// arena so no Go pointers are held by C across the call.
func CreateDevice(pd PhysicalDevice, ci DeviceCreateInfo) (Device, error) {
	var a arena
	defer a.free()

	exts, extCount, freeExts := cStrings(ci.Extensions)
	defer freeExts()

	qInfos, qCount := queueInfosC(&a, ci.QueueCreateInfos)
	info := C.VkDeviceCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
		pNext:                   unsafe.Pointer(ci.Features.c(&a)),
		queueCreateInfoCount:    qCount,
		pQueueCreateInfos:       qInfos,
		enabledExtensionCount:   extCount,
		ppEnabledExtensionNames: exts,
	}

	var out C.VkDevice
	if err := check(C.vkCreateDevice(C.VkPhysicalDevice(unsafe.Pointer(pd)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return Device(unsafe.Pointer(out)), nil
}

func DestroyDevice(d Device) {
	C.vkDestroyDevice(C.VkDevice(unsafe.Pointer(d)), nil)
}

func GetDeviceQueue(d Device, family, index uint32) Queue {
	var q C.VkQueue
	C.vkGetDeviceQueue(C.VkDevice(unsafe.Pointer(d)), C.uint32_t(family), C.uint32_t(index), &q)
	return Queue(unsafe.Pointer(q))
}

func DeviceWaitIdle(d Device) error {
	return check(C.vkDeviceWaitIdle(C.VkDevice(unsafe.Pointer(d))))
}
