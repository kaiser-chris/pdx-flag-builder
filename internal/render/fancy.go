package render

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/assets"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/mesh"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/pdx"
	"github.com/kaiser-chris/pdx-flag-builder-go/internal/texture"
)

// flagSize is how large the coat of arms is drawn before it is put on the
// cloth. The cloth covers a fraction of the picture it is drawn in, so a flag
// as wide as the whole picture already has more in it than the cloth can show,
// and a smaller cloth needs less again.
func flagSize(cloth Cloth) (width, height int32) {
	if cloth.RenderWidth > FlagWidth/2 {
		return FlagWidth, FlagHeight
	}

	return FlagWidth / 2, FlagHeight / 2
}

// The camera the game looks at the flag with, from the widget's own
// definition: nearly head on, far away and through a narrow lens, which is
// what keeps the cloth from looking like a perspective drawing.
//
// Two things differ from the game's numbers. The camera and the cloth are
// mirrored across the x axis, because the game composes its scene the way
// Direct3D does, with x growing to the right of the screen, and raylib the
// way OpenGL does, with x growing to the left of a camera looking this way;
// mirroring both puts the cloth on the screen the way round the game has it.
// And the camera looks at the middle of the cloth rather than at the point
// the game looks at, which is off to one side to leave room for the pole and
// the decoration the game draws beside it; here the cloth is on its own and
// belongs in the middle of its picture.
var (
	fancyCamera = rl.Camera3D{
		Position:   rl.Vector3{Z: -41},
		Target:     rl.Vector3{},
		Up:         rl.Vector3{Y: 1},
		Fovy:       10,
		Projection: rl.CameraPerspective,
	}

	fancyTransform = rl.MatrixScale(-1, 1, 1)
)

// shaderLocations is how many places raylib keeps in a shader's table of
// where its own values live.
const shaderLocations = 32

// The files of the game the cloth is drawn from: the mesh the flag hangs on,
// and the three maps its material is drawn with.
const (
	ClothMesh       = "ui_flag_01.mesh"
	ClothDiffuse    = "ui_flag_01_diffuse.dds"
	ClothNormal     = "ui_flag_01_normal.dds"
	ClothProperties = "ui_flag_01_properties.dds"
)

// Fancy draws a coat of arms as the cloth banner Victoria 3 waves in its
// interface: the game's own mesh, its three material maps, and its wave, with
// the coat of arms drawn into the cloth.
//
// The cloth belongs to the game rather than to any coat of arms, and is read
// from the configured folders the first time it is wanted. A set of folders
// without it draws nothing, and Ready says so.
type Fancy struct {
	cloth   Cloth
	painter *Painter

	// resolve turns the name of one of the game's files into a path on disk.
	resolve func(name string) (string, bool)

	// maps are the cloth's own material maps. The cloth holds them itself
	// rather than sharing the cache the coats of arms are drawn from, because
	// a material keeps nothing but the numbers the graphics card knows its
	// textures by: a cache that dropped one would leave the cloth drawing
	// whatever picture was given that number next.
	maps []rl.Texture2D

	// ready is set once the cloth and its maps have been read, and problem
	// holds why a cloth that was found could not be used.
	ready   bool
	problem error

	target rl.RenderTexture2D

	// flag is the coat of arms on its own, drawn without the checkerboard and
	// the border the editor's preview has, since it goes onto the cloth.
	flag rl.RenderTexture2D

	shader rl.Shader

	// The mesh is allocated on its own rather than kept in this struct.
	// Handing C a pointer into a larger structure has it check everything
	// that structure holds for pointers the garbage collector may move, and
	// the arrays of a mesh are the only ones that are pinned.
	model    *rl.Mesh
	material *rl.Material

	// The geometry stays referenced for as long as the mesh lives: raylib
	// keeps the arrays it was handed rather than copying them.
	geometry *mesh.Mesh
	pinned   runtime.Pinner

	time int32
}

// NewFancy prepares everything needed to draw the cloth. The cloth itself is
// read from the configured folders the first time it is drawn, since they may
// not have been read yet, or may not hold it at all.
//
// It requires an active OpenGL context, so it must be called after the window
// exists.
func NewFancy(cloth Cloth, painter *Painter, resolve func(name string) (string, bool)) (*Fancy, error) {
	vertex, err := assets.Read(assets.ShaderFlagVertex)
	if err != nil {
		return nil, err
	}

	fragment, err := assets.Read(assets.ShaderFlagFragment)
	if err != nil {
		return nil, err
	}

	shader := rl.LoadShaderFromMemory(string(vertex), string(fragment))
	if shader.ID == 0 {
		return nil, fmt.Errorf("the cloth shader did not compile")
	}

	flagWidth, flagHeight := flagSize(cloth)

	fancy := &Fancy{
		cloth:   cloth,
		painter: painter,
		resolve: resolve,
		target:  rl.LoadRenderTexture(cloth.RenderWidth, cloth.RenderHeight),
		flag:    rl.LoadRenderTexture(flagWidth, flagHeight),
		shader:  shader,
		time:    rl.GetShaderLocation(shader, "time"),
	}

	rl.SetTextureFilter(fancy.target.Texture, rl.FilterBilinear)
	rl.SetTextureFilter(fancy.flag.Texture, rl.FilterBilinear)

	material := rl.LoadMaterialDefault()
	material.Shader = shader
	fancy.material = &material

	// The coat of arms is a map of the material, because that is what raylib
	// hands to the shader: it binds every map the material has, each to the
	// slot of its own number, and tells the shader which slot through the
	// place in the shader's table that belongs to that map. The fourth map is
	// free, so the coat of arms goes there.
	rl.SetMaterialTexture(fancy.material, rl.MapRoughness, fancy.flag.Texture)
	unsafe.Slice(shader.Locs, shaderLocations)[rl.ShaderLocMapRoughness] = rl.GetShaderLocation(shader, "flagTexture")

	return fancy, nil
}

// Ready reports whether the cloth has been found and read.
func (f *Fancy) Ready() bool {
	return f.ready
}

// Problem is what went wrong with a cloth that was found but could not be
// used, such as a file of a shape this cannot read.
func (f *Fancy) Problem() error {
	return f.problem
}

// prepare reads the cloth once the folders hold it. It is tried once: a
// folder that has since gained the cloth is picked up by Forget.
func (f *Fancy) prepare() {
	if f.ready || f.problem != nil {
		return
	}

	if f.model == nil && !f.loadCloth() {
		return
	}

	for index, name := range map[int32]string{
		rl.MapDiffuse:  ClothDiffuse,
		rl.MapSpecular: ClothProperties,
		rl.MapNormal:   ClothNormal,
	} {
		uploaded, ok := f.loadMap(name)
		if !ok {
			return
		}

		rl.SetMaterialTexture(f.material, index, uploaded)
	}

	f.ready = true
}

// loadMap reads one of the cloth's material maps.
func (f *Fancy) loadMap(name string) (rl.Texture2D, bool) {
	path, ok := f.resolve(name)
	if !ok {
		return rl.Texture2D{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		f.problem = fmt.Errorf("read %s: %w", path, err)

		return rl.Texture2D{}, false
	}

	picture, err := texture.Load(path, data)
	if err != nil {
		f.problem = fmt.Errorf("read %s: %w", path, err)

		return rl.Texture2D{}, false
	}
	defer rl.UnloadImage(picture)

	uploaded := rl.LoadTextureFromImage(picture)
	rl.SetTextureFilter(uploaded, rl.FilterBilinear)

	f.maps = append(f.maps, uploaded)

	return uploaded, true
}

// Forget drops the cloth, so that the next draw reads it from the folders
// again. The application does this when the folders have been read, since
// what they hold may have changed.
func (f *Fancy) Forget() {
	f.unloadCloth()

	f.ready = false
	f.problem = nil
}

// unloadCloth releases the mesh and the maps.
func (f *Fancy) unloadCloth() {
	for _, uploaded := range f.maps {
		rl.UnloadTexture(uploaded)
	}

	f.maps = nil

	if f.model != nil && f.model.VaoID != 0 {
		rl.UnloadMesh(f.model)
	}

	f.model = nil
	f.pinned.Unpin()
	f.geometry = nil
}

// loadCloth reads the mesh from the configured folders and uploads it. It
// reports whether it was there to read.
func (f *Fancy) loadCloth() bool {
	path, ok := f.resolve(ClothMesh)
	if !ok {
		return false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		f.problem = fmt.Errorf("read the cloth %s: %w", path, err)

		return false
	}

	cloth, err := mesh.Read(data)
	if err != nil {
		f.problem = fmt.Errorf("read the cloth %s: %w", path, err)

		return false
	}

	f.geometry = cloth
	f.model = &rl.Mesh{
		VertexCount:   int32(cloth.Vertices()),
		TriangleCount: int32(cloth.Triangles()),
		Vertices:      &cloth.Positions[0],
		Normals:       &cloth.Normals[0],
		Tangents:      &cloth.Tangents[0],
		Texcoords:     &cloth.UV0[0],
		Texcoords2:    &cloth.UV1[0],
		Indices:       &cloth.Indices[0],
	}

	// The arrays stay pinned for as long as the mesh lives: every draw hands
	// them back to C.
	for _, pointer := range []any{
		f.model.Vertices, f.model.Normals, f.model.Tangents,
		f.model.Texcoords, f.model.Texcoords2, f.model.Indices,
	} {
		f.pinned.Pin(pointer)
	}

	rl.UploadMesh(f.model, false)

	if f.model.VaoID == 0 {
		f.problem = fmt.Errorf("the cloth %s could not be uploaded", path)
		f.model = nil

		return false
	}

	return true
}

// Draw waves the cloth with the coat of arms on it. seconds is how long the
// window has been open, which is what the wave runs on; passing nil leaves the
// cloth empty.
//
// It has to run while raylib drawing is active and before the interface
// samples the target.
func (f *Fancy) Draw(flag *pdx.Flag, seconds float32) {
	f.prepare()
	f.drawFlag(flag)

	rl.BeginTextureMode(f.target)
	defer rl.EndTextureMode()

	rl.ClearBackground(rl.Blank)

	if flag == nil || !f.ready {
		return
	}

	rl.SetShaderValue(f.shader, f.time, []float32{seconds}, rl.ShaderUniformFloat)

	rl.BeginMode3D(fancyCamera)

	// Mirroring the cloth turns its triangles the other way round, which the
	// GPU would otherwise take for the back of a face and throw away.
	rl.DisableBackfaceCulling()
	rl.DrawMesh(*f.model, *f.material, fancyTransform)
	rl.EnableBackfaceCulling()

	rl.EndMode3D()
}

// drawFlag paints the coat of arms into the target the cloth is textured with.
func (f *Fancy) drawFlag(flag *pdx.Flag) {
	rl.BeginTextureMode(f.flag)
	defer rl.EndTextureMode()

	rl.ClearBackground(rl.Blank)

	if flag != nil {
		f.painter.Draw(*flag, rl.Rectangle{
			Width:  float32(f.flag.Texture.Width),
			Height: float32(f.flag.Texture.Height),
		})
	}
}

// Target is the render texture the cloth was drawn into.
func (f *Fancy) Target() rl.RenderTexture2D {
	return f.target
}

// Image reads the cloth back from the GPU, the right way up. It needs the
// OpenGL context, so it has to be called from the goroutine that owns the
// window.
func (f *Fancy) Image() *image.RGBA {
	return readBack(f.target.Texture, false)
}

// Unload releases everything. It requires a live OpenGL context, so it has to
// run before the window is destroyed.
func (f *Fancy) Unload() {
	f.unloadCloth()

	if f.material != nil {
		// The material's shader is this one, which is unloaded below.
		f.material.Shader = rl.Shader{}
		rl.UnloadMaterial(*f.material)
	}

	rl.UnloadRenderTexture(f.target)
	rl.UnloadRenderTexture(f.flag)
	rl.UnloadShader(f.shader)
}
