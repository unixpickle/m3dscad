$fn=100;
linear_extrude(height=2)
union() {
  square(size=[2, 2], center=false);
  translate([2, 2, 0]) circle(r=1);
}
