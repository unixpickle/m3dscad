package scad

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/unixpickle/model3d/model3d"
)

func TestCacheEcho(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "RepeatedCallsAndHoisting",
			src: `
				saved();
				cache saved() { echo(1); sphere(1); }
				saved();
				translate([4, 0, 0]) saved();
			`,
			want: []string{"1"},
		},
		{
			name: "Unused",
			src:  `cache unused() { echo(1); assert(false); } sphere(1);`,
		},
		{
			name: "EmptyResult",
			src:  `cache empty() { echo(1); } empty(); empty();`,
			want: []string{"1"},
		},
		{
			name: "FreshPerModuleCall",
			src: `
				module outer(r) {
					cache saved() { echo(r); sphere(r); }
					saved();
					saved();
				}
				outer(1);
				outer(2);
			`,
			want: []string{"1", "2"},
		},
		{
			name: "FreshPerLoopIteration",
			src: `
				for (r = [1, 2]) {
					cache saved() { echo(r); sphere(r); }
					saved();
					saved();
				}
			`,
			want: []string{"1", "2"},
		},
		{
			name: "SharedAcrossCallerScopes",
			src: `
				cache saved() { echo(r); sphere(r); }
				r = 2;
				module outer(r) { saved(); }
				outer(10);
				outer(20);
				saved();
			`,
			want: []string{"2"},
		},
		{
			name: "ShadowedDeclarations",
			src: `
				cache saved() { echo(1); sphere(1); }
				{
					cache saved() { echo(2); sphere(2); }
					saved();
					saved();
				}
				saved();
				saved();
			`,
			want: []string{"2", "1"},
		},
		{
			name: "DeclarationInChildBlock",
			src: `
				module wrapper() { assert($children == 2); children(); }
				wrapper() {
					cache saved() { echo(1); sphere(1); }
					saved();
					saved();
				}
			`,
			want: []string{"1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatal(err)
			}
			var msgs []string
			if _, err := evalStmts(echoTester(&msgs), prog.Stmts); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(msgs, tc.want) {
				t.Fatalf("echo: got %v, want %v", msgs, tc.want)
			}
		})
	}
}

func TestCacheCapturesOuterChildren(t *testing.T) {
	prog, err := Parse(`
		module copies(dx) {
			cache saved() {
				echo(dx);
				assert($children == 1);
				children();
			}
			saved();
			translate([dx, 0, 0]) saved();
		}
		copies(4) sphere(1);
		translate([0, 10, 0]) copies(6) sphere(2);
	`)
	if err != nil {
		t.Fatal(err)
	}
	// Evaluating the same parsed program again must create fresh caches too.
	for i := 0; i < 2; i++ {
		var msgs []string
		shape, err := Eval(prog, Hooks{Echo: func(msg string) { msgs = append(msgs, msg) }})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(msgs, []string{"4", "6"}) {
			t.Fatalf("evaluation %d: unexpected echoes %v", i, msgs)
		}
		for _, p := range []model3d.Coord3D{
			model3d.Origin, model3d.XYZ(4.9, 0, 0),
			model3d.XYZ(0, 11.9, 0), model3d.XYZ(7.9, 10, 0),
		} {
			assertContains(t, shape.S3, p, true)
		}
		for _, p := range []model3d.Coord3D{
			model3d.XYZ(5.1, 0, 0), model3d.XYZ(0, 12.1, 0), model3d.XYZ(8.1, 10, 0),
		} {
			assertContains(t, shape.S3, p, false)
		}
	}
}

func TestCacheThroughRepeatedChildren(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "CacheInChildBlock",
			src: `
				module useMoreThanOnce() {
					children();
					translate([4, 0, 0]) children();
				}
				useMoreThanOnce() {
					cache foobar() { someModule(); }
					foobar();
				}
			`,
			want: []string{`"hi"`, `"hi"`},
		},
		{
			name: "CacheOutsideChildBlock",
			src: `
				module useMoreThanOnce() {
					children();
					translate([4, 0, 0]) children();
				}
				cache foobar() { someModule(); }
				useMoreThanOnce() { foobar(); }
			`,
			want: []string{`"hi"`},
		},
		{
			name: "CacheInWrapper",
			src: `
				module useMoreThanOnce() {
					cache foobar() { children(); }
					foobar();
					translate([4, 0, 0]) foobar();
				}
				useMoreThanOnce() someModule();
			`,
			want: []string{`"hi"`},
		},
		{
			name: "IndexedAndAllChildrenRecreateCache",
			src: `
				module useMoreThanOnce() {
					children(1);
					translate([4, 0, 0]) children();
					children(0);
				}
				useMoreThanOnce() {
					cache foobar() { someModule(); }
					foobar();
					foobar();
				}
			`,
			want: []string{`"hi"`, `"hi"`, `"hi"`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(`module someModule() { echo("hi"); sphere(1); }` + tc.src)
			if err != nil {
				t.Fatal(err)
			}
			var msgs []string
			shape, err := Eval(prog, Hooks{Echo: func(msg string) { msgs = append(msgs, msg) }})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(msgs, tc.want) {
				t.Fatalf("echo: got %v, want %v", msgs, tc.want)
			}
			assertContains(t, shape.S3, model3d.Origin, true)
			assertContains(t, shape.S3, model3d.XYZ(4, 0, 0), true)
			assertContains(t, shape.S3, model3d.XYZ(2, 0, 0), false)
		})
	}
}

func TestCacheChildBlockScopeLifetime(t *testing.T) {
	prog, err := Parse(`
		module useMoreThanOnce() {
			assert($children == 2);
			children();
			translate([4, 0, 0]) children();
		}
		for (r = [1, 2]) {
			useMoreThanOnce() {
				cache saved() { echo(r); sphere(r); }
				echo(r + 10);
				saved();
			}
		}
	`)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	if _, err := evalStmts(echoTester(&msgs), prog.Stmts); err != nil {
		t.Fatal(err)
	}
	// Every child selection reevaluates the block in a fresh scope, including
	// its cache declarations and ordinary statements.
	want := []string{"11", "1", "11", "1", "12", "2", "12", "2"}
	if !reflect.DeepEqual(msgs, want) {
		t.Fatalf("echo: got %v, want %v", msgs, want)
	}
}

func TestCacheReusesMesh(t *testing.T) {
	prog, err := Parse(`cache saved() { marching_cubes() sphere(1); } saved(); saved();`)
	if err != nil {
		t.Fatal(err)
	}
	mesh := model3d.NewMeshRect(model3d.Origin, model3d.Ones(1))
	var calls int
	e := newEnv(Hooks{MarchingCubes: func(ShapeRep, float64, int) (*model3d.Mesh, error) {
		calls++
		return mesh, nil
	}})
	if _, err := evalStmt(e, prog.Stmts[0]); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("cache evaluated at declaration time")
	}
	first, err := evalStmt(e, prog.Stmts[1])
	if err != nil {
		t.Fatal(err)
	}
	second, err := evalStmt(e, prog.Stmts[2])
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || first == nil || first != second || first.M3 != mesh {
		t.Fatalf("expected the same cached shape and mesh from one meshing call; calls=%d", calls)
	}
}

func TestCacheFromUsedFile(t *testing.T) {
	prog, err := ParseFile("main.scad", `use <lib.scad> saved(); wrapper(); saved();`)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	e := echoTester(&msgs)
	e.hooks.FS = fstest.MapFS{"lib.scad": {Data: []byte(`
		r = 2;
		cache saved() { echo(r); sphere(r); }
		module wrapper() { saved(); }
	`)}}
	shapes, err := evalStmts(e, prog.Stmts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(msgs, []string{"2"}) {
		t.Fatalf("expected imported calls to share the cache, got echoes %v", msgs)
	}
	if len(shapes) != 3 {
		t.Fatalf("expected three shapes, got %d", len(shapes))
	}
	for _, shape := range shapes {
		assertContains(t, shape.S3, model3d.XYZ(1.9, 0, 0), true)
		assertContains(t, shape.S3, model3d.XYZ(2.1, 0, 0), false)
	}
}

func TestCacheErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{"Parameter", `cache saved(r) { sphere(r); }`, "cache declarations cannot have parameters"},
		{"DefaultParameter", `cache saved(r=1) { sphere(r); }`, "cache declarations cannot have parameters"},
		{"Argument", `cache saved() { sphere(1); } saved(1);`, "does not take arguments"},
		{"NamedArgument", `cache saved() { sphere(1); } saved(r=1);`, "does not take arguments"},
		{"ArgumentAfterHit", `cache saved() { sphere(1); } saved(); saved(1);`, "does not take arguments"},
		{"Children", `cache saved() { sphere(1); } saved() sphere(2);`, "does not take children"},
		{"ChildrenAfterHit", `cache saved() { sphere(1); } saved(); saved() sphere(2);`, "does not take children"},
		{"Recursion", `cache saved() { saved(); } saved();`, "recursive evaluation of cache saved()"},
		{"MutualRecursion", `cache a() { b(); } cache b() { a(); } a();`, "recursive evaluation of cache a()"},
		{"Redeclaration", `module saved() {} cache saved() {}`, `cannot redeclare module "saved"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err == nil {
				_, err = Eval(prog, Hooks{})
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
			if countErrorPositions(err) == 0 {
				t.Fatalf("missing error position: %v", err)
			}
		})
	}
}
