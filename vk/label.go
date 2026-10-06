package vk

/*
#include <stdlib.h>
#include <vulkan/vulkan.h>

// VK_EXT_debug_utils is an extension, so the loader does not export its entry
// points: they have to be fetched with vkGetInstanceProcAddr once the instance
// exists. Both calls below are no-ops until ovd_load_labels has run
static PFN_vkCmdBeginDebugUtilsLabelEXT ovd_beginLabel = NULL;
static PFN_vkCmdEndDebugUtilsLabelEXT   ovd_endLabel   = NULL;

static void ovd_load_labels(VkInstance inst) {
	ovd_beginLabel = (PFN_vkCmdBeginDebugUtilsLabelEXT)vkGetInstanceProcAddr(inst, "vkCmdBeginDebugUtilsLabelEXT");
	ovd_endLabel   = (PFN_vkCmdEndDebugUtilsLabelEXT)vkGetInstanceProcAddr(inst, "vkCmdEndDebugUtilsLabelEXT");
}

static void ovd_begin_label(VkCommandBuffer cb, const char *name) {
	if (!ovd_beginLabel) return;
	VkDebugUtilsLabelEXT l;
	l.sType = VK_STRUCTURE_TYPE_DEBUG_UTILS_LABEL_EXT;
	l.pNext = NULL;
	l.pLabelName = name;
	l.color[0] = 0.0f; l.color[1] = 0.0f; l.color[2] = 0.0f; l.color[3] = 1.0f;
	ovd_beginLabel(cb, &l);
}

static void ovd_end_label(VkCommandBuffer cb) {
	if (!ovd_endLabel) return;
	ovd_endLabel(cb);
}
*/
import "C"

import "unsafe"

// The instance extension the two entry points below come from
const ExtDebugUtils = "VK_EXT_debug_utils"

// Fetches the label entry points, after instance creation
//
// Silently leaves them unloaded when the extension was not enabled, so both
// calls below become no-ops rather than a crash
func LoadDebugUtils(inst Instance) {
	C.ovd_load_labels(C.VkInstance(unsafe.Pointer(inst)))
}

// Opens a labelled region, which is what groups a RenderDoc capture by pass
func CmdBeginDebugLabel(cb CommandBuffer, name string) {
	c := C.CString(name)
	defer C.free(unsafe.Pointer(c))
	C.ovd_begin_label(C.VkCommandBuffer(unsafe.Pointer(cb)), c)
}

// Closes the region CmdBeginDebugLabel opened
func CmdEndDebugLabel(cb CommandBuffer) {
	C.ovd_end_label(C.VkCommandBuffer(unsafe.Pointer(cb)))
}
