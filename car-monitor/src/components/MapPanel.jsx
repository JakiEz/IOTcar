import { useEffect } from "react";
import { MapContainer, TileLayer, Marker, Popup, Polyline, useMap } from "react-leaflet";
import { divIcon } from "leaflet";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

// A divIcon avoids Leaflet's classic broken-default-marker problem in bundlers
// (no image path to resolve) and lets us style the marker with plain HTML/CSS.
const truckIcon = divIcon({
  className: "text-2xl leading-[34px] text-center drop-shadow-[0_2px_4px_rgba(0,0,0,0.6)]",
  html: "🚛",
  iconSize: [34, 34],
  iconAnchor: [17, 17],
});

// Keeps the map centered on the truck as it moves.
function Recenter({ position }) {
  const map = useMap();
  map.panTo(position, { animate: true, duration: 0.5 });
  return null;
}

// Leaflet measures its container on init; inside a flex/grid panel the final
// size isn't known yet, so tiles only fill part of the box. invalidateSize()
// after layout (and on window resize) forces a re-measure so tiles fill fully.
function FitToContainer() {
  const map = useMap();
  useEffect(() => {
    const fix = () => map.invalidateSize();
    const t = setTimeout(fix, 0);
    window.addEventListener("resize", fix);
    return () => {
      clearTimeout(t);
      window.removeEventListener("resize", fix);
    };
  }, [map]);
  return null;
}

export default function MapPanel({ truck, className }) {
  return (
    <Card className={cn("flex flex-col gap-0 overflow-hidden p-4", className)}>
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h2 className="text-[15px] font-semibold">{truck.name}</h2>
          <span className="text-xs text-muted-foreground">
            {truck.position[0].toFixed(4)}, {truck.position[1].toFixed(4)}
          </span>
        </div>
        <Badge
          variant={truck.online ? "default" : "secondary"}
          className={truck.online ? "bg-ok/15 text-ok" : "bg-muted text-muted-foreground"}
        >
          {truck.online ? "LIVE" : "OFFLINE"}
        </Badge>
      </div>

      <div className="relative min-h-0 flex-1 overflow-hidden rounded-xl">
        <MapContainer
          center={truck.position}
          zoom={15}
          zoomControl={false}
          style={{ height: "100%", width: "100%" }}
        >
          <TileLayer
            attribution="&copy; OpenStreetMap contributors"
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          <Polyline positions={truck.trail} pathOptions={{ color: "#a78bfa", weight: 4 }} />
          <Marker position={truck.position} icon={truckIcon}>
            <Popup>
              <strong>{truck.name}</strong>
              <br />
              {truck.speedKph} km/h
            </Popup>
          </Marker>
          <Recenter position={truck.position} />
          <FitToContainer />
        </MapContainer>

        <div className="absolute bottom-3 left-3 z-[500] flex flex-col rounded-lg border bg-card/90 px-3 py-2 backdrop-blur">
          <span className="text-xs text-muted-foreground">SPEED</span>
          <strong className="text-xl">
            {truck.speedKph}
            <em className="ml-1 text-xs not-italic text-muted-foreground">km/h</em>
          </strong>
        </div>
      </div>
    </Card>
  );
}
