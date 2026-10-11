domeHeight = 40;
domeRadius = 33;
domeThickness = 1.6;

module band(h, r) {
  bandRad = 2;
  rotate_extrude(360)
    translate([r, h])
    circle(bandRad);
}

module body() {
  clip(min_z=-5)
    band(h=-4, r=40);
  band(h=-2, r=39);
  band(h=0, r=38);
  band(h=15, r=34);
  translate([0, 0, -5])
    cylinder(r1=40, r2=38, h=5);

  // Inset rounded base.
  rotate_extrude(360)
    difference() {
      square([40, 15]);
      translate([47, 15 / 2])
        rotate(30)
        scale([0.7, 1])
        circle(r=15);
    }

  module traySideOutline() {
    translate([-7, -5])
      square_sdf([14, 5]);
    clip(min_y=0)
      circle_sdf(7);
  }

  // Tall part of body with cutouts
  difference() {
    translate([0, 0, 17])
      cylinder(r1=34, r2=29, h=55);

    // Cutout for dispenser section
    translate([0, 24, 26])
      rotate([-90, 180, 0])
      linear_extrude(20)
      solid()
      traySideOutline();
  }
  band(h=72, r=28);

  // Dispenser tray
  module trayOutline() {
    translate([-7, -10])
      square_sdf([14, 10]);
    circle_sdf(7);
  }
  translate([0, 34, 20])
    difference() {
      linear_extrude(5)
        solid()
        outset_sdf(1.2)
        trayOutline();
      translate([0, 0, 1.6])
        linear_extrude(5)
        solid()
        trayOutline();
    }

  translate([0, 24, 26])
    rotate([-90, 180, 0])
    linear_extrude(9)
    difference() {
      solid() outset_sdf(1.2) traySideOutline();
      solid() traySideOutline();
    }
}

module controlPanel(outset=0) {
  module bean() {
    translate([0, 9, 0])
      rotate([90, 0, 0])
      inset_extrude(1, bottom_fn="chamfer", bottom=0.2)
      scale(0.01)
      path_sdf("M 74.936 326.929 L 89.216 325.216 L 114.072 318.072 L 131.967 309.967 L 154.189 296.189 L 182.746 272.746 L 195.691 258.691 L 209.189 241.189 L 226.232 213.232 L 235.988 190.988 L 243.785 165.785 L 247.298 147.298 L 247.703 124.297 L 241.297 99.703 L 230.625 82.375 L 223.625 75.375 L 209.532 66.468 L 190.93 62.071 L 174.572 62.572 L 160.349 67.349 L 141.374 81.374 L 129.359 99.359 L 112.641 145.641 L 103.546 159.546 L 90.691 173.691 L 74.772 185.772 L 46.476 201.476 L 32.375 211.375 L 20.497 223.497 L 9.801 243.801 L 6.292 261.289 L 8.478 282.522 L 17.168 300.832 L 36.751 318.249 L 57.523 325.477 Z");
  }
  translate([0, 24, 39]) {
    solid()
      outset_sdf(1 + outset)
      translate([-14, 0, 0])
      cube_sdf([28, 7.5, 28]);

    bean();
    translate([-7, 0, 2]) rotate([0, -70, 0]) bean();
    translate([7, 0, 4]) rotate([0, 60, 0]) bean();
    translate([-13, 0, 15]) rotate([0, 90, 0]) bean();
    translate([12, 0, 12]) rotate([0, 180, 0]) bean();
    translate([12, 0, 17]) rotate([0, -90, 0]) bean();
    translate([-6, 0, 19]) rotate([0, -110, 0]) bean();
    translate([5, 0, 23]) rotate([0, 30, 0]) bean();
    translate([-8, 0, 27]) rotate([0, 180, 0]) bean();
    translate([10, 0, 27]) rotate([0, 110, 0]) bean();
    translate([-8, 0, 10]) rotate([0, 200, 0]) bean();

    translate([0, 7.5, 10])
    rotate([-90, 180, 0])
      union() {
        linear_extrude(2)
          difference() {
            union() {
              translate([0, 10]) circle(r=5);
              translate([-5, 5])
                square([5, 5]);
              translate([0, 2])
                circle(r=8);
            }
            translate([0, 10]) circle(r=4);
          }
        inset_extrude(4, top_fn="chamfer", top=0.2) {
          translate([0, 2]) circle_sdf(r=2);
          translate([0, 2]) square_sdf([10, 1.5], center=true);
        }
      }
  }
}

module controlPanelSpokes(outset=0) {
  for (dx = [-10, 10])
    for (dz = [0, 18])
      translate([0 + dx, 20, 44 + dz])
        rotate([0, 45, 0])
        translate([-1 - outset/2, 0, -1 - outset/2])
        cube([2 + outset, 5, 2 + outset]);
}

module screw(r, h, segments=4096, pitch=1.5) {
  profile = [
    [0, 1],
    [0.5, 0],
    [1, 1],
  ];
  linear_extrude(h, twist=-360 * h / pitch)
    polygon([
      for (i = [0:segments])
        let (
          theta = i / segments * 360,
          y = lookup(theta / 360, profile),
          rad = r - pitch/2*y,
        ) [cos(theta) * rad, sin(theta) * rad]
    ]);
}

module domeScrew(outset=0) {
  translate([0, 0, 69])
    screw(r=27 + outset, h=4);
}

module dome() {
  domeScrew();
  clip(min_z=73, max_z=73 + domeHeight)
    translate([0, 0, 73 + domeHeight/2])
    intersection() {
      difference() {
        sphere(r=domeRadius);
        sphere(r=domeRadius - domeThickness);
      }
      union() {
        for (theta = [0:15:360])
          rotate([0, 0, theta])
            cube([100, 2, 100], center=true);
  
        for (z = [-12, 10, 19])
          translate([0, 0, z])
            cube([100, 100, 2], center=true);
      }
    }
}

module lid() {
  baseZ = 73 + domeHeight;
  topRadius = 24.5;
  translate([0, 0, baseZ - 2])
    cylinder(r=topRadius, h=3);
  clip(min_z=baseZ)
    band(h=baseZ, r=topRadius);
  clip(min_z=baseZ)
    translate([0, 0, baseZ])
    scale([1, 1, 0.5])
      sphere(r=topRadius + 0.3);
  translate([0, 0, baseZ + topRadius/2 + 2])
    sphere(r=3);
}

module bean3d() {
  scale(3)
    inset_extrude(2.4, bottom_fn="fillet", top_fn="fillet", bottom=1.1, top=1.3)
    scale(0.02)
    path_sdf("M 68.152 326.847 L 62.704 326.296 L 57.523 325.477 L 51.039 323.961 L 45.021 321.979 L 40.079 319.921 L 36.751 318.249 L 31.82 315.18 L 28.386 312.614 L 26.273 310.727 L 20.31 304.69 L 17.168 300.832 L 15.533 298.467 L 13.352 294.648 L 11.083 289.917 L 9.937 287.063 L 8.478 282.522 L 7.337 277.663 L 6.744 274.256 L 6.161 268.853 L 6.092 265.043 L 6.292 261.289 L 7.223 254.223 L 7.86 250.86 L 9.801 243.801 L 12.072 238.072 L 15.743 230.743 L 18.82 225.82 L 20.497 223.497 L 24.254 219.254 L 29.254 214.254 L 32.375 211.375 L 38.628 206.628 L 46.476 201.476 L 56.735 195.735 L 64.654 191.654 L 69.806 188.806 L 74.772 185.772 L 79.544 182.544 L 83.546 179.546 L 90.691 173.691 L 97.691 166.691 L 103.546 159.546 L 106.544 155.544 L 109.764 150.764 L 112.641 145.641 L 115.128 140.128 L 117.083 135.083 L 120.08 126.08 L 123.395 114.395 L 124.918 109.918 L 126.872 104.872 L 129.359 99.359 L 132.194 94.194 L 134.464 90.464 L 137.348 86.348 L 141.374 81.374 L 145.374 77.374 L 150.348 73.348 L 155.745 69.745 L 160.349 67.349 L 165.937 64.937 L 168.932 63.932 L 174.572 62.572 L 181.043 62.044 L 190.93 62.071 L 194.673 62.327 L 198.127 62.873 L 203.691 64.309 L 209.532 66.468 L 212.943 68.057 L 217.35 70.65 L 223.625 75.375 L 227.744 79.256 L 230.625 82.375 L 234.536 87.464 L 237.255 91.745 L 241.297 99.703 L 243.063 103.937 L 244.072 106.928 L 245.778 113.222 L 246.975 119.025 L 247.703 124.297 L 247.993 132.007 L 247.933 139.933 L 247.298 147.298 L 245.908 155.908 L 243.785 165.785 L 242.109 172.109 L 240.083 179.083 L 238.12 185.12 L 235.988 190.988 L 233.71 196.71 L 231.302 202.302 L 226.232 213.232 L 221.266 222.266 L 216.773 229.773 L 209.189 241.189 L 202.418 250.418 L 195.691 258.691 L 190.37 264.37 L 187.63 267.63 L 182.746 272.746 L 171.631 282.631 L 165.64 287.64 L 158.854 292.854 L 154.189 296.189 L 146.989 300.989 L 143.307 303.307 L 134.542 308.542 L 131.967 309.967 L 123.931 313.931 L 114.072 318.072 L 105.12 321.12 L 100.534 322.534 L 94.171 324.171 L 89.216 325.216 L 82.298 326.298 L 74.936 326.929 Z");
}

/*union() {
  difference() {
    body();
    controlPanel(outset=0.1);
    translate([0, 0, 1])
      domeScrew(outset=0.4);
  }
  controlPanelSpokes();
}*/

/*difference() {
  controlPanel();
  controlPanelSpokes(outset=0.2);
}*/

//dome();
//lid();

//bean3d();
