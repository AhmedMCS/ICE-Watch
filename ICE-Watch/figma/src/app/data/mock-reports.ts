export type ReportType = "checkpoint" | "raid" | "patrol" | "office_visit" | "surveillance";
export type Severity = "low" | "medium" | "high";

export interface Report {
  id: string;
  type: ReportType;
  location: {
    lat: number;
    lng: number;
    address: string;
    neighborhood: string;
  };
  description: string;
  timestamp: Date;
  severity: Severity;
  verified: boolean;
  confirmations: number;
}

export const reportTypeLabels: Record<ReportType, string> = {
  checkpoint: "Checkpoint",
  raid: "Raid",
  patrol: "Patrol Vehicle",
  office_visit: "Office Visit",
  surveillance: "Surveillance",
};

export const reportTypeIcons: Record<ReportType, string> = {
  checkpoint: "shield-alert",
  raid: "siren",
  patrol: "car",
  office_visit: "building",
  surveillance: "eye",
};

export const severityColors: Record<Severity, string> = {
  low: "#22c55e",
  medium: "#f59e0b",
  high: "#ef4444",
};

export const mockReports: Report[] = [
  {
    id: "1",
    type: "checkpoint",
    location: {
      lat: 34.0522,
      lng: -118.2437,
      address: "S Broadway & W 7th St",
      neighborhood: "Downtown LA",
    },
    description: "ICE checkpoint set up near intersection. Multiple officers and vehicles present. Stopping pedestrians.",
    timestamp: new Date(Date.now() - 15 * 60 * 1000),
    severity: "high",
    verified: true,
    confirmations: 24,
  },
  {
    id: "2",
    type: "patrol",
    location: {
      lat: 34.0695,
      lng: -118.2453,
      address: "Echo Park Ave & Sunset Blvd",
      neighborhood: "Echo Park",
    },
    description: "Unmarked white van with federal plates circling the neighborhood slowly.",
    timestamp: new Date(Date.now() - 45 * 60 * 1000),
    severity: "medium",
    verified: true,
    confirmations: 12,
  },
  {
    id: "3",
    type: "raid",
    location: {
      lat: 34.0407,
      lng: -118.2468,
      address: "E Olympic Blvd & S Central Ave",
      neighborhood: "Central-Alameda",
    },
    description: "Officers entering residential building with tactical gear. Area blocked off.",
    timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000),
    severity: "high",
    verified: true,
    confirmations: 38,
  },
  {
    id: "4",
    type: "surveillance",
    location: {
      lat: 34.0561,
      lng: -118.2614,
      address: "W 3rd St & S Vermont Ave",
      neighborhood: "Koreatown",
    },
    description: "Two individuals in plain clothes asking questions at local businesses. Appear to have badges.",
    timestamp: new Date(Date.now() - 3 * 60 * 60 * 1000),
    severity: "medium",
    verified: false,
    confirmations: 6,
  },
  {
    id: "5",
    type: "office_visit",
    location: {
      lat: 34.0486,
      lng: -118.2582,
      address: "W Wilshire Blvd & S Hoover St",
      neighborhood: "Westlake",
    },
    description: "Officers seen visiting a workplace, asking management for employee records.",
    timestamp: new Date(Date.now() - 5 * 60 * 60 * 1000),
    severity: "low",
    verified: false,
    confirmations: 3,
  },
  {
    id: "6",
    type: "patrol",
    location: {
      lat: 34.0325,
      lng: -118.2639,
      address: "S Figueroa St & W Adams Blvd",
      neighborhood: "University Park",
    },
    description: "Marked ICE vehicle parked near school zone during drop-off hours.",
    timestamp: new Date(Date.now() - 6 * 60 * 60 * 1000),
    severity: "high",
    verified: true,
    confirmations: 45,
  },
  {
    id: "7",
    type: "checkpoint",
    location: {
      lat: 34.0782,
      lng: -118.2606,
      address: "N Alvarado St & Beverly Blvd",
      neighborhood: "Rampart Village",
    },
    description: "Mobile checkpoint near bus stop. Officers checking IDs of transit riders.",
    timestamp: new Date(Date.now() - 8 * 60 * 60 * 1000),
    severity: "high",
    verified: true,
    confirmations: 31,
  },
];

export function getTimeAgo(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}
