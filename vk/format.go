package vk

/*
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

// FormatProperties is a Goified VkFormatProperties: the feature bits a format
// supports for linear/optimal-tiled images and for buffers. Used to pick a
// supported depth/stencil format at runtime.
type FormatProperties struct {
	LinearTilingFeatures  FormatFeatureFlags
	OptimalTilingFeatures FormatFeatureFlags
	BufferFeatures        FormatFeatureFlags
}

// GetPhysicalDeviceFormatProperties2 queries the feature support for a format.
func GetPhysicalDeviceFormatProperties2(pd PhysicalDevice, f Format) FormatProperties {
	var fp C.VkFormatProperties2
	fp.sType = C.VK_STRUCTURE_TYPE_FORMAT_PROPERTIES_2
	C.vkGetPhysicalDeviceFormatProperties2(C.VkPhysicalDevice(unsafe.Pointer(pd)), C.VkFormat(f), &fp)
	p := fp.formatProperties
	return FormatProperties{
		LinearTilingFeatures:  FormatFeatureFlags(p.linearTilingFeatures),
		OptimalTilingFeatures: FormatFeatureFlags(p.optimalTilingFeatures),
		BufferFeatures:        FormatFeatureFlags(p.bufferFeatures),
	}
}
