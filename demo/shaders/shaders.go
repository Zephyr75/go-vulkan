// Package shaders embeds the offline-compiled SPIR-V used by the demo. Rebuild
// the .spv with `go generate ./shaders` after editing the GLSL sources.
package shaders

import _ "embed"

//go:generate glslc --target-env=vulkan1.3 shader.vert -o vert.spv
//go:generate glslc --target-env=vulkan1.3 shader.frag -o frag.spv

//go:embed vert.spv
var Vert []byte

//go:embed frag.spv
var Frag []byte
