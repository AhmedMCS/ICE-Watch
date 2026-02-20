import { useState } from "react";
import { useNavigate } from "react-router";
import {
  ArrowLeft,
  MapPin,
  ShieldAlert,
  Siren,
  Car,
  Building,
  Eye,
  Send,
  CheckCircle,
  AlertTriangle,
  Clock,
  Camera,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { toast } from "sonner";
import type { ReportType, Severity } from "../data/mock-reports";

const reportTypes: { type: ReportType; label: string; icon: React.ElementType; description: string }[] = [
  { type: "checkpoint", label: "Checkpoint", icon: ShieldAlert, description: "Officers stationed at a fixed location" },
  { type: "raid", label: "Raid", icon: Siren, description: "Officers entering a building or residence" },
  { type: "patrol", label: "Patrol", icon: Car, description: "Officers in vehicles patrolling an area" },
  { type: "office_visit", label: "Office Visit", icon: Building, description: "Officers visiting a workplace" },
  { type: "surveillance", label: "Surveillance", icon: Eye, description: "Officers observing or asking questions" },
];

const severityOptions: { level: Severity; label: string; color: string; description: string }[] = [
  { level: "low", label: "Low", color: "#22c55e", description: "Activity has ended or is minor" },
  { level: "medium", label: "Medium", color: "#f59e0b", description: "Active presence, limited scope" },
  { level: "high", label: "High", color: "#ef4444", description: "Active operations, large presence" },
];

export function ReportScreen() {
  const navigate = useNavigate();
  const [step, setStep] = useState(0);
  const [selectedType, setSelectedType] = useState<ReportType | null>(null);
  const [selectedSeverity, setSelectedSeverity] = useState<Severity | null>(null);
  const [description, setDescription] = useState("");
  const [location, setLocation] = useState("");
  const [submitted, setSubmitted] = useState(false);

  const canProceedStep0 = selectedType !== null;
  const canProceedStep1 = selectedSeverity !== null;
  const canSubmit = description.length > 0 && location.length > 0;

  const handleSubmit = () => {
    setSubmitted(true);
    toast.success("Report submitted successfully", {
      description: "Your report will help keep the community safe.",
    });
    setTimeout(() => {
      navigate("/");
    }, 2500);
  };

  if (submitted) {
    return (
      <div className="flex flex-col items-center justify-center h-full px-6">
        <motion.div
          initial={{ scale: 0, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ type: "spring", bounce: 0.4, duration: 0.6 }}
          className="w-20 h-20 bg-emerald-500/15 rounded-full flex items-center justify-center mb-6"
        >
          <CheckCircle size={40} className="text-emerald-400" />
        </motion.div>
        <motion.h2
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="text-foreground text-[22px] mb-2 text-center"
        >
          Report Submitted
        </motion.h2>
        <motion.p
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="text-muted-foreground text-[14px] text-center max-w-xs"
        >
          Thank you for helping keep your community safe. Your report is now
          visible to other users in the area.
        </motion.p>
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.5 }}
          className="flex items-center gap-2 mt-6 text-muted-foreground text-[13px]"
        >
          <Clock size={14} />
          Redirecting to home...
        </motion.div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center gap-3 px-5 pt-6 pb-4">
        <button
          onClick={() => (step > 0 ? setStep(step - 1) : navigate(-1))}
          className="w-9 h-9 bg-card border border-border rounded-xl flex items-center justify-center"
        >
          <ArrowLeft size={18} className="text-foreground" />
        </button>
        <div className="flex-1">
          <h1 className="text-foreground text-[20px]">Report Activity</h1>
          <p className="text-muted-foreground text-[13px]">
            Step {step + 1} of 3
          </p>
        </div>
        <div className="flex items-center gap-1">
          <AlertTriangle size={16} className="text-primary" />
          <span className="text-primary text-[13px]">Anonymous</span>
        </div>
      </div>

      {/* Progress Bar */}
      <div className="px-5 mb-6">
        <div className="h-1 bg-accent rounded-full overflow-hidden">
          <motion.div
            className="h-full bg-primary rounded-full"
            initial={{ width: "0%" }}
            animate={{ width: `${((step + 1) / 3) * 100}%` }}
            transition={{ type: "spring", bounce: 0, duration: 0.4 }}
          />
        </div>
      </div>

      {/* Step Content */}
      <div className="flex-1 px-5 overflow-y-auto">
        <AnimatePresence mode="wait">
          {/* Step 0: Type Selection */}
          {step === 0 && (
            <motion.div
              key="step0"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
            >
              <h2 className="text-foreground text-[18px] mb-1">
                What type of activity?
              </h2>
              <p className="text-muted-foreground text-[14px] mb-5">
                Select the type of ICE activity you observed
              </p>
              <div className="flex flex-col gap-3">
                {reportTypes.map((rt) => {
                  const isSelected = selectedType === rt.type;
                  const Icon = rt.icon;
                  return (
                    <button
                      key={rt.type}
                      onClick={() => setSelectedType(rt.type)}
                      className={`flex items-center gap-3 p-4 rounded-2xl border transition-all text-left ${
                        isSelected
                          ? "bg-primary/10 border-primary"
                          : "bg-card border-border"
                      }`}
                    >
                      <div
                        className={`w-10 h-10 rounded-xl flex items-center justify-center ${
                          isSelected ? "bg-primary/20" : "bg-accent"
                        }`}
                      >
                        <Icon
                          size={20}
                          className={isSelected ? "text-primary" : "text-muted-foreground"}
                        />
                      </div>
                      <div className="flex-1">
                        <p
                          className={`text-[14px] ${
                            isSelected ? "text-primary" : "text-foreground"
                          }`}
                        >
                          {rt.label}
                        </p>
                        <p className="text-muted-foreground text-[12px]">
                          {rt.description}
                        </p>
                      </div>
                      {isSelected && (
                        <motion.div
                          initial={{ scale: 0 }}
                          animate={{ scale: 1 }}
                          className="w-6 h-6 bg-primary rounded-full flex items-center justify-center"
                        >
                          <CheckCircle size={14} className="text-primary-foreground" />
                        </motion.div>
                      )}
                    </button>
                  );
                })}
              </div>
            </motion.div>
          )}

          {/* Step 1: Severity */}
          {step === 1 && (
            <motion.div
              key="step1"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
            >
              <h2 className="text-foreground text-[18px] mb-1">
                Severity level?
              </h2>
              <p className="text-muted-foreground text-[14px] mb-5">
                How urgent is this activity?
              </p>
              <div className="flex flex-col gap-3">
                {severityOptions.map((sev) => {
                  const isSelected = selectedSeverity === sev.level;
                  return (
                    <button
                      key={sev.level}
                      onClick={() => setSelectedSeverity(sev.level)}
                      className="flex items-center gap-3 p-4 rounded-2xl border transition-all text-left"
                      style={{
                        backgroundColor: isSelected
                          ? `${sev.color}10`
                          : "var(--card)",
                        borderColor: isSelected
                          ? sev.color
                          : "var(--border)",
                      }}
                    >
                      <div
                        className="w-10 h-10 rounded-xl flex items-center justify-center"
                        style={{
                          backgroundColor: isSelected
                            ? `${sev.color}25`
                            : "var(--accent)",
                        }}
                      >
                        <div
                          className="w-4 h-4 rounded-full"
                          style={{ backgroundColor: sev.color }}
                        />
                      </div>
                      <div className="flex-1">
                        <p
                          className="text-[14px]"
                          style={{
                            color: isSelected ? sev.color : "var(--foreground)",
                          }}
                        >
                          {sev.label}
                        </p>
                        <p className="text-muted-foreground text-[12px]">
                          {sev.description}
                        </p>
                      </div>
                      {isSelected && (
                        <motion.div
                          initial={{ scale: 0 }}
                          animate={{ scale: 1 }}
                          className="w-6 h-6 rounded-full flex items-center justify-center"
                          style={{ backgroundColor: sev.color }}
                        >
                          <CheckCircle size={14} className="text-white" />
                        </motion.div>
                      )}
                    </button>
                  );
                })}
              </div>
            </motion.div>
          )}

          {/* Step 2: Details */}
          {step === 2 && (
            <motion.div
              key="step2"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
            >
              <h2 className="text-foreground text-[18px] mb-1">
                Add details
              </h2>
              <p className="text-muted-foreground text-[14px] mb-5">
                Provide location and description
              </p>

              {/* Location Input */}
              <div className="mb-4">
                <label className="text-foreground text-[14px] mb-2 block">
                  Location
                </label>
                <div className="relative">
                  <MapPin
                    size={18}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                  />
                  <input
                    type="text"
                    placeholder="Enter street address or intersection"
                    value={location}
                    onChange={(e) => setLocation(e.target.value)}
                    className="w-full bg-card border border-border rounded-xl py-3 pl-10 pr-4 text-foreground text-[14px] placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary/30 transition-colors"
                  />
                </div>
              </div>

              {/* Description */}
              <div className="mb-4">
                <label className="text-foreground text-[14px] mb-2 block">
                  Description
                </label>
                <textarea
                  placeholder="Describe what you observed (number of officers, vehicles, activity type, etc.)"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={4}
                  className="w-full bg-card border border-border rounded-xl py-3 px-4 text-foreground text-[14px] placeholder:text-muted-foreground/50 focus:border-primary focus:ring-1 focus:ring-primary/30 transition-colors resize-none"
                />
                <p className="text-muted-foreground text-[12px] mt-1 text-right">
                  {description.length}/500
                </p>
              </div>

              {/* Photo Upload Area */}
              <div className="mb-4">
                <label className="text-foreground text-[14px] mb-2 block">
                  Photo (optional)
                </label>
                <div className="border-2 border-dashed border-border rounded-xl p-6 flex flex-col items-center gap-2 cursor-pointer hover:border-muted-foreground/30 transition-colors">
                  <Camera size={24} className="text-muted-foreground" />
                  <p className="text-muted-foreground text-[13px]">
                    Tap to add photo
                  </p>
                  <p className="text-muted-foreground/50 text-[11px]">
                    Photos are stripped of metadata for your safety
                  </p>
                </div>
              </div>

              {/* Safety Notice */}
              <div className="bg-primary/5 border border-primary/15 rounded-xl p-3 flex items-start gap-2">
                <AlertTriangle size={16} className="text-primary mt-0.5 flex-shrink-0" />
                <p className="text-muted-foreground text-[12px]">
                  Your report is completely anonymous. No personal information
                  or location data is collected from your device.
                </p>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Bottom Action */}
      <div className="px-5 py-4 flex-shrink-0">
        {step < 2 ? (
          <button
            onClick={() => setStep(step + 1)}
            disabled={step === 0 ? !canProceedStep0 : !canProceedStep1}
            className="w-full bg-primary text-primary-foreground rounded-2xl py-3.5 flex items-center justify-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed active:scale-[0.97] transition-all"
          >
            Continue
          </button>
        ) : (
          <button
            onClick={handleSubmit}
            disabled={!canSubmit}
            className="w-full bg-primary text-primary-foreground rounded-2xl py-3.5 flex items-center justify-center gap-2 disabled:opacity-40 disabled:cursor-not-allowed active:scale-[0.97] transition-all"
          >
            <Send size={18} />
            Submit Report
          </button>
        )}
      </div>
    </div>
  );
}
