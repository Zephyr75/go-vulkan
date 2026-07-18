package vk

import "unsafe"

// This file is a small VMA (Vulkan Memory Allocator) substitute exposing the
// same create/destroy shape as vmaCreateAllocator / vmaCreateBuffer /
// vmaCreateImage, built on the raw memory bindings in memory.go. Every
// allocation is dedicated (one VkDeviceMemory per resource: allocate + bind +
// optional persistent map) instead of VMA's sub-allocation, which is all a
// tutorial-scale renderer needs.

// Supported subset of the allocator create flags
type VmaAllocatorCreateFlags uint32

// Gives buffers created with BufferUsageShaderDeviceAddress device-address-capable memory
const VmaAllocatorCreateBufferDeviceAddressBit VmaAllocatorCreateFlags = 1 << 0

type VmaAllocatorCreateInfo struct {
	Flags          VmaAllocatorCreateFlags
	PhysicalDevice PhysicalDevice
	Device         Device
	Instance       Instance
}

// Caches the device and its memory properties so per-resource allocations can pick a memory type
type VmaAllocator struct {
	device   Device
	memProps PhysicalDeviceMemoryProperties
	flags    VmaAllocatorCreateFlags
}

// Supported subset of the allocation create flags
type VmaAllocationCreateFlags uint32

const (
	// Accepted for parity; every allocation here is dedicated already
	VmaAllocationCreateDedicatedMemory VmaAllocationCreateFlags = 1 << 0
	// Keeps the allocation persistently mapped; the pointer is returned in VmaAllocationInfo.MappedData
	VmaAllocationCreateMapped VmaAllocationCreateFlags = 1 << 1
	// Selects CPU-writable memory
	VmaAllocationCreateHostAccessSequentialWrite VmaAllocationCreateFlags = 1 << 2
	// Accepted for parity; this allocator never falls back to a staging transfer
	VmaAllocationCreateHostAccessAllowTransferInstead VmaAllocationCreateFlags = 1 << 3
)

// Only Auto is supported
type VmaMemoryUsage uint32

const VmaMemoryUsageAuto VmaMemoryUsage = 0

type VmaAllocationCreateInfo struct {
	Flags VmaAllocationCreateFlags
	Usage VmaMemoryUsage
}

// Dedicated memory backing one resource, plus its persistent mapping when one was requested
type VmaAllocation struct {
	Memory DeviceMemory
	mapped unsafe.Pointer
}

// Supported subset of VMA's allocation info
type VmaAllocationInfo struct {
	Size       uint64
	MappedData unsafe.Pointer
}

// Creates an allocator caching the device and its memory properties
func VmaCreateAllocator(ci VmaAllocatorCreateInfo) *VmaAllocator {
	return &VmaAllocator{
		device:   ci.Device,
		memProps: GetPhysicalDeviceMemoryProperties2(ci.PhysicalDevice),
		flags:    ci.Flags,
	}
}

// Does nothing; every allocation owns its own memory, so there is nothing to release
func VmaDestroyAllocator(*VmaAllocator) {}

// Creates a buffer with dedicated bound memory, persistently mapped when VmaAllocationCreateMapped is set
func (a *VmaAllocator) VmaCreateBuffer(ci BufferCreateInfo, aci VmaAllocationCreateInfo) (Buffer, VmaAllocation, VmaAllocationInfo, error) {
	buf, err := CreateBuffer(a.device, ci)
	if err != nil {
		return 0, VmaAllocation{}, VmaAllocationInfo{}, err
	}
	req := GetBufferMemoryRequirements(a.device, buf)
	idx, err := a.memoryType(req.MemoryTypeBits, aci.Flags)
	if err != nil {
		DestroyBuffer(a.device, buf)
		return 0, VmaAllocation{}, VmaAllocationInfo{}, err
	}
	mem, err := AllocateMemory(a.device, MemoryAllocateInfo{
		AllocationSize:  req.Size,
		MemoryTypeIndex: idx,
		DeviceAddress:   a.flags&VmaAllocatorCreateBufferDeviceAddressBit != 0 && ci.Usage&BufferUsageShaderDeviceAddress != 0,
	})
	if err != nil {
		DestroyBuffer(a.device, buf)
		return 0, VmaAllocation{}, VmaAllocationInfo{}, err
	}
	if err := BindBufferMemory(a.device, buf, mem, 0); err != nil {
		FreeMemory(a.device, mem)
		DestroyBuffer(a.device, buf)
		return 0, VmaAllocation{}, VmaAllocationInfo{}, err
	}
	var ptr unsafe.Pointer
	if aci.Flags&VmaAllocationCreateMapped != 0 {
		ptr, err = MapMemory(a.device, mem, 0, WholeSize)
		if err != nil {
			FreeMemory(a.device, mem)
			DestroyBuffer(a.device, buf)
			return 0, VmaAllocation{}, VmaAllocationInfo{}, err
		}
	}
	return buf, VmaAllocation{Memory: mem, mapped: ptr}, VmaAllocationInfo{Size: req.Size, MappedData: ptr}, nil
}

// Creates an image with dedicated device-local bound memory
func (a *VmaAllocator) VmaCreateImage(ci ImageCreateInfo, aci VmaAllocationCreateInfo) (Image, VmaAllocation, error) {
	img, err := CreateImage(a.device, ci)
	if err != nil {
		return 0, VmaAllocation{}, err
	}
	req := GetImageMemoryRequirements(a.device, img)
	idx, err := a.memoryType(req.MemoryTypeBits, aci.Flags)
	if err != nil {
		DestroyImage(a.device, img)
		return 0, VmaAllocation{}, err
	}
	mem, err := AllocateMemory(a.device, MemoryAllocateInfo{AllocationSize: req.Size, MemoryTypeIndex: idx})
	if err != nil {
		DestroyImage(a.device, img)
		return 0, VmaAllocation{}, err
	}
	if err := BindImageMemory(a.device, img, mem, 0); err != nil {
		FreeMemory(a.device, mem)
		DestroyImage(a.device, img)
		return 0, VmaAllocation{}, err
	}
	return img, VmaAllocation{Memory: mem}, nil
}

// Unmaps (if mapped), destroys the buffer, and frees its memory
func (a *VmaAllocator) VmaDestroyBuffer(b Buffer, al VmaAllocation) {
	if al.mapped != nil {
		UnmapMemory(a.device, al.Memory)
	}
	DestroyBuffer(a.device, b)
	FreeMemory(a.device, al.Memory)
}

// Destroys the image and frees its memory
func (a *VmaAllocator) VmaDestroyImage(img Image, al VmaAllocation) {
	DestroyImage(a.device, img)
	FreeMemory(a.device, al.Memory)
}

// Picks a memory type from the requirement mask: host-writable requests prefer ReBAR-style DEVICE_LOCAL|HOST_VISIBLE|HOST_COHERENT then plain host-visible, others take DEVICE_LOCAL
func (a *VmaAllocator) memoryType(bits uint32, flags VmaAllocationCreateFlags) (uint32, error) {
	if flags&VmaAllocationCreateHostAccessSequentialWrite != 0 {
		if idx, err := FindMemoryType(a.memProps, bits,
			MemoryPropertyDeviceLocal|MemoryPropertyHostVisible|MemoryPropertyHostCoherent); err == nil {
			return idx, nil
		}
		return FindMemoryType(a.memProps, bits, MemoryPropertyHostVisible|MemoryPropertyHostCoherent)
	}
	return FindMemoryType(a.memProps, bits, MemoryPropertyDeviceLocal)
}
