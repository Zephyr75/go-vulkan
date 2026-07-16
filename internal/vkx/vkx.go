// Package vkx holds the small Go-side helpers the howto demo needs but that are
// not part of the vk bindings themselves: the VMA-substitute allocators (raw
// allocate + bind + optional map, bundled into one call the way vmaCreateBuffer
// / vmaCreateImage bundle theirs), the GLFW->vk surface bridge, and a procedural
// cube used when no mesh asset is on disk.
//
// It lives in its own package so cmd/howto/main.go can stay a close, flat mirror
// of the single-file C++ reference: main.go reads top-to-bottom like the original
// and defers the Go/ecosystem glue to the calls below. Every helper aborts the
// program on failure, matching the reference's chk()/exit() convention, so the
// call sites stay one-liners.
package vkx

import (
	"log"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"

	"govk/internal/obj"
	"govk/vk"
)

// must aborts on error, mirroring the reference's chk().
func must(err error) {
	if err != nil {
		log.Fatalf("vkx: %v", err)
	}
}

// AllocBuffer is a buffer plus its backing memory and (for host-visible buffers)
// the persistently mapped pointer. It stands in for VMA's VmaAllocation +
// VmaAllocationInfo bundle.
type AllocBuffer struct {
	Buffer vk.Buffer
	Memory vk.DeviceMemory
	Mapped unsafe.Pointer // non-nil only when created host-visible
}

// CreateBuffer is the raw-Vulkan equivalent of vmaCreateBuffer: create the
// buffer, query its requirements, allocate memory, bind, and (when host-visible)
// map it persistently. hostVisible selects a CPU-writable memory type; bda
// requests a device-address-capable allocation for buffer_device_address.
func CreateBuffer(d vk.Device, mp vk.PhysicalDeviceMemoryProperties, ci vk.BufferCreateInfo, hostVisible, bda bool) AllocBuffer {
	buf, err := vk.CreateBuffer(d, ci)
	must(err)
	req := vk.GetBufferMemoryRequirements(d, buf)
	mem, err := vk.AllocateMemory(d, vk.MemoryAllocateInfo{
		AllocationSize:  req.Size,
		MemoryTypeIndex: memoryType(mp, req.MemoryTypeBits, hostVisible),
		DeviceAddress:   bda,
	})
	must(err)
	must(vk.BindBufferMemory(d, buf, mem, 0))
	var ptr unsafe.Pointer
	if hostVisible {
		ptr, err = vk.MapMemory(d, mem, 0, vk.WholeSize)
		must(err)
	}
	return AllocBuffer{Buffer: buf, Memory: mem, Mapped: ptr}
}

// Destroy unmaps (if mapped), destroys the buffer, and frees its memory.
func (b AllocBuffer) Destroy(d vk.Device) {
	if b.Mapped != nil {
		vk.UnmapMemory(d, b.Memory)
	}
	vk.DestroyBuffer(d, b.Buffer)
	vk.FreeMemory(d, b.Memory)
}

// CreateImage is the raw-Vulkan equivalent of vmaCreateImage: create the image,
// allocate device-local memory for it, and bind. The caller creates the image
// view separately (the reference does the same: vmaCreateImage then
// vkCreateImageView), because view aspect/format vary per use.
func CreateImage(d vk.Device, mp vk.PhysicalDeviceMemoryProperties, ci vk.ImageCreateInfo) (vk.Image, vk.DeviceMemory) {
	img, err := vk.CreateImage(d, ci)
	must(err)
	req := vk.GetImageMemoryRequirements(d, img)
	mem, err := vk.AllocateMemory(d, vk.MemoryAllocateInfo{
		AllocationSize:  req.Size,
		MemoryTypeIndex: memoryType(mp, req.MemoryTypeBits, false),
	})
	must(err)
	must(vk.BindImageMemory(d, img, mem, 0))
	return img, mem
}

// memoryType picks a suitable memory type index from the requirement bit mask.
// Host-visible allocations prefer a ReBAR-style DEVICE_LOCAL|HOST_VISIBLE|
// HOST_COHERENT type (fast CPU writes straight to VRAM), falling back to plain
// HOST_VISIBLE|HOST_COHERENT. Non-host-visible allocations take DEVICE_LOCAL.
// This is the heap walk VMA normally hides.
func memoryType(mp vk.PhysicalDeviceMemoryProperties, bits uint32, hostVisible bool) uint32 {
	if hostVisible {
		if idx, err := vk.FindMemoryType(mp, bits,
			vk.MemoryPropertyDeviceLocal|vk.MemoryPropertyHostVisible|vk.MemoryPropertyHostCoherent); err == nil {
			return idx
		}
		idx, err := vk.FindMemoryType(mp, bits, vk.MemoryPropertyHostVisible|vk.MemoryPropertyHostCoherent)
		must(err)
		return idx
	}
	idx, err := vk.FindMemoryType(mp, bits, vk.MemoryPropertyDeviceLocal)
	must(err)
	return idx
}

// CreateSurface bridges GLFW's window-surface creation to a vk.SurfaceKHR. GLFW
// wants the instance as a pointer-kind value and returns a pointer to the created
// surface handle, so both ends round-trip through unsafe.Pointer. This replaces
// the reference's SDL_Vulkan_CreateSurface.
func CreateSurface(instance vk.Instance, w *glfw.Window) vk.SurfaceKHR {
	instPtr := (*byte)(unsafe.Pointer(instance))
	surfRaw, err := w.CreateWindowSurface(instPtr, nil)
	must(err)
	return vk.SurfaceKHR(*(*uintptr)(unsafe.Pointer(surfRaw)))
}

// CubeMesh builds a unit cube with per-face normals and UVs. It stands in for
// the reference's tinyobjloader load of suzanne.obj when no mesh asset is present
// on disk, so the demo runs with zero external files.
func CubeMesh() *obj.Mesh {
	type face struct {
		normal     [3]float32
		a, b, c, d [3]float32
	}
	// Six faces wound counter-clockwise when viewed from outside.
	faces := []face{
		{[3]float32{0, 0, 1}, [3]float32{-1, -1, 1}, [3]float32{1, -1, 1}, [3]float32{1, 1, 1}, [3]float32{-1, 1, 1}},
		{[3]float32{0, 0, -1}, [3]float32{1, -1, -1}, [3]float32{-1, -1, -1}, [3]float32{-1, 1, -1}, [3]float32{1, 1, -1}},
		{[3]float32{1, 0, 0}, [3]float32{1, -1, 1}, [3]float32{1, -1, -1}, [3]float32{1, 1, -1}, [3]float32{1, 1, 1}},
		{[3]float32{-1, 0, 0}, [3]float32{-1, -1, -1}, [3]float32{-1, -1, 1}, [3]float32{-1, 1, 1}, [3]float32{-1, 1, -1}},
		{[3]float32{0, 1, 0}, [3]float32{-1, 1, 1}, [3]float32{1, 1, 1}, [3]float32{1, 1, -1}, [3]float32{-1, 1, -1}},
		{[3]float32{0, -1, 0}, [3]float32{-1, -1, -1}, [3]float32{1, -1, -1}, [3]float32{1, -1, 1}, [3]float32{-1, -1, 1}},
	}
	m := &obj.Mesh{}
	uv := [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}
	for _, f := range faces {
		base := uint32(len(m.Vertices))
		for k, p := range [4][3]float32{f.a, f.b, f.c, f.d} {
			m.Vertices = append(m.Vertices, obj.Vertex{Pos: p, Normal: f.normal, UV: uv[k]})
		}
		// Two triangles per quad face.
		m.Indices = append(m.Indices, base, base+1, base+2, base, base+2, base+3)
	}
	return m
}
