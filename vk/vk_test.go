package vk

import (
	"testing"
	"unsafe"
)

// Headless tests for the binding: no window, no surface, no swapchain. They
// exercise instance/device creation, the raw memory path, and the VMA
// substitute against the first GPU. Skipped when no Vulkan ICD or device is
// available so `go test ./...` stays green on machines without a GPU.

// Creates an instance + device on the first GPU, registers their teardown, and skips the test when Vulkan is unavailable
func newTestDevice(t *testing.T) (PhysicalDevice, Device) {
	t.Helper()
	inst, err := CreateInstance(InstanceCreateInfo{AppName: "vk-test", APIVersion: ApiVersion13})
	if err != nil {
		t.Skipf("no Vulkan available: %v", err)
	}
	t.Cleanup(func() { DestroyInstance(inst) })

	gpus, err := EnumeratePhysicalDevices(inst)
	if err != nil || len(gpus) == 0 {
		t.Skipf("no Vulkan device: %v", err)
	}
	pd := gpus[0]

	gfx := -1
	for i, f := range GetPhysicalDeviceQueueFamilyProperties(pd) {
		if f.QueueFlags&QueueGraphics != 0 {
			gfx = i
			break
		}
	}
	if gfx < 0 {
		t.Fatal("no graphics queue family")
	}

	dev, err := CreateDevice(pd, DeviceCreateInfo{
		QueueCreateInfos: []DeviceQueueCreateInfo{{QueueFamilyIndex: uint32(gfx), Priorities: []float32{1}}},
		Features: Features{
			SamplerAnisotropy:   true,
			DynamicRendering:    true,
			Synchronization2:    true,
			BufferDeviceAddress: true,
		},
	})
	if err != nil {
		t.Skipf("CreateDevice: %v", err)
	}
	t.Cleanup(func() { DestroyDevice(dev) })
	return pd, dev
}

func TestInstanceDevice(t *testing.T) {
	pd, dev := newTestDevice(t)
	props := GetPhysicalDeviceProperties2(pd)
	if props.DeviceName == "" {
		t.Error("empty device name")
	}
	t.Logf("GPU: %s (api %d.%d.%d)", props.DeviceName,
		props.APIVersion>>22, (props.APIVersion>>12)&0x3ff, props.APIVersion&0xfff)

	mem := GetPhysicalDeviceMemoryProperties2(pd)
	if len(mem.MemoryTypes) == 0 || len(mem.MemoryHeaps) == 0 {
		t.Errorf("empty memory properties: %d types, %d heaps", len(mem.MemoryTypes), len(mem.MemoryHeaps))
	}

	if q := GetDeviceQueue(dev, 0, 0); q == 0 {
		t.Error("nil queue")
	}
}

func TestDepthFormatProbe(t *testing.T) {
	pd, _ := newTestDevice(t)
	found := false
	for _, f := range []Format{FormatD32SfloatS8Uint, FormatD24UnormS8Uint} {
		fp := GetPhysicalDeviceFormatProperties2(pd, f)
		if fp.OptimalTilingFeatures&FormatFeatureDepthStencilAttachment != 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("no supported depth/stencil format among the tutorial's candidates")
	}
}

func TestFindMemoryType(t *testing.T) {
	pd, _ := newTestDevice(t)
	mp := GetPhysicalDeviceMemoryProperties2(pd)

	if _, err := FindMemoryType(mp, ^uint32(0), MemoryPropertyDeviceLocal); err != nil {
		t.Errorf("no device-local memory type: %v", err)
	}
	if _, err := FindMemoryType(mp, 0, MemoryPropertyDeviceLocal); err == nil {
		t.Error("empty type-bits mask should not match any memory type")
	}
}

func TestRawBufferLifecycle(t *testing.T) {
	pd, dev := newTestDevice(t)
	mp := GetPhysicalDeviceMemoryProperties2(pd)

	buf, err := CreateBuffer(dev, BufferCreateInfo{Size: 256, Usage: BufferUsageTransferSrc})
	if err != nil {
		t.Fatalf("CreateBuffer: %v", err)
	}
	defer DestroyBuffer(dev, buf)

	req := GetBufferMemoryRequirements(dev, buf)
	if req.Size < 256 {
		t.Errorf("requirements size %d < requested 256", req.Size)
	}
	idx, err := FindMemoryType(mp, req.MemoryTypeBits, MemoryPropertyHostVisible|MemoryPropertyHostCoherent)
	if err != nil {
		t.Fatalf("FindMemoryType: %v", err)
	}
	mem, err := AllocateMemory(dev, MemoryAllocateInfo{AllocationSize: req.Size, MemoryTypeIndex: idx})
	if err != nil {
		t.Fatalf("AllocateMemory: %v", err)
	}
	defer FreeMemory(dev, mem)
	if err := BindBufferMemory(dev, buf, mem, 0); err != nil {
		t.Fatalf("BindBufferMemory: %v", err)
	}

	ptr, err := MapMemory(dev, mem, 0, WholeSize)
	if err != nil {
		t.Fatalf("MapMemory: %v", err)
	}
	defer UnmapMemory(dev, mem)

	src := []uint32{1, 2, 3, 4}
	MemCopy(ptr, src)
	got := unsafe.Slice((*uint32)(ptr), len(src))
	for i, v := range src {
		if got[i] != v {
			t.Errorf("mapped[%d] = %d, want %d", i, got[i], v)
		}
	}
}

func TestVmaBuffer(t *testing.T) {
	pd, dev := newTestDevice(t)
	allocator := VmaCreateAllocator(VmaAllocatorCreateInfo{
		Flags:          VmaAllocatorCreateBufferDeviceAddressBit,
		PhysicalDevice: pd,
		Device:         dev,
	})
	defer VmaDestroyAllocator(allocator)

	buf, alloc, info, err := allocator.VmaCreateBuffer(
		BufferCreateInfo{Size: 512, Usage: BufferUsageVertexBuffer},
		VmaAllocationCreateInfo{
			Flags: VmaAllocationCreateHostAccessSequentialWrite | VmaAllocationCreateMapped,
			Usage: VmaMemoryUsageAuto,
		})
	if err != nil {
		t.Fatalf("VmaCreateBuffer: %v", err)
	}
	defer allocator.VmaDestroyBuffer(buf, alloc)

	if info.MappedData == nil {
		t.Fatal("VmaAllocationCreateMapped set but MappedData is nil")
	}
	if info.Size < 512 {
		t.Errorf("allocation size %d < requested 512", info.Size)
	}
	data := []float32{1.5, 2.5, 3.5}
	MemCopy(info.MappedData, data)
	got := unsafe.Slice((*float32)(info.MappedData), len(data))
	for i, v := range data {
		if got[i] != v {
			t.Errorf("mapped[%d] = %v, want %v", i, got[i], v)
		}
	}
}

func TestVmaBufferDeviceAddress(t *testing.T) {
	pd, dev := newTestDevice(t)
	allocator := VmaCreateAllocator(VmaAllocatorCreateInfo{
		Flags:          VmaAllocatorCreateBufferDeviceAddressBit,
		PhysicalDevice: pd,
		Device:         dev,
	})
	defer VmaDestroyAllocator(allocator)

	buf, alloc, _, err := allocator.VmaCreateBuffer(
		BufferCreateInfo{Size: 64, Usage: BufferUsageShaderDeviceAddress},
		VmaAllocationCreateInfo{
			Flags: VmaAllocationCreateHostAccessSequentialWrite | VmaAllocationCreateMapped,
			Usage: VmaMemoryUsageAuto,
		})
	if err != nil {
		t.Fatalf("VmaCreateBuffer: %v", err)
	}
	defer allocator.VmaDestroyBuffer(buf, alloc)

	if addr := GetBufferDeviceAddress(dev, buf); addr == 0 {
		t.Error("GetBufferDeviceAddress returned 0")
	}
}

func TestVmaImage(t *testing.T) {
	pd, dev := newTestDevice(t)
	allocator := VmaCreateAllocator(VmaAllocatorCreateInfo{PhysicalDevice: pd, Device: dev})
	defer VmaDestroyAllocator(allocator)

	img, alloc, err := allocator.VmaCreateImage(ImageCreateInfo{
		ImageType: ImageType2D,
		Format:    FormatR8G8B8A8Unorm,
		Extent:    Extent3D{Width: 64, Height: 64, Depth: 1},
		Usage:     ImageUsageTransferDst | ImageUsageSampled,
	}, VmaAllocationCreateInfo{Usage: VmaMemoryUsageAuto})
	if err != nil {
		t.Fatalf("VmaCreateImage: %v", err)
	}
	defer allocator.VmaDestroyImage(img, alloc)

	view, err := CreateImageView(dev, ImageViewCreateInfo{
		Image: img, ViewType: ImageViewType2D, Format: FormatR8G8B8A8Unorm,
		SubresourceRange: ImageSubresourceRange{AspectMask: ImageAspectColor, LevelCount: 1, LayerCount: 1},
	})
	if err != nil {
		t.Fatalf("CreateImageView: %v", err)
	}
	DestroyImageView(dev, view)
}

func TestClearValues(t *testing.T) {
	cv := ClearColor(0.25, 0.5, 0.75, 1)
	f := (*[4]float32)(unsafe.Pointer(&cv[0]))
	if f[0] != 0.25 || f[1] != 0.5 || f[2] != 0.75 || f[3] != 1 {
		t.Errorf("ClearColor layout wrong: %v", *f)
	}

	ds := ClearDepthStencil(1, 42)
	if d := *(*float32)(unsafe.Pointer(&ds[0])); d != 1 {
		t.Errorf("depth = %v, want 1", d)
	}
	if s := *(*uint32)(unsafe.Pointer(&ds[4])); s != 42 {
		t.Errorf("stencil = %d, want 42", s)
	}
}
