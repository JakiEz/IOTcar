// Dependency-free line chart. Inline SVG keeps the bundle small — no chart lib.
export default function Sparkline({ data, height = 70 }) {
  if (!data || data.length < 2) return null;

  const w = 100; // viewBox width; SVG scales to container via width:100%
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const stepX = w / (data.length - 1);

  const pts = data.map((v, i) => {
    const x = i * stepX;
    const y = height - ((v - min) / range) * (height - 8) - 4;
    return [x, y];
  });

  const line = pts.map(([x, y]) => `${x},${y}`).join(" ");
  const area = `0,${height} ${line} ${w},${height}`;

  return (
    <svg
      viewBox={`0 0 ${w} ${height}`}
      preserveAspectRatio="none"
      style={{ width: "100%", height, display: "block" }}
    >
      <polygon
        points={area}
        fill="color-mix(in oklab, var(--primary) 18%, transparent)"
      />
      <polyline
        points={line}
        fill="none"
        stroke="var(--primary)"
        strokeWidth="1.5"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}
