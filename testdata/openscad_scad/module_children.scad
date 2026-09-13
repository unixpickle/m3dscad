$fn = 96;

module spread(offset) {
  for (i = [0:$children - 1])
    translate(offset * i) children(i);
}

module copy_all() {
  children();
  translate([0, 4, 0]) children();
}

r = 0.5;
copy_all() spread([3, 0, 0]) {
  r = 1;
  sphere(r=r);
  cube(2, center=true);
}
