package vk

import "testing"

// TestInstanceDevice is a headless smoke test for Milestone 1: create an
// instance with validation, pick the first GPU, create a 1.3 device with the
// tutorial's features, print the name. Run: go test ./vk/ -run Instance -v
func TestInstanceDevice(t *testing.T) {
	inst, err := CreateInstance(InstanceCreateInfo{
		AppName:    "smoke",
		APIVersion: ApiVersion13,
		Layers:     []string{"VK_LAYER_KHRONOS_validation"},
	})
	if err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}
	defer DestroyInstance(inst)

	gpus, err := EnumeratePhysicalDevices(inst)
	if err != nil {
		t.Fatalf("EnumeratePhysicalDevices: %v", err)
	}
	if len(gpus) == 0 {
		t.Fatal("no GPUs")
	}
	pd := gpus[0]
	props := GetPhysicalDeviceProperties2(pd)
	t.Logf("GPU: %s (type %d, api %d.%d.%d)", props.DeviceName, props.DeviceType,
		props.APIVersion>>22, (props.APIVersion>>12)&0x3ff, props.APIVersion&0xfff)

	fams := GetPhysicalDeviceQueueFamilyProperties(pd)
	gfx := -1
	for i, f := range fams {
		if f.QueueFlags&QueueGraphics != 0 {
			gfx = i
			break
		}
	}
	if gfx < 0 {
		t.Fatal("no graphics queue family")
	}
	t.Logf("graphics queue family: %d", gfx)

	mem := GetPhysicalDeviceMemoryProperties2(pd)
	t.Logf("memory: %d types, %d heaps", len(mem.MemoryTypes), len(mem.MemoryHeaps))

	dev, err := CreateDevice(pd, DeviceCreateInfo{
		QueueCreateInfos: []DeviceQueueCreateInfo{{QueueFamilyIndex: uint32(gfx), Priorities: []float32{1.0}}},
		Features: Features{
			SamplerAnisotropy:                         true,
			DynamicRendering:                          true,
			Synchronization2:                          true,
			BufferDeviceAddress:                       true,
			DescriptorIndexing:                        true,
			RuntimeDescriptorArray:                    true,
			DescriptorBindingPartiallyBound:           true,
			DescriptorBindingVariableDescriptorCount:  true,
			ShaderSampledImageArrayNonUniformIndexing: true,
		},
	})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	defer DestroyDevice(dev)

	q := GetDeviceQueue(dev, uint32(gfx), 0)
	if q == 0 {
		t.Fatal("nil queue")
	}
	t.Log("device + queue created OK")
}
