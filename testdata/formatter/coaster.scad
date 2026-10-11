size = 95;
cornerInset = 10;
moldingSize = 3;
innerInset = 10;

module basic_shape(size=size, cornerInset=cornerInset) {
  polygon_mesh([
    [0, cornerInset],
    [cornerInset, cornerInset],
    [cornerInset, 0],
    [size - cornerInset, 0],
    [size - cornerInset, cornerInset],
    [size, cornerInset],
    [size, size - cornerInset],
    [size - cornerInset, size - cornerInset],
    [size - cornerInset, size],
    [cornerInset, size],
    [cornerInset, size - cornerInset],
    [0, size - cornerInset],
  ]);
}

module inset_shape(inset) {
  translate([inset, inset])
    basic_shape(size=size - inset*2, cornerInset=cornerInset);
}

module border_shape() {
  difference() {
    basic_shape();
    inset_shape(moldingSize);
  }

  difference() {
    inset_shape(innerInset);
    inset_shape(innerInset + moldingSize);
  }
}

linear_extrude(3)
  basic_shape();

difference() {
  linear_extrude(3.8)
    difference() {
      basic_shape();
      border_shape();
    }
  translate([size / 2, size / 2, 3.4])
    linear_extrude(1)
    marching_squares(0.2)
    text("Book Club", 7, halign="center", valign="center");
}
