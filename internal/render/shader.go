package render

import (
	"fmt"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/kaiser-chris/pdx-flag-builder-go/assets"
)

// maxRecolorings is how many colour replacements the shader can do in one draw.
// It has to match MAX_RECOLORS in the shader source.
const maxRecolorings = 8

// Recoloring replaces one colour of a texture with another.
//
// Pattern and emblem textures are not painted in their final colours. They are
// painted in marker colours that say which colour slot each pixel belongs to,
// and the shader swaps those markers for the colours the coat of arms asks for.
type Recoloring struct {
	Source      color.RGBA
	Replacement color.RGBA
}

// Mask restricts a coloured emblem to the part of the flag pattern painted in
// one particular marker colour, which is what the mask attribute of a coat of
// arms does.
//
// The emblem and the pattern are drawn as separate quads with their own
// positions, scales and rotations, so the shader cannot simply reuse the
// emblem's texture coordinates to sample the pattern. UVOffset, UVAxisX and
// UVAxisY carry the mapping from one to the other:
//
//	patternUV = UVOffset + u*UVAxisX + v*UVAxisY
type Mask struct {
	Texture  rl.Texture2D
	Color    color.RGBA
	UVOffset [2]float32
	UVAxisX  [2]float32
	UVAxisY  [2]float32
}

// Recolor is the recolouring shader together with the locations of its uniforms.
type Recolor struct {
	shader rl.Shader

	recolorCount       int32
	sourceColors       int32
	replacementColors  int32
	tolerance          int32
	preserveShading    int32
	blueChannelShading int32
	useMask            int32
	maskTexture        int32
	maskColor          int32
	maskUVOffset       int32
	maskUVAxisX        int32
	maskUVAxisY        int32
}

// LoadRecolor compiles the recolouring shader. It needs an OpenGL context, so
// it has to be called after the window exists.
func LoadRecolor() (Recolor, error) {
	source, err := assets.Read(assets.ShaderRecolor)
	if err != nil {
		return Recolor{}, err
	}

	// An empty vertex shader means the default one, which is all this shader
	// needs: only the fragment stage does anything unusual.
	shader := rl.LoadShaderFromMemory("", string(source))

	if shader.ID == 0 {
		return Recolor{}, fmt.Errorf("the recolouring shader did not compile")
	}

	return Recolor{
		shader: shader,

		// The first element of an array uniform is what the location is asked
		// for; the rest follow from it.
		recolorCount:       rl.GetShaderLocation(shader, "recolorCount"),
		sourceColors:       rl.GetShaderLocation(shader, "sourceColors[0]"),
		replacementColors:  rl.GetShaderLocation(shader, "replacementColors[0]"),
		tolerance:          rl.GetShaderLocation(shader, "tolerance"),
		preserveShading:    rl.GetShaderLocation(shader, "preserveShading"),
		blueChannelShading: rl.GetShaderLocation(shader, "blueChannelShading"),
		useMask:            rl.GetShaderLocation(shader, "useMask"),
		maskTexture:        rl.GetShaderLocation(shader, "maskTexture"),
		maskColor:          rl.GetShaderLocation(shader, "maskColor"),
		maskUVOffset:       rl.GetShaderLocation(shader, "maskUVOffset"),
		maskUVAxisX:        rl.GetShaderLocation(shader, "maskUVAxisX"),
		maskUVAxisY:        rl.GetShaderLocation(shader, "maskUVAxisY"),
	}, nil
}

// Unload releases the shader.
func (r Recolor) Unload() {
	if r.shader.ID != 0 {
		rl.UnloadShader(r.shader)
	}
}

// DrawOptions are the parts of a recoloured draw that vary between a pattern
// and an emblem.
type DrawOptions struct {
	Origin   rl.Vector2
	Rotation float32

	// Recolorings are applied in order, and the first one that matches a pixel
	// wins.
	Recolorings []Recoloring

	// BlueChannelShading is set for coloured emblems, whose textures carry a
	// per pixel brightness in the blue channel instead of part of the marker
	// colour, so only red and green identify the slot.
	BlueChannelShading bool

	// Mask is set when the emblem is restricted to part of the pattern.
	Mask *Mask
}

// Draw draws a texture with its marker colours replaced.
func (r Recolor) Draw(texture rl.Texture2D, source, destination rl.Rectangle, options DrawOptions) {
	// The shader has to be bound before its uniforms are set, because setting a
	// sampler uniform for the mask acts on whichever program is bound, unlike
	// the other uniforms which bind the shader themselves.
	rl.BeginShaderMode(r.shader)
	defer rl.EndShaderMode()

	r.apply(options)

	rl.DrawTexturePro(texture, source, destination, options.Origin, options.Rotation, rl.White)
}

func (r Recolor) apply(options DrawOptions) {
	r.setMask(options.Mask)

	count := min(len(options.Recolorings), maxRecolorings)
	if count <= 0 {
		r.setInt(r.recolorCount, 0)

		return
	}

	// Laid out as a flat run of three floats per colour, which is how the
	// shader reads an array of vec3.
	sources := make([]float32, 0, count*3)
	replacements := make([]float32, 0, count*3)

	for _, recoloring := range options.Recolorings[:count] {
		sources = append(sources, channels(recoloring.Source)...)
		replacements = append(replacements, channels(recoloring.Replacement)...)
	}

	r.setInt(r.recolorCount, int32(count))
	rl.SetShaderValueV(r.shader, r.sourceColors, sources, rl.ShaderUniformVec3, int32(count))
	rl.SetShaderValueV(r.shader, r.replacementColors, replacements, rl.ShaderUniformVec3, int32(count))

	rl.SetShaderValue(r.shader, r.tolerance, []float32{defaultTolerance}, rl.ShaderUniformFloat)
	rl.SetShaderValue(r.shader, r.preserveShading, []float32{0}, rl.ShaderUniformFloat)

	r.setBool(r.blueChannelShading, options.BlueChannelShading)
}

// defaultTolerance is how far a pixel may be from a marker colour and still
// count as that slot. The textures are not painted in exactly the marker
// colours once they have been through block compression.
const defaultTolerance = 0.8

func (r Recolor) setMask(mask *Mask) {
	r.setBool(r.useMask, mask != nil)

	if mask == nil {
		return
	}

	rl.SetShaderValueTexture(r.shader, r.maskTexture, mask.Texture)
	rl.SetShaderValue(r.shader, r.maskColor, channels(mask.Color), rl.ShaderUniformVec3)
	rl.SetShaderValue(r.shader, r.maskUVOffset, mask.UVOffset[:], rl.ShaderUniformVec2)
	rl.SetShaderValue(r.shader, r.maskUVAxisX, mask.UVAxisX[:], rl.ShaderUniformVec2)
	rl.SetShaderValue(r.shader, r.maskUVAxisY, mask.UVAxisY[:], rl.ShaderUniformVec2)
}

func (r Recolor) setBool(location int32, value bool) {
	if value {
		r.setInt(location, 1)

		return
	}

	r.setInt(location, 0)
}

// setInt passes an integer through an interface that only takes floats.
//
// raylib reinterprets the bytes it is handed according to the uniform type, and
// the Go binding types that pointer as a float, so an integer has to travel as
// the float with the same bit pattern.
func (r Recolor) setInt(location int32, value int32) {
	rl.SetShaderValue(r.shader, location, []float32{math.Float32frombits(uint32(value))}, rl.ShaderUniformInt)
}

// channels turns a colour into the zero to one triple the shader works in.
func channels(value color.RGBA) []float32 {
	return []float32{
		float32(value.R) / 255,
		float32(value.G) / 255,
		float32(value.B) / 255,
	}
}
