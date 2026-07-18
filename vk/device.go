package vk

/*
#include <stdlib.h>
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

// Physical device features
type Features struct {
	// 1.0
	SamplerAnisotropy bool
	// 1.2
	BufferDeviceAddress                          bool
	DescriptorIndexing                           bool
	RuntimeDescriptorArray                       bool
	DescriptorBindingPartiallyBound              bool
	DescriptorBindingVariableDescriptorCount     bool
	ShaderSampledImageArrayNonUniformIndexing    bool
	DescriptorBindingSampledImageUpdateAfterBind bool
	// 1.3
	DynamicRendering bool
	Synchronization2 bool
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

// Builds the features pNext chain (Features2 -> Vulkan12 -> Vulkan13) in the arena and returns the head
func (f Features) chain(a *arena) *C.VkPhysicalDeviceFeatures2 {
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

// Converts local DeviceQueueCreateInfo to vulkan format
func vulkanQueueCreateInfo(a *arena, ci []DeviceQueueCreateInfo) (*C.VkDeviceQueueCreateInfo, C.uint32_t) {
	infoCount := len(ci)
	if infoCount == 0 {
		return nil, 0
	}
	vkInfoAllocPtr := (*C.VkDeviceQueueCreateInfo)(a.alloc(infoCount, unsafe.Sizeof(C.VkDeviceQueueCreateInfo{})))
	vkInfoSlice := unsafe.Slice(vkInfoAllocPtr, infoCount)
	for i, q := range ci {
		prioCount := len(q.Priorities)
		prioAllocPtr := (*C.float)(a.alloc(prioCount, unsafe.Sizeof(C.float(0))))
		prioSlice := unsafe.Slice(prioAllocPtr, prioCount)
		for k, v := range q.Priorities {
			prioSlice[k] = C.float(v)
		}
		vkInfoSlice[i].sType = C.VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO
		vkInfoSlice[i].queueFamilyIndex = C.uint32_t(q.QueueFamilyIndex)
		vkInfoSlice[i].queueCount = C.uint32_t(prioCount)
		vkInfoSlice[i].pQueuePriorities = prioAllocPtr
	}
	return vkInfoAllocPtr, C.uint32_t(infoCount)
}

// Builds the logical device
func CreateDevice(pd PhysicalDevice, ci DeviceCreateInfo) (Device, error) {
	var a arena
	defer a.free()

	extensions, extensionsCount, freeExtensions := cStrings(ci.Extensions)
	defer freeExtensions()

	qInfos, qCount := vulkanQueueCreateInfo(&a, ci.QueueCreateInfos)
	info := C.VkDeviceCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
		pNext:                   unsafe.Pointer(ci.Features.chain(&a)),
		queueCreateInfoCount:    qCount,
		pQueueCreateInfos:       qInfos,
		enabledExtensionCount:   extensionsCount,
		ppEnabledExtensionNames: extensions,
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
	var queue C.VkQueue
	C.vkGetDeviceQueue(C.VkDevice(unsafe.Pointer(d)), C.uint32_t(family), C.uint32_t(index), &queue)
	return Queue(unsafe.Pointer(queue))
}

func DeviceWaitIdle(d Device) error {
	return check(C.vkDeviceWaitIdle(C.VkDevice(unsafe.Pointer(d))))
}
