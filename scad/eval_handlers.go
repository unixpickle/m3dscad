package scad

type callHandler struct {
	AllowChildren   bool
	RequireChildren bool
	NeedsChildUnion bool
	Eval            func(e *env, st *CallStmt, children []ShapeRep, childUnion *ShapeRep) (ShapeRep, error)
}

var builtinHandlers = map[string]callHandler{
	"union": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleUnion,
	},
	"difference": {
		AllowChildren:   true,
		RequireChildren: true,
		Eval:            handleDifference,
	},
	"intersection": {
		AllowChildren:   true,
		RequireChildren: true,
		Eval:            handleIntersection,
	},
	"translate": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleTranslate,
	},
	"scale": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleScale,
	},
	"rotate": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleRotate,
	},
	"linear_extrude": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleLinearExtrude,
	},
	"rotate_extrude": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleRotateExtrude,
	},
	"marching_squares": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleMarchingSquares,
	},
	"marching_cubes": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleMarchingCubes,
	},
	"dual_contour": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleDualContour,
	},
	"mesh_to_sdf": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleMeshToSDF,
	},
	"inset_sdf": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleInsetSDF,
	},
	"outset_sdf": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleOutsetSDF,
	},
	"solid": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleSolid,
	},
	"sphere": {
		Eval: handleSphere,
	},
	"sphere_sdf": {
		Eval: handleSphereSDF,
	},
	"cube": {
		Eval: handleCube,
	},
	"cube_sdf": {
		Eval: handleCubeSDF,
	},
	"cylinder": {
		Eval: handleCylinder,
	},
	"cylinder_sdf": {
		Eval: handleCylinderSDF,
	},
	"circle": {
		Eval: handleCircle,
	},
	"circle_sdf": {
		Eval: handleCircleSDF,
	},
	"square": {
		Eval: handleSquare,
	},
	"square_sdf": {
		Eval: handleSquareSDF,
	},
	"polygon": {
		Eval: handlePolygon,
	},
	"text": {
		Eval: handleText,
	},
	"text_mesh": {
		Eval: handleTextMesh,
	},
}
