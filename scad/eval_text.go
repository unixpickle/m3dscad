package scad

import (
	"fmt"
	"sync"

	m3dfonts "github.com/unixpickle/m3dscad/fonts"
	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/textcurve"
)

var (
	liberationFontOnce sync.Once
	liberationFont     *textcurve.ParsedFont
	liberationFontErr  error
)

func handleText(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parseTextMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(mesh.Solid()), nil
}

func handleTextMesh(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parseTextMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMesh2D(mesh), nil
}

func parseTextMesh(e *env, st *CallStmt) (*model2d.Mesh, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "text", Pos: 0, Required: true},
		{Name: "size", Pos: 1, Default: Num(10)},
		{Name: "font", Pos: 2, Default: String("Liberation Sans")},
		{Name: "halign", Pos: 3, Default: String("left")},
		{Name: "valign", Pos: 4, Default: String("baseline")},
		{Name: "spacing", Pos: 5, Default: Num(1)},
		{Name: "segments", Pos: 6, Default: Num(8)},
	})
	if err != nil {
		return nil, err
	}

	text, err := argString(args, "text", st.pos())
	if err != nil {
		return nil, err
	}
	size, err := argNum(args, "size", st.pos())
	if err != nil {
		return nil, err
	}
	font, err := argString(args, "font", st.pos())
	if err != nil {
		return nil, err
	}
	halign, err := argString(args, "halign", st.pos())
	if err != nil {
		return nil, err
	}
	valign, err := argString(args, "valign", st.pos())
	if err != nil {
		return nil, err
	}
	spacing, err := argNum(args, "spacing", st.pos())
	if err != nil {
		return nil, err
	}
	segments, err := argNum(args, "segments", st.pos())
	if err != nil {
		return nil, err
	}
	if float64(int(segments)) != segments {
		return nil, fmt.Errorf("text(): segments must be an integer")
	}
	if size <= 0 {
		return nil, fmt.Errorf("text(): size must be > 0")
	}

	if font != "Liberation Sans" && font != "Liberation Sans:style=Regular" {
		return nil, fmt.Errorf("text(): unsupported font %q", font)
	}

	align, err := parseTextAlign(halign, valign)
	if err != nil {
		return nil, err
	}

	parsed, err := getLiberationFont()
	if err != nil {
		return nil, err
	}
	outlines, err := textcurve.TextOutlines(parsed, text, textcurve.Options{
		Size:      size,
		CurveSegs: int(segments),
		Align:     align,
		Kerning:   true,
		Spacing:   spacing,
	})
	if err != nil {
		return nil, err
	}
	mesh := textcurve.OutlinesMesh(outlines)
	if mesh == nil || mesh.NumSegments() == 0 {
		return nil, fmt.Errorf("text(): no outlines produced")
	}
	return mesh, nil
}

func parseTextAlign(halign, valign string) (textcurve.Align, error) {
	var res textcurve.Align
	switch halign {
	case "left":
		res.HAlign = textcurve.HAlignLeft
	case "center":
		res.HAlign = textcurve.HAlignCenter
	case "right":
		res.HAlign = textcurve.HAlignRight
	default:
		return textcurve.Align{}, fmt.Errorf("text(): invalid halign %q", halign)
	}
	switch valign {
	case "baseline":
		res.VAlign = textcurve.VAlignBaseline
	case "top":
		res.VAlign = textcurve.VAlignTop
	case "center":
		res.VAlign = textcurve.VAlignCenter
	case "bottom":
		res.VAlign = textcurve.VAlignBottom
	default:
		return textcurve.Align{}, fmt.Errorf("text(): invalid valign %q", valign)
	}
	return res, nil
}

func getLiberationFont() (*textcurve.ParsedFont, error) {
	liberationFontOnce.Do(func() {
		liberationFont, liberationFontErr = textcurve.ParseTTF(m3dfonts.LiberationSansRegularTTF)
	})
	return liberationFont, liberationFontErr
}
