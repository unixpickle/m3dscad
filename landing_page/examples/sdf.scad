solid()
  outset_sdf(0.1) // biases the field
  union() {
    translate([0, 0, 0.5])
        cylinder_sdf(r=0.2, h=0.5);
    cube_sdf(1, center=true);
  }