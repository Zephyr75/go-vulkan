package vk

/*
#include <vulkan/vulkan.h>
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// SurfaceCapabilities is a Goified subset of VkSurfaceCapabilitiesKHR.
type SurfaceCapabilities struct {
	MinImageCount           uint32
	MaxImageCount           uint32
	CurrentExtent           Extent2D
	MinImageExtent          Extent2D
	MaxImageExtent          Extent2D
	CurrentTransform        SurfaceTransformFlags
	SupportedCompositeAlpha CompositeAlphaFlags
	SupportedUsageFlags     ImageUsageFlags
}

func GetPhysicalDeviceSurfaceCapabilitiesKHR(pd PhysicalDevice, s SurfaceKHR) (SurfaceCapabilities, error) {
	var c C.VkSurfaceCapabilitiesKHR
	err := check(C.vkGetPhysicalDeviceSurfaceCapabilitiesKHR(
		C.VkPhysicalDevice(unsafe.Pointer(pd)),
		C.VkSurfaceKHR(unsafe.Pointer(s)), &c))
	if err != nil {
		return SurfaceCapabilities{}, err
	}
	return SurfaceCapabilities{
		MinImageCount:           uint32(c.minImageCount),
		MaxImageCount:           uint32(c.maxImageCount),
		CurrentExtent:           Extent2D{uint32(c.currentExtent.width), uint32(c.currentExtent.height)},
		MinImageExtent:          Extent2D{uint32(c.minImageExtent.width), uint32(c.minImageExtent.height)},
		MaxImageExtent:          Extent2D{uint32(c.maxImageExtent.width), uint32(c.maxImageExtent.height)},
		CurrentTransform:        SurfaceTransformFlags(c.currentTransform),
		SupportedCompositeAlpha: CompositeAlphaFlags(c.supportedCompositeAlpha),
		SupportedUsageFlags:     ImageUsageFlags(c.supportedUsageFlags),
	}, nil
}

type SurfaceFormat struct {
	Format     Format
	ColorSpace ColorSpace
}

func GetPhysicalDeviceSurfaceFormatsKHR(pd PhysicalDevice, s SurfaceKHR) ([]SurfaceFormat, error) {
	dev := C.VkPhysicalDevice(unsafe.Pointer(pd))
	surf := C.VkSurfaceKHR(unsafe.Pointer(s))
	raw, err := enumerate(func(count *C.uint32_t, out *C.VkSurfaceFormatKHR) C.VkResult {
		return C.vkGetPhysicalDeviceSurfaceFormatsKHR(dev, surf, count, out)
	})
	if err != nil {
		return nil, err
	}
	res := make([]SurfaceFormat, len(raw))
	for i := range raw {
		res[i] = SurfaceFormat{Format(raw[i].format), ColorSpace(raw[i].colorSpace)}
	}
	return res, nil
}

func GetPhysicalDeviceSurfacePresentModesKHR(pd PhysicalDevice, s SurfaceKHR) ([]PresentMode, error) {
	dev := C.VkPhysicalDevice(unsafe.Pointer(pd))
	surf := C.VkSurfaceKHR(unsafe.Pointer(s))
	raw, err := enumerate(func(count *C.uint32_t, out *C.VkPresentModeKHR) C.VkResult {
		return C.vkGetPhysicalDeviceSurfacePresentModesKHR(dev, surf, count, out)
	})
	if err != nil {
		return nil, err
	}
	res := make([]PresentMode, len(raw))
	for i := range raw {
		res[i] = PresentMode(raw[i])
	}
	return res, nil
}

func GetPhysicalDeviceSurfaceSupportKHR(pd PhysicalDevice, family uint32, s SurfaceKHR) (bool, error) {
	var sup C.VkBool32
	err := check(C.vkGetPhysicalDeviceSurfaceSupportKHR(
		C.VkPhysicalDevice(unsafe.Pointer(pd)), C.uint32_t(family),
		C.VkSurfaceKHR(unsafe.Pointer(s)), &sup))
	return sup == C.VK_TRUE, err
}

// SwapchainCreateInfo is the Go-facing input to CreateSwapchainKHR.
type SwapchainCreateInfo struct {
	Surface          SurfaceKHR
	MinImageCount    uint32
	ImageFormat      Format
	ImageColorSpace  ColorSpace
	ImageExtent      Extent2D
	ImageArrayLayers uint32
	ImageUsage       ImageUsageFlags
	PreTransform     SurfaceTransformFlags
	CompositeAlpha   CompositeAlphaFlags
	PresentMode      PresentMode
	Clipped          bool
	OldSwapchain     SwapchainKHR
}

func CreateSwapchainKHR(d Device, ci SwapchainCreateInfo) (SwapchainKHR, error) {
	if ci.ImageArrayLayers == 0 {
		ci.ImageArrayLayers = 1
	}
	clipped := C.VkBool32(C.VK_FALSE)
	if ci.Clipped {
		clipped = C.VK_TRUE
	}
	info := C.VkSwapchainCreateInfoKHR{
		sType:            C.VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
		surface:          C.VkSurfaceKHR(unsafe.Pointer(ci.Surface)),
		minImageCount:    C.uint32_t(ci.MinImageCount),
		imageFormat:      C.VkFormat(ci.ImageFormat),
		imageColorSpace:  C.VkColorSpaceKHR(ci.ImageColorSpace),
		imageArrayLayers: C.uint32_t(ci.ImageArrayLayers),
		imageUsage:       C.VkImageUsageFlags(ci.ImageUsage),
		imageSharingMode: C.VK_SHARING_MODE_EXCLUSIVE,
		preTransform:     C.VkSurfaceTransformFlagBitsKHR(ci.PreTransform),
		compositeAlpha:   C.VkCompositeAlphaFlagBitsKHR(ci.CompositeAlpha),
		presentMode:      C.VkPresentModeKHR(ci.PresentMode),
		clipped:          clipped,
		oldSwapchain:     C.VkSwapchainKHR(unsafe.Pointer(ci.OldSwapchain)),
	}
	info.imageExtent = C.VkExtent2D{width: C.uint32_t(ci.ImageExtent.Width), height: C.uint32_t(ci.ImageExtent.Height)}

	var out C.VkSwapchainKHR
	if err := check(C.vkCreateSwapchainKHR(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return SwapchainKHR(unsafe.Pointer(out)), nil
}

func GetSwapchainImagesKHR(d Device, sc SwapchainKHR) ([]Image, error) {
	dev := C.VkDevice(unsafe.Pointer(d))
	swap := C.VkSwapchainKHR(unsafe.Pointer(sc))
	raw, err := enumerate(func(count *C.uint32_t, out *C.VkImage) C.VkResult {
		return C.vkGetSwapchainImagesKHR(dev, swap, count, out)
	})
	if err != nil {
		return nil, err
	}
	res := make([]Image, len(raw))
	for i := range raw {
		res[i] = Image(unsafe.Pointer(raw[i]))
	}
	return res, nil
}

func DestroySwapchainKHR(d Device, sc SwapchainKHR) {
	C.vkDestroySwapchainKHR(C.VkDevice(unsafe.Pointer(d)), C.VkSwapchainKHR(unsafe.Pointer(sc)), nil)
}

func DestroySurfaceKHR(i Instance, s SurfaceKHR) {
	C.vkDestroySurfaceKHR(C.VkInstance(unsafe.Pointer(i)), C.VkSurfaceKHR(unsafe.Pointer(s)), nil)
}

// ---- image views ---------------------------------------------------------

type ImageSubresourceRange struct {
	AspectMask     ImageAspectFlags
	BaseMipLevel   uint32
	LevelCount     uint32
	BaseArrayLayer uint32
	LayerCount     uint32
}

type ImageViewCreateInfo struct {
	Image            Image
	ViewType         ImageViewType
	Format           Format
	SubresourceRange ImageSubresourceRange
}

func CreateImageView(d Device, ci ImageViewCreateInfo) (ImageView, error) {
	info := C.VkImageViewCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		image:    C.VkImage(unsafe.Pointer(ci.Image)),
		viewType: C.VkImageViewType(ci.ViewType),
		format:   C.VkFormat(ci.Format),
	}
	info.subresourceRange = C.VkImageSubresourceRange{
		aspectMask:     C.VkImageAspectFlags(ci.SubresourceRange.AspectMask),
		baseMipLevel:   C.uint32_t(ci.SubresourceRange.BaseMipLevel),
		levelCount:     C.uint32_t(ci.SubresourceRange.LevelCount),
		baseArrayLayer: C.uint32_t(ci.SubresourceRange.BaseArrayLayer),
		layerCount:     C.uint32_t(ci.SubresourceRange.LayerCount),
	}
	var out C.VkImageView
	if err := check(C.vkCreateImageView(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return ImageView(unsafe.Pointer(out)), nil
}

func DestroyImageView(d Device, v ImageView) {
	C.vkDestroyImageView(C.VkDevice(unsafe.Pointer(d)), C.VkImageView(unsafe.Pointer(v)), nil)
}

// ---- present -------------------------------------------------------------

// AcquireNextImageKHR returns the index of the next swapchain image. On a
// resize it returns the image index alongside a non-nil error equal to
// ErrOutOfDateKHR or SuboptimalKHR — the caller branches on those rather than
// treating them as fatal.
func AcquireNextImageKHR(d Device, sc SwapchainKHR, timeout uint64, sem Semaphore, fence Fence) (uint32, error) {
	var idx C.uint32_t
	r := C.vkAcquireNextImageKHR(C.VkDevice(unsafe.Pointer(d)),
		C.VkSwapchainKHR(unsafe.Pointer(sc)), C.uint64_t(timeout),
		C.VkSemaphore(unsafe.Pointer(sem)), C.VkFence(unsafe.Pointer(fence)), &idx)
	return uint32(idx), check(r)
}

// QueuePresentKHR presents one swapchain image after waiting on wait. Same
// out-of-date/suboptimal error handling as AcquireNextImageKHR.
func QueuePresentKHR(q Queue, wait Semaphore, sc SwapchainKHR, imageIndex uint32) error {
	sem := C.VkSemaphore(unsafe.Pointer(wait))
	swap := C.VkSwapchainKHR(unsafe.Pointer(sc))
	idx := C.uint32_t(imageIndex)
	// info embeds pointers to the Go locals above; pin them across the call.
	var pin runtime.Pinner
	defer pin.Unpin()
	pin.Pin(&sem)
	pin.Pin(&swap)
	pin.Pin(&idx)
	info := C.VkPresentInfoKHR{
		sType:              C.VK_STRUCTURE_TYPE_PRESENT_INFO_KHR,
		waitSemaphoreCount: 1,
		pWaitSemaphores:    &sem,
		swapchainCount:     1,
		pSwapchains:        &swap,
		pImageIndices:      &idx,
	}
	return check(C.vkQueuePresentKHR(C.VkQueue(unsafe.Pointer(q)), &info))
}
