import { ActivityReport } from '../types';
import { ActivityReportCard } from './ActivityReportCard';
import { motion, AnimatePresence } from 'motion/react';

interface ActivityReportListProps {
  reports: ActivityReport[];
  onDelete: (id: string) => void;
  onToggleVerified: (id: string) => void;
}

export function ActivityReportList({ reports, onDelete, onToggleVerified }: ActivityReportListProps) {
  if (reports.length === 0) {
    return (
      <motion.div 
        className="text-center py-12"
        initial={{ opacity: 0, scale: 0.8 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ type: 'spring', stiffness: 200 }}
      >
        <motion.div 
          className="inline-flex items-center justify-center w-16 h-16 bg-green-100 rounded-full mb-4"
          animate={{ scale: [1, 1.1, 1] }}
          transition={{ duration: 2, repeat: Infinity }}
        >
          <span className="text-3xl">✓</span>
        </motion.div>
        <p className="text-gray-500">No recent activity reported</p>
      </motion.div>
    );
  }

  return (
    <div className="space-y-3 max-h-[700px] overflow-y-auto pr-2">
      <AnimatePresence mode="popLayout">
        {reports.map((report, index) => (
          <motion.div
            key={report.id}
            layout
            initial={{ opacity: 0, y: -20, scale: 0.9 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, x: -100, scale: 0.8 }}
            transition={{ 
              type: 'spring',
              stiffness: 300,
              damping: 25,
              delay: index * 0.05
            }}
          >
            <ActivityReportCard 
              report={report} 
              onDelete={onDelete}
              onToggleVerified={onToggleVerified}
            />
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}