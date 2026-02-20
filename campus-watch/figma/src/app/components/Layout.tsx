import { Outlet, useLocation, useNavigate } from "react-router";
import { Home, Map, AlertTriangle, Bell } from "lucide-react";
import { motion } from "motion/react";

const navItems = [
  { path: "/", icon: Home, label: "Home" },
  { path: "/map", icon: Map, label: "Map" },
  { path: "/report", icon: AlertTriangle, label: "Report" },
  { path: "/alerts", icon: Bell, label: "Alerts" },
];

export function Layout() {
  const location = useLocation();
  const navigate = useNavigate();

  return (
    <div className="flex flex-col h-full w-full bg-background font-['Inter',sans-serif] max-w-md mx-auto relative overflow-hidden">
      {/* Main Content */}
      <div className="flex-1 overflow-y-auto overflow-x-hidden">
        <Outlet />
      </div>

      {/* Bottom Navigation */}
      <nav className="flex-shrink-0 bg-[#0f0f1a] border-t border-border px-2 pb-[env(safe-area-inset-bottom)]">
        <div className="flex items-center justify-around h-16">
          {navItems.map((item) => {
            const isActive = location.pathname === item.path;
            const Icon = item.icon;
            return (
              <button
                key={item.path}
                onClick={() => navigate(item.path)}
                className="relative flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-colors"
              >
                {isActive && (
                  <motion.div
                    layoutId="activeTab"
                    className="absolute inset-0 bg-primary/10 rounded-xl"
                    transition={{ type: "spring", bounce: 0.2, duration: 0.4 }}
                  />
                )}
                <Icon
                  size={20}
                  className={`relative z-10 transition-colors ${
                    isActive ? "text-primary" : "text-muted-foreground"
                  }`}
                />
                <span
                  className={`relative z-10 text-[11px] transition-colors ${
                    isActive ? "text-primary" : "text-muted-foreground"
                  }`}
                >
                  {item.label}
                </span>
              </button>
            );
          })}
        </div>
      </nav>
    </div>
  );
}
