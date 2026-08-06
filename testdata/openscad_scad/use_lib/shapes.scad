use <private.scad>

shift = 2;

module used_shape() {
    translate([shift, 0, 0]) sphere(base_radius());
}

// A use statement must not execute top-level geometry.
translate([100, 0, 0]) cube(20);
