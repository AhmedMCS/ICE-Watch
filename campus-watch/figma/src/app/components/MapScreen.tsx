import { useState, useEffect, useRef, useCallback } from "react";
import L from "leaflet";
import {
  Filter,
  X,
  CheckCircle,
  Clock,
  Users,
  MapPin,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import {
  mockReports,
  getTimeAgo,
  severityColors,
  reportTypeLabels,
  type Report,
  type Severity,
} from "../data/mock-reports";

function createMarkerIcon(severity: Severity, verified: boolean) {
  const color = severityColors[severity];
  const size = severity === "high" ? 32 : 26;
  const pulseRing =
    severity === "high"
      ? `<circle cx="${size / 2}" cy="${size / 2}" r="${size / 2}" fill="${color}" opacity="0.2"><animate attributeName="r" from="${size / 2}" to="${size}" dur="1.5s" repeatCount="indefinite"/><animate attributeName="opacity" from="0.3" to="0" dur="1.5s" repeatCount="indefinite"/></circle>`
      : "";

  return L.divIcon({
    html: `
      <div style="position:relative;width:${size}px;height:${size}px;">
        <svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}">
          ${pulseRing}
          <circle cx="${size / 2}" cy="${size / 2}" r="${size / 2 - 2}" fill="${color}" stroke="rgba(0,0,0,0.3)" stroke-width="2"/>
          ${verified ? `<circle cx="${size / 2}" cy="${size / 2}" r="4" fill="white"/>` : ""}
        </svg>
      </div>
    `,
    className: "custom-marker",
    iconSize: [size, size],
    iconAnchor: [size / 2, size / 2],
  });
}

function ReportPopupCard({
  report,
  onClose,
}: {
  report: Report;
  onClose: () => void;
}) {
  const color = severityColors[report.severity];

  return (
    <motion.div
      initial={{ opacity: 0, y: 60 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 60 }}
      transition={{ type: "spring", bounce: 0.15, duration: 0.4 }}
      className="absolute bottom-4 left-4 right-4 z-[1000] bg-card rounded-2xl border border-border p-4 shadow-2xl"
    >
      <button
        onClick={onClose}
        className="absolute top-3 right-3 w-8 h-8 bg-accent rounded-full flex items-center justify-center"
      >
        <X size={16} className="text-muted-foreground" />
      </button>

      <div className="flex items-center gap-2 mb-2">
        <span
          className="text-[13px] px-2.5 py-1 rounded-full"
          style={{ backgroundColor: `${color}15`, color }}
        >
          {reportTypeLabels[report.type]}
        </span>
        <span
          className="text-[12px] px-2 py-0.5 rounded-full capitalize"
          style={{ backgroundColor: `${color}15`, color }}
        >
          {report.severity}
        </span>
        {report.verified && (
          <span className="flex items-center gap-1 text-[12px] text-emerald-400">
            <CheckCircle size={12} />
            Verified
          </span>
        )}
      </div>

      <p className="text-foreground text-[14px] mb-3">{report.description}</p>

      <div className="flex items-center gap-4 text-muted-foreground">
        <span className="flex items-center gap-1 text-[12px]">
          <MapPin size={12} />
          {report.location.neighborhood}
        </span>
        <span className="flex items-center gap-1 text-[12px]">
          <Clock size={12} />
          {getTimeAgo(report.timestamp)}
        </span>
        <span className="flex items-center gap-1 text-[12px]">
          <Users size={12} />
          {report.confirmations} confirmed
        </span>
      </div>
    </motion.div>
  );
}

export function MapScreen() {
  const [selectedReport, setSelectedReport] = useState<Report | null>(null);
  const [filterOpen, setFilterOpen] = useState(false);
  const [activeFilters, setActiveFilters] = useState<Severity[]>([
    "low",
    "medium",
    "high",
  ]);

  const mapContainerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const markersRef = useRef<L.Marker[]>([]);

  const center: [number, number] = [34.0522, -118.2437];

  const filteredReports = mockReports.filter((r) =>
    activeFilters.includes(r.severity)
  );

  const handleMarkerClick = useCallback((report: Report) => {
    setSelectedReport(report);
  }, []);

  // Initialize map once
  useEffect(() => {
    if (!mapContainerRef.current || mapRef.current) return;

    const map = L.map(mapContainerRef.current, {
      center,
      zoom: 13,
      zoomControl: false,
      attributionControl: false,
    });

    L.tileLayer(
      "https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
    ).addTo(map);

    mapRef.current = map;

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, []);

  // Update markers when filters change
  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;

    // Clear existing markers
    markersRef.current.forEach((m) => m.remove());
    markersRef.current = [];

    // Add filtered markers
    filteredReports.forEach((report) => {
      const marker = L.marker(
        [report.location.lat, report.location.lng],
        { icon: createMarkerIcon(report.severity, report.verified) }
      ).addTo(map);

      marker.on("click", () => handleMarkerClick(report));
      markersRef.current.push(marker);
    });
  }, [filteredReports, handleMarkerClick]);

  const toggleFilter = (sev: Severity) => {
    setActiveFilters((prev) =>
      prev.includes(sev) ? prev.filter((s) => s !== sev) : [...prev, sev]
    );
  };

  return (
    <div className="relative h-full w-full flex flex-col">
      {/* Map Header */}
      <div className="absolute top-0 left-0 right-0 z-[1000] px-4 pt-4 pb-2">
        <div className="flex items-center justify-between bg-card/90 backdrop-blur-lg rounded-2xl p-3 border border-border shadow-lg">
          <div className="flex items-center gap-2">
            <MapPin size={18} className="text-primary" />
            <div>
              <h2 className="text-foreground text-[15px]">Activity Map</h2>
              <p className="text-muted-foreground text-[11px]">
                {filteredReports.length} reports in area
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setFilterOpen(!filterOpen)}
              className={`w-9 h-9 rounded-xl flex items-center justify-center transition-colors ${
                filterOpen
                  ? "bg-primary text-primary-foreground"
                  : "bg-accent text-foreground"
              }`}
            >
              <Filter size={16} />
            </button>
          </div>
        </div>

        {/* Filter Dropdown */}
        <AnimatePresence>
          {filterOpen && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              className="mt-2 bg-card/95 backdrop-blur-lg rounded-2xl p-3 border border-border shadow-lg"
            >
              <p className="text-muted-foreground text-[12px] mb-2">
                Filter by severity
              </p>
              <div className="flex gap-2">
                {(["high", "medium", "low"] as Severity[]).map((sev) => {
                  const active = activeFilters.includes(sev);
                  return (
                    <button
                      key={sev}
                      onClick={() => toggleFilter(sev)}
                      className="flex-1 py-2 rounded-xl text-[13px] capitalize transition-all border"
                      style={{
                        backgroundColor: active
                          ? `${severityColors[sev]}20`
                          : "transparent",
                        borderColor: active
                          ? severityColors[sev]
                          : "rgba(255,255,255,0.08)",
                        color: active
                          ? severityColors[sev]
                          : "var(--muted-foreground)",
                      }}
                    >
                      {sev}
                    </button>
                  );
                })}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Map Container */}
      <div className="flex-1 relative">
        <div ref={mapContainerRef} style={{ height: "100%", width: "100%" }} />
      </div>

      {/* Legend */}
      <div className="absolute bottom-20 right-4 z-[1000]">
        <div className="bg-card/90 backdrop-blur-lg rounded-xl p-2.5 border border-border shadow-lg">
          {(["high", "medium", "low"] as Severity[]).map((sev) => (
            <div key={sev} className="flex items-center gap-2 py-1">
              <div
                className="w-3 h-3 rounded-full"
                style={{ backgroundColor: severityColors[sev] }}
              />
              <span className="text-[11px] text-muted-foreground capitalize">
                {sev}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Selected Report Card */}
      <AnimatePresence>
        {selectedReport && (
          <ReportPopupCard
            report={selectedReport}
            onClose={() => setSelectedReport(null)}
          />
        )}
      </AnimatePresence>
    </div>
  );
}
