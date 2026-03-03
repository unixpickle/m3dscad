module heart2d(scale=1, steps=200) {
    points = [
        for (i = [0:steps])
            let(t = 360 * i / steps)
            [
                scale * 16 * pow(sin(t), 3),
                scale * (
                    13*cos(t)
                    - 5*cos(2*t)
                    - 2*cos(3*t)
                    - cos(4*t)
                )
            ]
    ];
    polygon(points);
}

linear_extrude(height=10)
    heart2d(scale=1.5);