import { ActivityReport } from '../types';
import { MapPin, Clock, Trash2, CheckCircle, Circle } from 'lucide-react';
import { motion } from 'motion/react';
import { useState } from 'react';

interface ActivityReportCardProps {
  report: ActivityReport;
  onDelete: (id: string) => void;
  onToggleVerified: (id: string) => void;
}

export function ActivityReportCard({ report, onDelete, onToggleVerified }: ActivityReportCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const getActivityTypeLabel = (type: string) => {
    switch (type) {
      case 'agents':
        return 'Agents Present';
      case 'vehicle':
        return 'Suspicious Vehicle';
      case 'raid':
        return 'Raid/Enforcement';
      case 'questioning':
        return 'Questioning';
      case 'other':
        return 'Other Activity';
      default:
        return type;
    }
  };

  const getActivityTypeColor = (type: string) => {
    switch (type) {
      case 'agents':
        return 'bg-red-100 text-red-800 border-red-300';
      case 'vehicle':
        return 'bg-orange-100 text-orange-800 border-orange-300';
      case 'raid':
        return 'bg-red-200 text-red-900 border-red-400';
      case 'questioning':
        return 'bg-yellow-100 text-yellow-800 border-yellow-300';
      case 'other':
        return 'bg-slate-100 text-slate-800 border-slate-300';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-300';
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
    <motion.div 
      className={`rounded-lg border-2 p-4 cursor-pointer ${report.verified ? 'bg-blue-50 border-blue-300' : 'bg-white border-slate-200'}`}
      whileHover={{ 
        scale: 1.02,
        boxShadow: '0 8px 16px rgba(0,0,0,0.1)'
      }}
      onClick={() => setIsExpanded(!isExpanded)}
      layout
    >
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="flex-1 min-w-0">
          {/* Location & Type */}
          <div className="flex items-center gap-2 mb-2 flex-wrap">
            <motion.div
              whileHover={{ rotate: 10 }}
            >
              <MapPin className="w-4 h-4 flex-shrink-0 text-slate-600" />
            </motion.div>
            <span className="font-semibold text-slate-900">{report.location}</span>
          </div>

          {/* Activity Type Badge */}
          <div className="mb-2">
            <motion.span 
              className={`inline-block text-xs font-semibold px-2 py-1 rounded border ${getActivityTypeColor(report.activityType)}`}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              {getActivityTypeLabel(report.activityType)}
            </motion.span>
          </div>

          {/* Description */}
          <motion.p 
            className="text-sm text-slate-700 mb-3"
            initial={false}
            animate={{ 
              height: isExpanded ? 'auto' : '40px',
              overflow: 'hidden'
            }}
          >
            {report.description}
          </motion.p>

          {/* Metadata */}
          <div className="flex items-center gap-4 text-xs text-slate-500">
            <motion.span 
              className="flex items-center gap-1"
              whileHover={{ scale: 1.1 }}
            >
              <Clock className="w-3 h-3" />
              {getTimeAgo(report.timestamp)}
            </motion.span>
            {report.verified && (
              <motion.span 
                className="flex items-center gap-1 text-blue-600 font-medium"
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                whileHover={{ scale: 1.1 }}
              >
                <CheckCircle className="w-3 h-3" />
                Community Verified
              </motion.span>
            )}
          </div>
        </div>

        {/* Actions */}
        <div className="flex flex-col gap-2" onClick={(e) => e.stopPropagation()}>
          <motion.button
            onClick={() => onToggleVerified(report.id)}
            className="flex-shrink-0 p-2 hover:bg-slate-100 rounded-lg transition-colors"
            aria-label={report.verified ? "Unverify report" : "Verify report"}
            title={report.verified ? "Unverify" : "Verify"}
            whileHover={{ scale: 1.2, rotate: 10 }}
            whileTap={{ scale: 0.9 }}
          >
            {report.verified ? (
              <motion.div
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ type: 'spring', stiffness: 300 }}
              >
                <CheckCircle className="w-4 h-4 text-blue-600" />
              </motion.div>
            ) : (
              <Circle className="w-4 h-4 text-slate-400" />
            )}
          </motion.button>
          <motion.button
            onClick={() => onDelete(report.id)}
            className="flex-shrink-0 p-2 hover:bg-red-100 rounded-lg transition-colors"
            aria-label="Delete report"
            whileHover={{ scale: 1.2, rotate: -10 }}
            whileTap={{ scale: 0.9 }}
          >
            <Trash2 className="w-4 h-4 text-slate-600 hover:text-red-600" />
          </motion.button>
        </div>
      </div>

      {/* Expand Indicator */}
      <motion.div 
        className="text-center text-xs text-slate-400 mt-2"
        animate={{ opacity: isExpanded ? 0 : 1 }}
      >
        Click to {isExpanded ? 'collapse' : 'expand'}
      </motion.div>
    </motion.div>
  );
}