package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/unixpickle/model3d/model3d"

	"github.com/unixpickle/m3dscad/scad"
)

func main() {
	inPath := flag.String("in", "", "Input .scad-like file")
	outPath := flag.String("out", "out.stl", "Output STL path")
	delta := flag.Float64("delta", 0.02, "Marching cubes resolution (smaller = finer)")
	subdiv := flag.Int("subdiv", 8, "Marching cubes search subdivisions")
	flag.Parse()

	if *inPath == "" {
		fmt.Fprintln(os.Stderr, "missing -in")
		os.Exit(2)
	}

	srcBytes, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}

	prog, err := scad.Parse(string(srcBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}

	solid, err := scad.Eval(prog)
	if err != nil {
		fmt.Fprintln(os.Stderr, "eval:", err)
		os.Exit(1)
	}

	// model3d: solid -> mesh via marching cubes, then save STL. :contentReference[oaicite:1]{index=1}
	mesh := model3d.MarchingCubesSearch(solid, *delta, *subdiv)
	if err := mesh.SaveGroupedSTL(*outPath); err != nil {
		fmt.Fprintln(os.Stderr, "save stl:", err)
		os.Exit(1)
	}
	fmt.Println("wrote:", *outPath)
}
