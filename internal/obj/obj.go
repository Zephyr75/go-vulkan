// Package obj is a minimal Wavefront OBJ parser: positions, normals, UVs and
// triangulated faces, deduplicated into an interleaved vertex + index buffer.
package obj

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Vertex is the interleaved layout consumed by the demo's graphics pipeline:
// position (3), normal (3), uv (2). 32 bytes.
type Vertex struct {
	Pos    [3]float32
	Normal [3]float32
	UV     [2]float32
}

// Mesh is the parsed result.
type Mesh struct {
	Vertices []Vertex
	Indices  []uint32
}

// Load parses an OBJ file. UVs get the tutorial's v-flip (v -> 1-v) for
// Vulkan's top-left texture origin.
func Load(path string) (*Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var positions [][3]float32
	var normals [][3]float32
	var uvs [][2]float32

	mesh := &Mesh{}
	cache := map[[3]int]uint32{} // (v,vt,vn) -> vertex index

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "v":
			p, err := parse3(fields[1:])
			if err != nil {
				return nil, err
			}
			positions = append(positions, p)
		case "vn":
			n, err := parse3(fields[1:])
			if err != nil {
				return nil, err
			}
			normals = append(normals, n)
		case "vt":
			if len(fields) < 3 {
				return nil, fmt.Errorf("obj: bad vt %q", sc.Text())
			}
			u, err := strconv.ParseFloat(fields[1], 32)
			if err != nil {
				return nil, err
			}
			v, err := strconv.ParseFloat(fields[2], 32)
			if err != nil {
				return nil, err
			}
			uvs = append(uvs, [2]float32{float32(u), 1 - float32(v)})
		case "f":
			// Triangulate a polygon fan.
			idx := make([]uint32, 0, len(fields)-1)
			for _, tok := range fields[1:] {
				vi, err := resolve(tok, cache, mesh, positions, normals, uvs)
				if err != nil {
					return nil, err
				}
				idx = append(idx, vi)
			}
			for i := 1; i+1 < len(idx); i++ {
				mesh.Indices = append(mesh.Indices, idx[0], idx[i], idx[i+1])
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return mesh, nil
}

func resolve(tok string, cache map[[3]int]uint32, mesh *Mesh, pos [][3]float32, norm [][3]float32, uv [][2]float32) (uint32, error) {
	parts := strings.Split(tok, "/")
	key := [3]int{-1, -1, -1}
	for i := 0; i < len(parts) && i < 3; i++ {
		if parts[i] == "" {
			continue
		}
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return 0, err
		}
		key[i] = n
	}
	if vi, ok := cache[key]; ok {
		return vi, nil
	}
	var vert Vertex
	if key[0] > 0 && key[0] <= len(pos) {
		vert.Pos = pos[key[0]-1]
	}
	if key[1] > 0 && key[1] <= len(uv) {
		vert.UV = uv[key[1]-1]
	}
	if key[2] > 0 && key[2] <= len(norm) {
		vert.Normal = norm[key[2]-1]
	}
	vi := uint32(len(mesh.Vertices))
	mesh.Vertices = append(mesh.Vertices, vert)
	cache[key] = vi
	return vi, nil
}

func parse3(f []string) ([3]float32, error) {
	if len(f) < 3 {
		return [3]float32{}, fmt.Errorf("obj: need 3 floats, got %v", f)
	}
	var out [3]float32
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(f[i], 32)
		if err != nil {
			return out, err
		}
		out[i] = float32(v)
	}
	return out, nil
}
