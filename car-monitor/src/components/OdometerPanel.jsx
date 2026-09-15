import Sparkline from "@/components/Sparkline";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

function Stat({ label, value, unit }) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border bg-secondary/50 px-3 py-2.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <strong className="text-lg font-semibold">
        {value}
        {unit && <em className="ml-0.5 text-xs not-italic text-muted-foreground">{unit}</em>}
      </strong>
    </div>
  );
}

export default function OdometerPanel({ truck, className }) {
  return (
    <Card className={cn("flex flex-col gap-0 overflow-hidden p-4", className)}>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-[15px] font-semibold">Drivetrain &amp; Trip</h2>
        <span className="text-xs text-muted-foreground">battery % · last 30 samples</span>
      </div>

      <Sparkline data={truck.history} />

      <div className="mt-3.5 grid grid-cols-3 gap-2.5">
        <Stat label="Odometer" value={truck.odometer.totalKm.toLocaleString()} unit="km" />
        <Stat label="Trip" value={truck.odometer.tripKm.toFixed(1)} unit="km" />
        <Stat label="Speed" value={truck.speedKph} unit="km/h" />
        <Stat label="Motor temp" value={truck.energy.motorTempC} unit="°C" />
        <Stat label="Consumption" value={truck.energy.consumptionKwhPer100km} unit="kWh/100km" />
        <Stat label="Range left" value={truck.energy.rangeKm} unit="km" />
      </div>
    </Card>
  );
}
