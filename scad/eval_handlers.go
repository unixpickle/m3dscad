package scad

type callHandler struct {
	AllowChildren   bool
	RequireChildren bool
	NeedsChildUnion bool
	Eval            func(e *env, st *CallStmt, children []SolidValue, childUnion *SolidValue) (SolidValue, error)
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
	"sphere": {
		Eval: handleSphere,
	},
	"cube": {
		Eval: handleCube,
	},
	"cylinder": {
		Eval: handleCylinder,
	},
	"circle": {
		Eval: handleCircle,
	},
	"square": {
		Eval: handleSquare,
	},
}
