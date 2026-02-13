export interface ActivityReport {
  id: string;
  location: string;
  activityType: 'agents' | 'vehicle' | 'raid' | 'questioning' | 'other';
  description: string;
  timestamp: string;
  verified: boolean;
}
