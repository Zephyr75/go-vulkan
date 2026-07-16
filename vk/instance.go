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

// InstanceCreateInfo is the Go-facing input to CreateInstance.
type InstanceCreateInfo struct {
	AppName    string
	EngineName string
	APIVersion uint32
	Extensions []string
	Layers     []string
}

// CreateInstance creates a VkInstance. Extensions typically come from
// glfw.GetRequiredInstanceExtensions().
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
	app := C.VkApplicationInfo{
		sType:              C.VK_STRUCTURE_TYPE_APPLICATION_INFO,
		pApplicationName:   appName,
		applicationVersion: v100,
		pEngineName:        engName,
		engineVersion:      v100,
		apiVersion:         C.uint32_t(api),
	}

	exts, extCount, freeExts := cStrings(ci.Extensions)
	defer freeExts()
	layers, layerCount, freeLayers := cStrings(ci.Layers)
	defer freeLayers()

	info := C.VkInstanceCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
		pApplicationInfo:        &app,
		enabledExtensionCount:   extCount,
		ppEnabledExtensionNames: exts,
		enabledLayerCount:       layerCount,
		ppEnabledLayerNames:     layers,
	}

	// info.pApplicationInfo points at the Go-stack `app`; pin it for the call.
	var pin runtime.Pinner
	pin.Pin(&app)
	defer pin.Unpin()

	var out C.VkInstance
	if err := check(C.vkCreateInstance(&info, nil, &out)); err != nil {
		return 0, err
	}
	return Instance(unsafe.Pointer(out)), nil
}

func DestroyInstance(i Instance) {
	C.vkDestroyInstance(C.VkInstance(unsafe.Pointer(i)), nil)
}

// EnumeratePhysicalDevices lists all GPUs the instance sees.
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

// PhysicalDeviceProperties is a Goified subset of VkPhysicalDeviceProperties.
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

func GetPhysicalDeviceProperties2(pd PhysicalDevice) PhysicalDeviceProperties {
	var props C.VkPhysicalDeviceProperties2
	props.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2
	C.vkGetPhysicalDeviceProperties2(C.VkPhysicalDevice(unsafe.Pointer(pd)), &props)
	p := props.properties
	return PhysicalDeviceProperties{
		APIVersion:                      uint32(p.apiVersion),
		DriverVersion:                   uint32(p.driverVersion),
		VendorID:                        uint32(p.vendorID),
		DeviceID:                        uint32(p.deviceID),
		DeviceType:                      PhysicalDeviceType(p.deviceType),
		DeviceName:                      C.GoString(&p.deviceName[0]),
		MinUniformBufferOffsetAlignment: uint64(p.limits.minUniformBufferOffsetAlignment),
		MinStorageBufferOffsetAlignment: uint64(p.limits.minStorageBufferOffsetAlignment),
		MaxSamplerAnisotropy:            float32(p.limits.maxSamplerAnisotropy),
	}
}

// QueueFamilyProperties is a Goified subset.
type QueueFamilyProperties struct {
	QueueFlags QueueFlags
	QueueCount uint32
}

func GetPhysicalDeviceQueueFamilyProperties(pd PhysicalDevice) []QueueFamilyProperties {
	dev := C.VkPhysicalDevice(unsafe.Pointer(pd))
	raw := enumerateVoid(func(count *C.uint32_t, out *C.VkQueueFamilyProperties) {
		C.vkGetPhysicalDeviceQueueFamilyProperties(dev, count, out)
	})
	res := make([]QueueFamilyProperties, len(raw))
	for k := range raw {
		res[k] = QueueFamilyProperties{
			QueueFlags: QueueFlags(raw[k].queueFlags),
			QueueCount: uint32(raw[k].queueCount),
		}
	}
	return res
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

func GetPhysicalDeviceMemoryProperties2(pd PhysicalDevice) PhysicalDeviceMemoryProperties {
	var mp C.VkPhysicalDeviceMemoryProperties2
	mp.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_MEMORY_PROPERTIES_2
	C.vkGetPhysicalDeviceMemoryProperties2(C.VkPhysicalDevice(unsafe.Pointer(pd)), &mp)
	m := mp.memoryProperties

	types := make([]MemoryType, m.memoryTypeCount)
	for k := 0; k < int(m.memoryTypeCount); k++ {
		types[k] = MemoryType{
			PropertyFlags: MemoryPropertyFlags(m.memoryTypes[k].propertyFlags),
			HeapIndex:     uint32(m.memoryTypes[k].heapIndex),
		}
	}
	heaps := make([]MemoryHeap, m.memoryHeapCount)
	for k := 0; k < int(m.memoryHeapCount); k++ {
		heaps[k] = MemoryHeap{
			Size:  uint64(m.memoryHeaps[k].size),
			Flags: uint32(m.memoryHeaps[k].flags),
		}
	}
	return PhysicalDeviceMemoryProperties{MemoryTypes: types, MemoryHeaps: heaps}
}
