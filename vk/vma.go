package vk

import "unsafe"

// This file is a small VMA (Vulkan Memory Allocator) substitute exposing the
// same create/destroy shape as vmaCreateAllocator / vmaCreateBuffer /
// vmaCreateImage, built on the raw memory bindings in memory.go. Every
// allocation is dedicated (one VkDeviceMemory per resource: allocate + bind +
// optional persistent map) instead of VMA's sub-allocation, which is all a
// tutorial-scale renderer needs.

// VmaAllocatorCreateFlags mirrors VmaAllocatorCreateFlags (supported subset).
type VmaAllocatorCreateFlags uint32

// VmaAllocatorCreateBufferDeviceAddressBit mirrors
// VMA_ALLOCATOR_CREATE_BUFFER_DEVICE_ADDRESS_BIT: buffers created with
// BufferUsageShaderDeviceAddress get device-address-capable memory.
const VmaAllocatorCreateBufferDeviceAddressBit VmaAllocatorCreateFlags = 1 << 0

// VmaAllocatorCreateInfo mirrors VmaAllocatorCreateInfo.
type VmaAllocatorCreateInfo struct {
	Flags          VmaAllocatorCreateFlags
	PhysicalDevice PhysicalDevice
	Device         Device
	Instance       Instance
}

// VmaAllocator stands in for VmaAllocator: it caches the device and its memory
// properties so per-resource allocations can pick a memory type.
type VmaAllocator struct {
	device   Device
	memProps PhysicalDeviceMemoryProperties
	flags    VmaAllocatorCreateFlags
}

// VmaAllocationCreateFlags mirrors VmaAllocationCreateFlags (supported subset).
type VmaAllocationCreateFlags uint32

const (
	// VmaAllocationCreateDedicatedMemory is accepted for parity with
	// VMA_ALLOCATION_CREATE_DEDICATED_MEMORY_BIT; every allocation here is
	// dedicated already.
	VmaAllocationCreateDedicatedMemory VmaAllocationCreateFlags = 1 << 0
	// VmaAllocationCreateMapped keeps the allocation persistently mapped; the
	// pointer is returned in AllocationInfo.MappedData.
	VmaAllocationCreateMapped VmaAllocationCreateFlags = 1 << 1
	// VmaAllocationCreateHostAccessSequentialWrite selects CPU-writable memory.
	VmaAllocationCreateHostAccessSequentialWrite VmaAllocationCreateFlags = 1 << 2
	// VmaAllocationCreateHostAccessAllowTransferInstead is accepted for parity;
	// this allocator never falls back to a staging transfer.
	VmaAllocationCreateHostAccessAllowTransferInstead VmaAllocationCreateFlags = 1 << 3
)

// VmaMemoryUsage mirrors VmaMemoryUsage; only Auto is supported.
type VmaMemoryUsage uint32

// VmaMemoryUsageAuto mirrors VMA_MEMORY_USAGE_AUTO.
const VmaMemoryUsageAuto VmaMemoryUsage = 0

// VmaAllocationCreateInfo mirrors VmaAllocationCreateInfo.
type VmaAllocationCreateInfo struct {
	Flags VmaAllocationCreateFlags
	Usage VmaMemoryUsage
}

// VmaAllocation stands in for VmaAllocation: the dedicated memory backing one
// resource, plus its persistent mapping when one was requested.
type VmaAllocation struct {
	Memory DeviceMemory
	mapped unsafe.Pointer
}

// VmaAllocationInfo mirrors VmaAllocationInfo (supported subset).
type VmaAllocationInfo struct {
	Size       uint64
	MappedData unsafe.Pointer
}

func VmaCreateAllocator(ci VmaAllocatorCreateInfo) *VmaAllocator {
	return &VmaAllocator{
		device:   ci.Device,
		memProps: GetPhysicalDeviceMemoryProperties2(ci.PhysicalDevice),
		flags:    ci.Flags,
	}
}

func VmaDestroyAllocator(*VmaAllocator) {}

// VmaCreateBuffer mirrors vmaCreateBuffer: create the buffer, allocate dedicated
// memory for it, bind, and (with AllocationCreateMapped) map persistently.
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

// VmaCreateImage mirrors vmaCreateImage: create the image, allocate dedicated
// device-local memory for it, and bind.
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

// VmaDestroyBuffer mirrors vmaDestroyBuffer: unmap if mapped, destroy the buffer,
// free its memory.
func (a *VmaAllocator) VmaDestroyBuffer(b Buffer, al VmaAllocation) {
	if al.mapped != nil {
		UnmapMemory(a.device, al.Memory)
	}
	DestroyBuffer(a.device, b)
	FreeMemory(a.device, al.Memory)
}

// VmaDestroyImage mirrors vmaDestroyImage: destroy the image and free its memory.
func (a *VmaAllocator) VmaDestroyImage(img Image, al VmaAllocation) {
	DestroyImage(a.device, img)
	FreeMemory(a.device, al.Memory)
}

// memoryType picks a memory type index from the requirement bit mask.
// Host-writable allocations prefer a ReBAR-style DEVICE_LOCAL|HOST_VISIBLE|
// HOST_COHERENT type (fast CPU writes straight to VRAM), falling back to plain
// HOST_VISIBLE|HOST_COHERENT; device-only allocations take DEVICE_LOCAL. This
// is the heap walk VMA normally hides.
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
