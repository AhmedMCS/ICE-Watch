import { IceReport } from '../types';
import { MapPin, Clock, User, Trash2 } from 'lucide-react';

interface IceReportCardProps {
  report: IceReport;
  onDelete: (id: string) => void;
}

export function IceReportCard({ report, onDelete }: IceReportCardProps) {
  const getSeverityStyles = (severity: string) => {
    switch (severity) {
      case 'high':
        return 'bg-red-100 border-red-300 text-red-800';
      case 'medium':
        return 'bg-orange-100 border-orange-300 text-orange-800';
      case 'low':
        return 'bg-yellow-100 border-yellow-300 text-yellow-800';
      default:
        return 'bg-gray-100 border-gray-300 text-gray-800';
    }
  };

  const getTimeAgo = (timestamp: string) => {
    const now = new Date().getTime();
    const reportTime = new Date(timestamp).getTime();
    const diffMinutes = Math.floor((now - reportTime) / (1000 * 60));

    if (diffMinutes < 1) return 'Just now';
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    const diffHours = Math.floor(diffMinutes / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${Math.floor(diffHours / 24)}d ago`;
  };

  return (
    <div className={`rounded-lg border-2 p-4 ${getSeverityStyles(report.severity)}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          {/* Location & Severity */}
          <div className="flex items-center gap-2 mb-2">
            <MapPin className="w-4 h-4 flex-shrink-0" />
            <span className="font-semibold truncate">{report.location}</span>
            <span className="text-xs uppercase font-bold px-2 py-0.5 rounded border border-current">
              {report.severity}
            </span>
          </div>

          {/* Description */}
          {report.description && (
            <p className="text-sm mb-2 opacity-90">{report.description}</p>
          )}

          {/* Metadata */}
          <div className="flex items-center gap-4 text-xs opacity-75">
            <span className="flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {getTimeAgo(report.timestamp)}
            </span>
            <span className="flex items-center gap-1">
              <User className="w-3 h-3" />
              {report.reportedBy}
            </span>
          </div>
        </div>

        {/* Delete Button */}
        <button
          onClick={() => onDelete(report.id)}
          className="flex-shrink-0 p-2 hover:bg-white/50 rounded-lg transition-colors"
          aria-label="Delete report"
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
