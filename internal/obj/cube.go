package obj

// Cube builds a unit cube with per-face normals and UVs. It stands in for the
// reference's tinyobjloader load of suzanne.obj when no mesh asset is present
// on disk, so the demo runs with zero external files.
func Cube() *Mesh {
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
	m := &Mesh{}
	uv := [4][2]float32{{0, 1}, {1, 1}, {1, 0}, {0, 0}}
	for _, f := range faces {
		base := uint32(len(m.Vertices))
		for k, p := range [4][3]float32{f.a, f.b, f.c, f.d} {
			m.Vertices = append(m.Vertices, Vertex{Pos: p, Normal: f.normal, UV: uv[k]})
		}
		// Two triangles per quad face.
		m.Indices = append(m.Indices, base, base+1, base+2, base, base+2, base+3)
	}
	return m
}
