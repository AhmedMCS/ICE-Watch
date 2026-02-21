import { useState } from "react";
import {
  Bell,
  BellOff,
  Shield,
  MapPin,
  Clock,
  ChevronRight,
  Settings,
  CheckCircle,
  AlertTriangle,
} from "lucide-react";
import { motion } from "motion/react";
import { mockReports, getTimeAgo, severityColors, reportTypeLabels } from "../data/mock-reports";

interface AlertSetting {
  id: string;
  label: string;
  description: string;
  enabled: boolean;
}

export function AlertsScreen() {
  const [activeTab, setActiveTab] = useState<"notifications" | "settings">("notifications");
  const [settings, setSettings] = useState<AlertSetting[]>([
    {
      id: "high",
      label: "High Severity Alerts",
      description: "Immediate notification for raids and large operations",
      enabled: true,
    },
    {
      id: "medium",
      label: "Medium Severity Alerts",
      description: "Notifications for patrol sightings and checkpoints",
      enabled: true,
    },
    {
      id: "low",
      label: "Low Severity Alerts",
      description: "General activity reports and surveillance",
      enabled: false,
    },
    {
      id: "nearby",
      label: "Nearby Activity Only",
      description: "Only get alerts within 2 miles of your location",
      enabled: true,
    },
    {
      id: "verified",
      label: "Verified Reports Only",
      description: "Only receive community-verified alerts",
      enabled: false,
    },
  ]);

  const toggleSetting = (id: string) => {
    setSettings((prev) =>
      prev.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s))
    );
  };

  const notifications = mockReports.slice(0, 5).map((report) => ({
    id: report.id,
    title: `${reportTypeLabels[report.type]} Reported`,
    message: `${report.location.neighborhood} - ${report.description.slice(0, 80)}...`,
    time: getTimeAgo(report.timestamp),
    severity: report.severity,
    read: Number(report.id) > 2,
  }));

  return (
    <div className="flex flex-col pb-4">
      {/* Header */}
      <div className="px-5 pt-6 pb-4">
        <div className="flex items-center gap-2 mb-1">
          <div className="w-9 h-9 bg-primary/15 rounded-xl flex items-center justify-center">
            <Bell size={20} className="text-primary" />
          </div>
          <h1 className="text-foreground text-[20px]">Alerts</h1>
        </div>
        <p className="text-muted-foreground text-[13px] mt-1">
          Manage your notifications and alert preferences
        </p>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 px-5 mb-5">
        <button
          onClick={() => setActiveTab("notifications")}
          className={`flex-1 py-2.5 rounded-xl text-[14px] transition-colors ${
            activeTab === "notifications"
              ? "bg-primary text-primary-foreground"
              : "bg-card border border-border text-muted-foreground"
          }`}
        >
          Notifications
        </button>
        <button
          onClick={() => setActiveTab("settings")}
          className={`flex-1 py-2.5 rounded-xl text-[14px] transition-colors ${
            activeTab === "settings"
              ? "bg-primary text-primary-foreground"
              : "bg-card border border-border text-muted-foreground"
          }`}
        >
          Settings
        </button>
      </div>

      {activeTab === "notifications" ? (
        <div className="px-5">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-foreground text-[15px]">Recent</h3>
            <span className="text-primary text-[13px] cursor-pointer">
              Mark all read
            </span>
          </div>

          <div className="flex flex-col gap-3">
            {notifications.map((notif, idx) => (
              <motion.div
                key={notif.id}
                initial={{ opacity: 0, y: 15 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: idx * 0.05 }}
                className={`bg-card rounded-2xl p-4 border transition-colors ${
                  !notif.read ? "border-primary/30" : "border-border"
                }`}
              >
                <div className="flex items-start gap-3">
                  <div
                    className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0"
                    style={{
                      backgroundColor: `${severityColors[notif.severity]}15`,
                    }}
                  >
                    <AlertTriangle
                      size={18}
                      style={{ color: severityColors[notif.severity] }}
                    />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="text-foreground text-[14px]">{notif.title}</p>
                      {!notif.read && (
                        <div className="w-2 h-2 bg-primary rounded-full" />
                      )}
                    </div>
                    <p className="text-muted-foreground text-[13px] mt-0.5 line-clamp-2">
                      {notif.message}
                    </p>
                    <span className="text-muted-foreground/60 text-[12px] flex items-center gap-1 mt-1.5">
                      <Clock size={11} />
                      {notif.time}
                    </span>
                  </div>
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      ) : (
        <div className="px-5">
          <div className="flex flex-col gap-3">
            {settings.map((setting, idx) => (
              <motion.div
                key={setting.id}
                initial={{ opacity: 0, y: 15 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: idx * 0.05 }}
                className="bg-card rounded-2xl p-4 border border-border"
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1 mr-4">
                    <p className="text-foreground text-[14px]">{setting.label}</p>
                    <p className="text-muted-foreground text-[12px] mt-0.5">
                      {setting.description}
                    </p>
                  </div>
                  <button
                    onClick={() => toggleSetting(setting.id)}
                    className={`w-12 h-7 rounded-full relative transition-colors ${
                      setting.enabled ? "bg-primary" : "bg-accent"
                    }`}
                  >
                    <motion.div
                      className="w-5 h-5 bg-white rounded-full absolute top-1"
                      animate={{
                        left: setting.enabled ? "calc(100% - 24px)" : "4px",
                      }}
                      transition={{ type: "spring", bounce: 0.2, duration: 0.3 }}
                    />
                  </button>
                </div>
              </motion.div>
            ))}
          </div>

          {/* Know Your Rights */}
          <div className="mt-6 bg-card rounded-2xl p-4 border border-border">
            <div className="flex items-center gap-2 mb-2">
              <Shield size={18} className="text-primary" />
              <h3 className="text-foreground text-[15px]">Know Your Rights</h3>
            </div>
            <p className="text-muted-foreground text-[13px] mb-3">
              Learn about your rights when encountering immigration enforcement officers.
            </p>
            <div className="flex flex-col gap-2">
              {[
                "You have the right to remain silent",
                "You do not have to open your door",
                "You have the right to speak with a lawyer",
                "You can refuse to sign documents",
              ].map((right, i) => (
                <div key={i} className="flex items-center gap-2">
                  <CheckCircle size={14} className="text-emerald-400 flex-shrink-0" />
                  <span className="text-foreground text-[13px]">{right}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
