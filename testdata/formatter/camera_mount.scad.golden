pluginThickness = 5;
pluginWidth = 90 + 0.2;
pluginHeight = 100 + 0.2;

poleHeight = 200;
poleRadius = 10;
poleSpikeRadius = 4;
poleSpikeLength = 7;

holderWidth = 40;
holderGap = 12;
holderGripSize = 3;
holderGripDepth = 15;
holderAngle = 20;

module emptyPlugin() {
  translate([0, 0, pluginThickness])
    rotate([-90, 0, 0])
    linear_extrude(pluginHeight)
    polygon([
      [0, 0],
      [-pluginThickness / 2, pluginThickness / 2],
      [0, pluginThickness],
      [pluginWidth, pluginThickness],
      [pluginWidth + pluginThickness/2, pluginThickness / 2],
      [pluginWidth, 0],
    ]);
}

module pole() {
  emptyPlugin();
  translate([pluginWidth / 2, pluginHeight / 2, pluginThickness]) {
    cylinder(h=poleHeight, r=poleRadius);
    translate([0, 0, poleHeight])
      cylinder(h=poleSpikeLength, r=poleSpikeRadius);
  }
}

module topPart() {
  h = poleSpikeLength + 20;
  difference() {
    clip(min_z=0)
      translate([0, 0, -8])
      rotate([holderAngle, 0, 0]) {
        cube([holderWidth, holderGap + holderGripSize*2, h]);
        cube([holderWidth, holderGripSize, h + holderGripDepth]);
        translate([0, holderGap + holderGripSize, 0])
          cube([holderWidth, holderGripSize, h + holderGripDepth]);
      }
    translate([holderWidth / 2, holderGap/2 + holderGripSize, 0])
      cylinder(h=poleSpikeLength + 0.5, r=poleSpikeRadius);
  }
}

//pole();
topPart();
