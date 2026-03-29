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
	"transform": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleTransform,
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
	"metaball": {
		AllowChildren:   true,
		RequireChildren: true,
		NeedsChildUnion: true,
		Eval:            handleMetaball,
	},
	"weight_metaball": {
		AllowChildren:   true,
		RequireChildren: true,
		Eval:            handleWeightMetaball,
	},
	"metaball_solid": {
		AllowChildren:   true,
		RequireChildren: true,
		Eval:            handleMetaballSolid,
	},
	"sphere": {
		Eval: handleSphere,
	},
	"sphere_metaball": {
		Eval: handleSphereMetaball,
	},
	"sphere_sdf": {
		Eval: handleSphereSDF,
	},
	"cube": {
		Eval: handleCube,
	},
	"cube_metaball": {
		Eval: handleCubeMetaball,
	},
	"cube_sdf": {
		Eval: handleCubeSDF,
	},
	"cylinder": {
		Eval: handleCylinder,
	},
	"cylinder_metaball": {
		Eval: handleCylinderMetaball,
	},
	"cylinder_sdf": {
		Eval: handleCylinderSDF,
	},
	"capsule": {
		Eval: handleCapsule,
	},
	"capsule_metaball": {
		Eval: handleCapsuleMetaball,
	},
	"capsule_sdf": {
		Eval: handleCapsuleSDF,
	},
	"line_join": {
		Eval: handleLineJoin,
	},
	"circle": {
		Eval: handleCircle,
	},
	"circle_metaball": {
		Eval: handleCircleMetaball,
	},
	"circle_sdf": {
		Eval: handleCircleSDF,
	},
	"teardrop": {
		Eval: handleTeardrop,
	},
	"square": {
		Eval: handleSquare,
	},
	"square_metaball": {
		Eval: handleSquareMetaball,
	},
	"square_sdf": {
		Eval: handleSquareSDF,
	},
	"fn_solid": {
		Eval: handleFnSolid,
	},
	"polygon": {
		Eval: handlePolygon,
	},
	"polygon_sdf": {
		Eval: handlePolygonSDF,
	},
	"polygon_mesh": {
		Eval: handlePolygonMesh,
	},
	"path": {
		Eval: handlePath,
	},
	"path_sdf": {
		Eval: handlePathSDF,
	},
	"path_mesh": {
		Eval: handlePathMesh,
	},
	"text": {
		Eval: handleText,
	},
	"text_mesh": {
		Eval: handleTextMesh,
	},
}
