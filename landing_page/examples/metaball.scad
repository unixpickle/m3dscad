metaball_solid(1, falloff="gaussian") {
  // Sofa body
  cube_metaball([4, 2, 2], center=true);

  // Sofa cutout
  weight_metaball(-1)
    translate([0, 1, 1.3])
    cube_metaball([2.8, 2, 2], center=true);
}
