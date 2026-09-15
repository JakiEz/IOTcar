import { packStats } from "@/data/mockTruck";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

// Charge → CSS-var color. Green healthy, amber getting low, red critical.
function cellColor(pct) {
  if (pct >= 60) return "var(--ok)";
  if (pct >= 30) return "var(--warn)";
  return "var(--bad)";
}

function Chip({ label, value, alert }) {
  return (
    <div className="flex flex-col gap-0.5 rounded-lg border bg-secondary/50 px-2.5 py-2">
      <span className="text-xs text-muted-foreground">{label}</span>
      <strong className={cn("text-sm", alert && "text-bad")}>{value}</strong>
    </div>
  );
}

export default function BatteryPanel({ truck, className }) {
  const { totalPct, minCell, maxCell, deltaMv, maxTempC } = packStats(truck);
  const cells = truck.battery.cells;

  return (
    <Card className={cn("flex flex-col gap-3 overflow-hidden p-4", className)}>
      <div className="flex items-center justify-between">
        <h2 className="text-[15px] font-semibold">Battery Pack</h2>
        <span className="text-xs text-muted-foreground">{cells.length} cells</span>
      </div>

      {/* Hero: total charge, like the big "arrival" stat in the reference */}
      <div className="flex items-center gap-4 rounded-xl bg-secondary/50 px-4 py-3.5">
        <div className="text-5xl leading-none font-bold" style={{ color: cellColor(totalPct) }}>
          {totalPct}
          <em className="ml-0.5 text-xl not-italic">%</em>
        </div>
        <div className="flex flex-col gap-1 text-[13px]">
          <div><span className="inline-block w-16 text-muted-foreground">Pack</span> {truck.battery.packVoltage} V</div>
          <div><span className="inline-block w-16 text-muted-foreground">Current</span> {truck.battery.packCurrentA} A</div>
          <div><span className="inline-block w-16 text-muted-foreground">Range</span> {truck.energy.rangeKm} km</div>
        </div>
      </div>

      {/* Health stats that actually matter for a battery: spread + temp */}
      <div className="grid grid-cols-2 gap-2">
        <Chip label="Cell Δ" value={`${deltaMv} mV`} alert={deltaMv > 150} />
        <Chip label="Max temp" value={`${maxTempC}°C`} alert={maxTempC > 45} />
        <Chip label="Lowest" value={`#${minCell.id} · ${minCell.voltage}V`} />
        <Chip label="Highest" value={`#${maxCell.id} · ${maxCell.voltage}V`} />
      </div>

      {/* Cell grid — the reference's cargo grid, repurposed as the BMS view.
          Auto-fits however many cells the pack reports. */}
      <div className="grid grid-cols-[repeat(auto-fill,minmax(64px,1fr))] content-start gap-2 overflow-y-auto">
        {cells.map((c) => {
          const isLow = c.id === minCell.id;
          const isHigh = c.id === maxCell.id;
          return (
            <div
              key={c.id}
              className={cn(
                "relative flex flex-col items-center gap-0.5 overflow-hidden rounded-lg border bg-secondary/50 px-1.5 pb-1.5 pt-2",
                isLow && "border-bad",
                isHigh && "border-ok"
              )}
              title={`Cell ${c.id}: ${c.voltage}V · ${c.pct}% · ${c.tempC}°C`}
            >
              {/* charge fill rises from the bottom of each cell */}
              <span
                className="absolute inset-x-0 bottom-0 opacity-20"
                style={{ height: `${c.pct}%`, background: cellColor(c.pct) }}
              />
              <span className="relative z-10 text-[11px] text-muted-foreground">{c.id}</span>
              <span className="relative z-10 text-[13px] font-semibold">{c.voltage}V</span>
              <span className="relative z-10 text-[10px] text-muted-foreground">{c.tempC}°</span>
            </div>
          );
        })}
      </div>
    </Card>
  );
}
