package vk

/*
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

type BufferCreateInfo struct {
	Size  uint64
	Usage BufferUsageFlags
}

func CreateBuffer(d Device, ci BufferCreateInfo) (Buffer, error) {
	info := C.VkBufferCreateInfo{
		sType:       C.VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		size:        C.VkDeviceSize(ci.Size),
		usage:       C.VkBufferUsageFlags(ci.Usage),
		sharingMode: C.VK_SHARING_MODE_EXCLUSIVE,
	}
	var out C.VkBuffer
	if err := check(C.vkCreateBuffer(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return Buffer(unsafe.Pointer(out)), nil
}

func DestroyBuffer(d Device, b Buffer) {
	C.vkDestroyBuffer(C.VkDevice(unsafe.Pointer(d)), C.VkBuffer(unsafe.Pointer(b)), nil)
}

// ImageCreateInfo is the Go-facing input to CreateImage. Tiling is OPTIMAL,
// samples 1, initial layout UNDEFINED, exclusive sharing.
type ImageCreateInfo struct {
	ImageType   ImageType
	Format      Format
	Extent      Extent3D
	MipLevels   uint32
	ArrayLayers uint32
	Usage       ImageUsageFlags
}

func CreateImage(d Device, ci ImageCreateInfo) (Image, error) {
	if ci.MipLevels == 0 {
		ci.MipLevels = 1
	}
	if ci.ArrayLayers == 0 {
		ci.ArrayLayers = 1
	}
	info := C.VkImageCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		imageType:     C.VkImageType(ci.ImageType),
		format:        C.VkFormat(ci.Format),
		mipLevels:     C.uint32_t(ci.MipLevels),
		arrayLayers:   C.uint32_t(ci.ArrayLayers),
		samples:       C.VK_SAMPLE_COUNT_1_BIT,
		tiling:        C.VK_IMAGE_TILING_OPTIMAL,
		usage:         C.VkImageUsageFlags(ci.Usage),
		sharingMode:   C.VK_SHARING_MODE_EXCLUSIVE,
		initialLayout: C.VK_IMAGE_LAYOUT_UNDEFINED,
	}
	info.extent = C.VkExtent3D{
		width:  C.uint32_t(ci.Extent.Width),
		height: C.uint32_t(ci.Extent.Height),
		depth:  C.uint32_t(ci.Extent.Depth),
	}
	var out C.VkImage
	if err := check(C.vkCreateImage(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return Image(unsafe.Pointer(out)), nil
}

func DestroyImage(d Device, img Image) {
	C.vkDestroyImage(C.VkDevice(unsafe.Pointer(d)), C.VkImage(unsafe.Pointer(img)), nil)
}

// SamplerCreateInfo is the Go-facing input to CreateSampler.
type SamplerCreateInfo struct {
	MagFilter        Filter
	MinFilter        Filter
	MipmapMode       SamplerMipmapMode
	AddressModeU     SamplerAddressMode
	AddressModeV     SamplerAddressMode
	AddressModeW     SamplerAddressMode
	AnisotropyEnable bool
	MaxAnisotropy    float32
	MaxLod           float32
}

func CreateSampler(d Device, ci SamplerCreateInfo) (Sampler, error) {
	aniso := C.VkBool32(C.VK_FALSE)
	if ci.AnisotropyEnable {
		aniso = C.VK_TRUE
	}
	info := C.VkSamplerCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
		magFilter:        C.VkFilter(ci.MagFilter),
		minFilter:        C.VkFilter(ci.MinFilter),
		mipmapMode:       C.VkSamplerMipmapMode(ci.MipmapMode),
		addressModeU:     C.VkSamplerAddressMode(ci.AddressModeU),
		addressModeV:     C.VkSamplerAddressMode(ci.AddressModeV),
		addressModeW:     C.VkSamplerAddressMode(ci.AddressModeW),
		anisotropyEnable: aniso,
		maxAnisotropy:    C.float(ci.MaxAnisotropy),
		minLod:           0,
		maxLod:           C.float(ci.MaxLod),
		borderColor:      C.VK_BORDER_COLOR_FLOAT_OPAQUE_BLACK,
	}
	var out C.VkSampler
	if err := check(C.vkCreateSampler(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return Sampler(unsafe.Pointer(out)), nil
}

func DestroySampler(d Device, s Sampler) {
	C.vkDestroySampler(C.VkDevice(unsafe.Pointer(d)), C.VkSampler(unsafe.Pointer(s)), nil)
}
