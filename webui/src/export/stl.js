export function buildBinarySTL(positions, normals) {
  const triCount = Math.floor(positions.length / 9);
  const buffer = new ArrayBuffer(84 + triCount * 50);
  const view = new DataView(buffer);
  let offset = 80;
  view.setUint32(offset, triCount, true);
  offset += 4;
  for (let i = 0; i < triCount; i++) {
    const nIdx = i * 9;
    const pIdx = i * 9;
    const n0 = normals[nIdx] ?? 0;
    const n1 = normals[nIdx + 1] ?? 0;
    const n2 = normals[nIdx + 2] ?? 0;
    view.setFloat32(offset, n0, true);
    view.setFloat32(offset + 4, n1, true);
    view.setFloat32(offset + 8, n2, true);
    offset += 12;
    for (let v = 0; v < 9; v++) {
      view.setFloat32(offset, positions[pIdx + v] ?? 0, true);
      offset += 4;
    }
    view.setUint16(offset, 0, true);
    offset += 2;
  }
  return buffer;
}
