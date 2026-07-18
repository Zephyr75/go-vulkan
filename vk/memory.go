package vk

/*
#include <vulkan/vulkan.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// MemoryRequirements is a Goified subset of VkMemoryRequirements.
type MemoryRequirements struct {
	Size           uint64
	Alignment      uint64
	MemoryTypeBits uint32
}

func GetBufferMemoryRequirements(d Device, b Buffer) MemoryRequirements {
	var r C.VkMemoryRequirements
	C.vkGetBufferMemoryRequirements(C.VkDevice(unsafe.Pointer(d)), C.VkBuffer(unsafe.Pointer(b)), &r)
	return MemoryRequirements{uint64(r.size), uint64(r.alignment), uint32(r.memoryTypeBits)}
}

func GetImageMemoryRequirements(d Device, img Image) MemoryRequirements {
	var r C.VkMemoryRequirements
	C.vkGetImageMemoryRequirements(C.VkDevice(unsafe.Pointer(d)), C.VkImage(unsafe.Pointer(img)), &r)
	return MemoryRequirements{uint64(r.size), uint64(r.alignment), uint32(r.memoryTypeBits)}
}

// MemoryAllocateInfo is the Go-facing input to AllocateMemory. Set
// DeviceAddress for allocations backing buffer-device-address buffers; it
// chains VkMemoryAllocateFlagsInfo with the DEVICE_ADDRESS bit.
type MemoryAllocateInfo struct {
	AllocationSize  uint64
	MemoryTypeIndex uint32
	DeviceAddress   bool
}

// Allocates device memory, chaining the DEVICE_ADDRESS flag when requested
func AllocateMemory(d Device, ai MemoryAllocateInfo) (DeviceMemory, error) {
	info := C.VkMemoryAllocateInfo{
		sType:           C.VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		allocationSize:  C.VkDeviceSize(ai.AllocationSize),
		memoryTypeIndex: C.uint32_t(ai.MemoryTypeIndex),
	}
	var pin runtime.Pinner
	defer pin.Unpin()
	if ai.DeviceAddress {
		flags := C.VkMemoryAllocateFlagsInfo{
			sType: C.VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_FLAGS_INFO,
			flags: C.VK_MEMORY_ALLOCATE_DEVICE_ADDRESS_BIT,
		}
		pin.Pin(&flags)
		info.pNext = unsafe.Pointer(&flags)
	}
	var out C.VkDeviceMemory
	if err := check(C.vkAllocateMemory(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return DeviceMemory(unsafe.Pointer(out)), nil
}

func FreeMemory(d Device, m DeviceMemory) {
	C.vkFreeMemory(C.VkDevice(unsafe.Pointer(d)), C.VkDeviceMemory(unsafe.Pointer(m)), nil)
}

func BindBufferMemory(d Device, b Buffer, m DeviceMemory, offset uint64) error {
	return check(C.vkBindBufferMemory(C.VkDevice(unsafe.Pointer(d)),
		C.VkBuffer(unsafe.Pointer(b)), C.VkDeviceMemory(unsafe.Pointer(m)), C.VkDeviceSize(offset)))
}

func BindImageMemory(d Device, img Image, m DeviceMemory, offset uint64) error {
	return check(C.vkBindImageMemory(C.VkDevice(unsafe.Pointer(d)),
		C.VkImage(unsafe.Pointer(img)), C.VkDeviceMemory(unsafe.Pointer(m)), C.VkDeviceSize(offset)))
}

// Maps the range and returns a host pointer; write through MemCopy
func MapMemory(d Device, m DeviceMemory, offset, size uint64) (unsafe.Pointer, error) {
	var p unsafe.Pointer
	err := check(C.vkMapMemory(C.VkDevice(unsafe.Pointer(d)),
		C.VkDeviceMemory(unsafe.Pointer(m)), C.VkDeviceSize(offset), C.VkDeviceSize(size), 0, &p))
	if err != nil {
		return nil, err
	}
	return p, nil
}

func UnmapMemory(d Device, m DeviceMemory) {
	C.vkUnmapMemory(C.VkDevice(unsafe.Pointer(d)), C.VkDeviceMemory(unsafe.Pointer(m)))
}

// Returns the GPU address of a buffer created with BufferUsageShaderDeviceAddress on BDA-capable memory
func GetBufferDeviceAddress(d Device, b Buffer) uint64 {
	info := C.VkBufferDeviceAddressInfo{
		sType:  C.VK_STRUCTURE_TYPE_BUFFER_DEVICE_ADDRESS_INFO,
		buffer: C.VkBuffer(unsafe.Pointer(b)),
	}
	return uint64(C.vkGetBufferDeviceAddress(C.VkDevice(unsafe.Pointer(d)), &info))
}

// Picks a memory type index allowed by typeBits whose properties include all of want
func FindMemoryType(props PhysicalDeviceMemoryProperties, typeBits uint32, want MemoryPropertyFlags) (uint32, error) {
	for i, mt := range props.MemoryTypes {
		if typeBits&(1<<uint(i)) == 0 {
			continue
		}
		if mt.PropertyFlags&want == want {
			return uint32(i), nil
		}
	}
	return 0, fmt.Errorf("vk: no memory type for bits %#x with props %#x", typeBits, uint32(want))
}
