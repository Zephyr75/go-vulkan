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

type InstanceCreateInfo struct {
	AppName    string
	EngineName string
	APIVersion uint32
	Extensions []string
	Layers     []string
}

// Creates instance from application, extensions and layers information
func CreateInstance(ci InstanceCreateInfo) (Instance, error) {
	appName := C.CString(ci.AppName)
	defer C.free(unsafe.Pointer(appName))
	engName := C.CString(ci.EngineName)
	defer C.free(unsafe.Pointer(engName))

	api := ci.APIVersion
	if api == 0 {
		api = ApiVersion13
	}

	const v100 = 1 << 22 // VK_MAKE_API_VERSION(0,1,0,0)
	vkAppInfo := C.VkApplicationInfo{
		sType:              C.VK_STRUCTURE_TYPE_APPLICATION_INFO,
		pApplicationName:   appName,
		applicationVersion: v100,
		pEngineName:        engName,
		engineVersion:      v100,
		apiVersion:         C.uint32_t(api),
	}

	extensions, extensionsCount, freeExtensions := cStrings(ci.Extensions)
	defer freeExtensions()
	layers, layerCount, freeLayers := cStrings(ci.Layers)
	defer freeLayers()

	vkInfo := C.VkInstanceCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
		pApplicationInfo:        &vkAppInfo,
		enabledExtensionCount:   extensionsCount,
		ppEnabledExtensionNames: extensions,
		enabledLayerCount:       layerCount,
		ppEnabledLayerNames:     layers,
	}

	// app is stored on Go stack and could move, pin keeps it at a constant address
	var pin runtime.Pinner
	pin.Pin(&vkAppInfo)
	defer pin.Unpin()

	var vkInstance C.VkInstance
	if err := check(C.vkCreateInstance(&vkInfo, nil, &vkInstance)); err != nil {
		return 0, err
	}
	return Instance(unsafe.Pointer(vkInstance)), nil
}

func DestroyInstance(i Instance) {
	C.vkDestroyInstance(C.VkInstance(unsafe.Pointer(i)), nil)
}

// Lists all GPUs the instance sees
func EnumeratePhysicalDevices(i Instance) ([]PhysicalDevice, error) {
	inst := C.VkInstance(unsafe.Pointer(i))
	handles, err := enumerate(func(count *C.uint32_t, out *C.VkPhysicalDevice) C.VkResult {
		return C.vkEnumeratePhysicalDevices(inst, count, out)
	})
	if err != nil {
		return nil, err
	}
	res := make([]PhysicalDevice, len(handles))
	for k, h := range handles {
		res[k] = PhysicalDevice(unsafe.Pointer(h))
	}
	return res, nil
}

type PhysicalDeviceProperties struct {
	APIVersion    uint32
	DriverVersion uint32
	VendorID      uint32
	DeviceID      uint32
	DeviceType    PhysicalDeviceType
	DeviceName    string

	MinUniformBufferOffsetAlignment uint64
	MinStorageBufferOffsetAlignment uint64
	MaxSamplerAnisotropy            float32
}

// Queries a GPU's identity (name, IDs, type) and the limits the demo needs
func GetPhysicalDeviceProperties2(pd PhysicalDevice) PhysicalDeviceProperties {
	var vkDeviceProperties C.VkPhysicalDeviceProperties2
	vkDeviceProperties.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2
	C.vkGetPhysicalDeviceProperties2(C.VkPhysicalDevice(unsafe.Pointer(pd)), &vkDeviceProperties)
	properties := vkDeviceProperties.properties
	return PhysicalDeviceProperties{
		APIVersion:                      uint32(properties.apiVersion),
		DriverVersion:                   uint32(properties.driverVersion),
		VendorID:                        uint32(properties.vendorID),
		DeviceID:                        uint32(properties.deviceID),
		DeviceType:                      PhysicalDeviceType(properties.deviceType),
		DeviceName:                      C.GoString(&properties.deviceName[0]),
		MinUniformBufferOffsetAlignment: uint64(properties.limits.minUniformBufferOffsetAlignment),
		MinStorageBufferOffsetAlignment: uint64(properties.limits.minStorageBufferOffsetAlignment),
		MaxSamplerAnisotropy:            float32(properties.limits.maxSamplerAnisotropy),
	}
}

type QueueFamilyProperties struct {
	QueueFlags QueueFlags
	QueueCount uint32
}

// Lists the queue families of a physical device
func GetPhysicalDeviceQueueFamilyProperties(pd PhysicalDevice) []QueueFamilyProperties {
	vkDevice := C.VkPhysicalDevice(unsafe.Pointer(pd))
	raw := enumerateVoid(func(count *C.uint32_t, out *C.VkQueueFamilyProperties) {
		C.vkGetPhysicalDeviceQueueFamilyProperties(vkDevice, count, out)
	})
	result := make([]QueueFamilyProperties, len(raw))
	for k := range raw {
		result[k] = QueueFamilyProperties{
			QueueFlags: QueueFlags(raw[k].queueFlags),
			QueueCount: uint32(raw[k].queueCount),
		}
	}
	return result
}

// MemoryType / MemoryHeap / PhysicalDeviceMemoryProperties for manual alloc.
type MemoryType struct {
	PropertyFlags MemoryPropertyFlags
	HeapIndex     uint32
}
type MemoryHeap struct {
	Size  uint64
	Flags uint32
}
type PhysicalDeviceMemoryProperties struct {
	MemoryTypes []MemoryType
	MemoryHeaps []MemoryHeap
}

// Queries the memory types and heaps of a physical device
func GetPhysicalDeviceMemoryProperties2(pd PhysicalDevice) PhysicalDeviceMemoryProperties {
	var vkMemoryProperties C.VkPhysicalDeviceMemoryProperties2
	vkMemoryProperties.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_MEMORY_PROPERTIES_2
	C.vkGetPhysicalDeviceMemoryProperties2(C.VkPhysicalDevice(unsafe.Pointer(pd)), &vkMemoryProperties)
	vkProperties := vkMemoryProperties.memoryProperties

	types := make([]MemoryType, vkProperties.memoryTypeCount)
	for k := 0; k < int(vkProperties.memoryTypeCount); k++ {
		types[k] = MemoryType{
			PropertyFlags: MemoryPropertyFlags(vkProperties.memoryTypes[k].propertyFlags),
			HeapIndex:     uint32(vkProperties.memoryTypes[k].heapIndex),
		}
	}
	heaps := make([]MemoryHeap, vkProperties.memoryHeapCount)
	for k := 0; k < int(vkProperties.memoryHeapCount); k++ {
		heaps[k] = MemoryHeap{
			Size:  uint64(vkProperties.memoryHeaps[k].size),
			Flags: uint32(vkProperties.memoryHeaps[k].flags),
		}
	}
	return PhysicalDeviceMemoryProperties{MemoryTypes: types, MemoryHeaps: heaps}
}
