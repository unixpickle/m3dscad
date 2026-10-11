width = 100;
rounding = 10;
height = 80;
thickness = 4;

scraperDepth = 20;
scraperThickness = 2;
scraperInset = 15;

module inner_scraper() {
  module blade() {
    translate([width/2 - rounding, 0, height - scraperInset])
      rotate([-90, 0, 0])
      linear_extrude(scraperThickness, center=true)
      polygon([[0, 0], [scraperDepth, 0], [scraperDepth, scraperDepth]]);
  }
  for (theta = [0, 180])
    rotate([0, 0, theta])
      for (i = [-8:1:8])
        translate([0, i * scraperThickness * 2, 0])
          blade();
}

module bottom_scraper() {
  count = floor(width / 2 / (scraperThickness * 2));
  for (i = [-count:1:count])
    translate([0, i * scraperThickness * 2, (thickness + 1) / 2])
      cube([width, scraperThickness, thickness+1], center=true);
}

solid()
  difference() {
    linear_extrude(height)
      outset_sdf(rounding)
      square_sdf(width, center=true);

    translate([0, 0, thickness])
      linear_extrude(height)
      outset_sdf(rounding - thickness)
      square_sdf(width, center=true);
  }

inner_scraper();
bottom_scraper();
