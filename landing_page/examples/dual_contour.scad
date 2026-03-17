solid() // back to a solid
  outset_sdf(0.2) // add to the SDF
  mesh_to_sdf() // create an SDF
  dual_contour(delta=0.02) // meshify
  difference() {
    // Imagine some complex shape here which
    // can't be expressed as an SDF directly.
    cube(2, center=true);
    translate([1, 1, 1]) sphere(1);
  }