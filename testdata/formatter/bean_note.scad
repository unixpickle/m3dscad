fontSize = 2.6;
zspace = 6;
thickness = 1;

linear_extrude(thickness)
  solid()
  outset_sdf(5)
  square_sdf([40, 40], center=true);

lines = [
  "I love you so much,",
  "my favorite bean.",
  "You make me laugh, smile,",
  "and look forward to living",
  "my entire life with you.",
];

for (i = [0 : len(lines)-1])
  linear_extrude(thickness + 0.08*4)
    solid()
    outset_sdf(0.1)
    translate([0, -zspace*i + 5])
    text_sdf(lines[i], fontSize, halign="center");
  
inset_extrude(thickness + 2, top_fn="fillet", top=2)
  mesh_to_sdf()
  smooth_sq(10)
  translate([-2, 28])
  scale(0.05)
  rotate(45 + 180)
  image("bean.png");
