linear_extrude(0.5)
  rotate(180) union() {
    text("Hello,", size=1.5, valign="center", halign="center");
    translate([0, -2, 0])
      text("World!", size=1.5, valign="center", halign="center");
  }