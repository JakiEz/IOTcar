import { ChevronDown } from "lucide-react";
import { packStats } from "@/data/mockTruck";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

// Charge → color, same bands as the battery panel.
function chargeColor(pct) {
  if (pct >= 60) return "var(--ok)";
  if (pct >= 30) return "var(--warn)";
  return "var(--bad)";
}

function Dot({ online }) {
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full",
        online ? "bg-ok shadow-[0_0_6px_var(--ok)]" : "bg-muted-foreground"
      )}
    />
  );
}

// Dropdown truck picker in the top bar. Pick a truck to make it the selected one.
export default function TruckSelector({ fleet, selectedId, onSelect }) {
  const selected = fleet.find((t) => t.id === selectedId) ?? fleet[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          className="gap-2 rounded-full border-primary bg-primary/10 font-semibold text-primary hover:bg-primary/20 hover:text-primary"
        >
          <Dot online={selected.online} />
          {selected.id}
          <ChevronDown className="size-3.5 opacity-80" />
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel>Fleet</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {fleet.map((t) => {
          const { totalPct } = packStats(t);
          return (
            <DropdownMenuItem
              key={t.id}
              onSelect={() => onSelect(t.id)}
              className={cn("gap-2.5 py-2", t.id === selectedId && "bg-accent")}
            >
              <Dot online={t.online} />
              <div className="flex flex-col leading-tight">
                <strong className="text-[13px] font-semibold">{t.name}</strong>
                <span className="text-xs text-muted-foreground">{t.id}</span>
              </div>
              <span
                className="ml-auto text-sm font-bold"
                style={{ color: t.online ? chargeColor(totalPct) : "var(--muted-foreground)" }}
              >
                {t.online ? `${totalPct}%` : "—"}
              </span>
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
