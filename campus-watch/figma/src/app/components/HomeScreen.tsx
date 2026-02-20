import { useState } from "react";
import { useNavigate } from "react-router";
import {
  Shield,
  AlertTriangle,
  ChevronRight,
  Eye,
  Car,
  Building,
  ShieldAlert,
  Siren,
  CheckCircle,
  Clock,
  Users,
  MapPin,
} from "lucide-react";
import { motion } from "motion/react";
import {
  mockReports,
  getTimeAgo,
  severityColors,
  reportTypeLabels,
  type Report,
  type ReportType,
} from "../data/mock-reports";

const typeIconMap: Record<ReportType, React.ElementType> = {
  checkpoint: ShieldAlert,
  raid: Siren,
  patrol: Car,
  office_visit: Building,
  surveillance: Eye,
};

function ReportCard({ report, index }: { report: Report; index: number }) {
  const navigate = useNavigate();
  const Icon = typeIconMap[report.type];
  const severityColor = severityColors[report.severity];

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.05, duration: 0.3 }}
      onClick={() => navigate("/map")}
      className="bg-card rounded-2xl p-4 border border-border cursor-pointer active:scale-[0.98] transition-transform"
    >
      <div className="flex items-start gap-3">
        {/* Icon */}
        <div
          className="flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center"
          style={{ backgroundColor: `${severityColor}15` }}
        >
          <Icon size={20} style={{ color: severityColor }} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span
              className="text-[13px] px-2 py-0.5 rounded-full"
              style={{
                backgroundColor: `${severityColor}15`,
                color: severityColor,
              }}
            >
              {reportTypeLabels[report.type]}
            </span>
            {report.verified && (
              <span className="flex items-center gap-1 text-[12px] text-emerald-400">
                <CheckCircle size={12} />
                Verified
              </span>
            )}
          </div>

          <p className="text-foreground text-[14px] mb-2 line-clamp-2">
            {report.description}
          </p>

          <div className="flex items-center gap-3 text-muted-foreground">
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
              {report.confirmations}
            </span>
          </div>
        </div>

        <ChevronRight size={18} className="text-muted-foreground flex-shrink-0 mt-1" />
      </div>
    </motion.div>
  );
}

export function HomeScreen() {
  const navigate = useNavigate();
  const [filter, setFilter] = useState<"all" | "verified" | "high">("all");

  const filteredReports = mockReports.filter((r) => {
    if (filter === "verified") return r.verified;
    if (filter === "high") return r.severity === "high";
    return true;
  });

  const activeAlerts = mockReports.filter(
    (r) => Date.now() - r.timestamp.getTime() < 4 * 60 * 60 * 1000
  ).length;

  const verifiedCount = mockReports.filter((r) => r.verified).length;
  const highSeverityCount = mockReports.filter(
    (r) => r.severity === "high"
  ).length;

  return (
    <div className="flex flex-col pb-4">
      {/* Header */}
      <div className="px-5 pt-6 pb-4">
        <div className="flex items-center justify-between mb-1">
          <div className="flex items-center gap-2">
            <div className="w-9 h-9 bg-primary/15 rounded-xl flex items-center justify-center">
              <Shield size={20} className="text-primary" />
            </div>
            <div>
              <h1 className="text-foreground text-[20px]">ICE Watch</h1>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <div className="relative">
              <div className="w-3 h-3 bg-red-500 rounded-full animate-pulse" />
            </div>
            <span className="text-[13px] text-red-400">LIVE</span>
          </div>
        </div>
        <p className="text-muted-foreground text-[13px] mt-1">
          Community-powered safety alerts in your area
        </p>
      </div>

      {/* Alert Banner */}
      {highSeverityCount > 0 && (
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          className="mx-5 mb-4"
        >
          <div className="bg-red-500/10 border border-red-500/20 rounded-2xl p-4 flex items-center gap-3">
            <div className="w-10 h-10 bg-red-500/20 rounded-xl flex items-center justify-center flex-shrink-0">
              <AlertTriangle size={20} className="text-red-400" />
            </div>
            <div className="flex-1">
              <p className="text-red-300 text-[14px]">
                {highSeverityCount} high severity{" "}
                {highSeverityCount === 1 ? "alert" : "alerts"} nearby
              </p>
              <p className="text-red-400/60 text-[12px] mt-0.5">
                Exercise caution in reported areas
              </p>
            </div>
          </div>
        </motion.div>
      )}

      {/* Stats Row */}
      <div className="flex gap-3 px-5 mb-5">
        {[
          { label: "Active", value: activeAlerts, color: "#f59e0b" },
          { label: "Verified", value: verifiedCount, color: "#22c55e" },
          { label: "High Alert", value: highSeverityCount, color: "#ef4444" },
        ].map((stat) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex-1 bg-card rounded-2xl p-3 border border-border text-center"
          >
            <p
              className="text-[22px]"
              style={{ color: stat.color }}
            >
              {stat.value}
            </p>
            <p className="text-muted-foreground text-[12px]">{stat.label}</p>
          </motion.div>
        ))}
      </div>

      {/* Quick Actions */}
      <div className="flex gap-3 px-5 mb-5">
        <button
          onClick={() => navigate("/report")}
          className="flex-1 bg-primary text-primary-foreground rounded-2xl py-3.5 flex items-center justify-center gap-2 active:scale-[0.97] transition-transform"
        >
          <AlertTriangle size={18} />
          <span className="text-[14px]">Report Activity</span>
        </button>
        <button
          onClick={() => navigate("/map")}
          className="flex-1 bg-card border border-border text-foreground rounded-2xl py-3.5 flex items-center justify-center gap-2 active:scale-[0.97] transition-transform"
        >
          <MapPin size={18} />
          <span className="text-[14px]">View Map</span>
        </button>
      </div>

      {/* Filter Tabs */}
      <div className="flex gap-2 px-5 mb-4">
        {(["all", "verified", "high"] as const).map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={`px-4 py-2 rounded-full text-[13px] transition-colors ${
              filter === f
                ? "bg-primary text-primary-foreground"
                : "bg-card border border-border text-muted-foreground"
            }`}
          >
            {f === "all" ? "All Reports" : f === "verified" ? "Verified" : "High Alert"}
          </button>
        ))}
      </div>

      {/* Reports Feed */}
      <div className="px-5">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-foreground text-[16px]">Recent Reports</h2>
          <span className="text-muted-foreground text-[13px]">
            {filteredReports.length} reports
          </span>
        </div>

        <div className="flex flex-col gap-3">
          {filteredReports.map((report, idx) => (
            <ReportCard key={report.id} report={report} index={idx} />
          ))}
        </div>
      </div>
    </div>
  );
}
