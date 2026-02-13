import { useState } from 'react';
import { ActivityReport } from '../types';
import { MapPin, FileText, Activity } from 'lucide-react';
import { motion } from 'motion/react';

interface ActivityReportFormProps {
  onSubmit: (report: Omit<ActivityReport, 'id' | 'timestamp'>) => void;
}

export function ActivityReportForm({ onSubmit }: ActivityReportFormProps) {
  const [location, setLocation] = useState('');
  const [activityType, setActivityType] = useState<ActivityReport['activityType']>('other');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!location.trim() || !description.trim()) return;

    setIsSubmitting(true);

    // Simulate submission delay for animation
    await new Promise(resolve => setTimeout(resolve, 500));

    onSubmit({
      location: location.trim(),
      activityType,
      description: description.trim(),
      verified: false
    });

    // Reset form
    setLocation('');
    setActivityType('other');
    setDescription('');
    setIsSubmitting(false);
  };

  const activityTypes: { value: ActivityReport['activityType']; label: string }[] = [
    { value: 'agents', label: 'Agents Present' },
    { value: 'vehicle', label: 'Suspicious Vehicle' },
    { value: 'raid', label: 'Raid/Enforcement Action' },
    { value: 'questioning', label: 'Questioning Individuals' },
    { value: 'other', label: 'Other Activity' }
  ];

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Location */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
      >
        <label htmlFor="location" className="block text-sm font-medium text-gray-700 mb-2">
          <MapPin className="w-4 h-4 inline mr-1" />
          Location *
        </label>
        <motion.input
          type="text"
          id="location"
          value={location}
          onChange={(e) => setLocation(e.target.value)}
          placeholder="e.g., Near Main Library, Parking Lot B"
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent transition-all"
          required
          whileFocus={{ scale: 1.02 }}
        />
      </motion.div>

      {/* Activity Type */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
      >
        <label className="block text-sm font-medium text-gray-700 mb-2">
          <Activity className="w-4 h-4 inline mr-1" />
          Activity Type *
        </label>
        <div className="grid grid-cols-2 gap-2">
          {activityTypes.map((type, index) => (
            <motion.button
              key={type.value}
              type="button"
              onClick={() => setActivityType(type.value)}
              className={`px-3 py-2 rounded-lg text-sm font-medium transition-all ${
                activityType === type.value
                  ? 'bg-slate-800 text-white shadow-md'
                  : 'bg-slate-100 text-slate-700 hover:bg-slate-200'
              }`}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: 0.2 + index * 0.05 }}
            >
              {type.label}
            </motion.button>
          ))}
        </div>
      </motion.div>

      {/* Description */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.3 }}
      >
        <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-2">
          <FileText className="w-4 h-4 inline mr-1" />
          Description *
        </label>
        <motion.textarea
          id="description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Provide details: What did you observe? How many people/vehicles? What time? Any other relevant information..."
          rows={4}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-slate-500 focus:border-transparent resize-none transition-all"
          required
          whileFocus={{ scale: 1.02 }}
        />
      </motion.div>

      {/* Privacy Notice */}
      <motion.div 
        className="bg-amber-50 border border-amber-200 rounded-lg p-3"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4 }}
        whileHover={{ scale: 1.02 }}
      >
        <p className="text-xs text-amber-800">
          <strong>Privacy Note:</strong> Reports are anonymous. Only share what you're comfortable sharing. Be as specific as possible without compromising safety.
        </p>
      </motion.div>

      {/* Submit Button */}
      <motion.button
        type="submit"
        className="w-full bg-slate-800 hover:bg-slate-900 text-white font-medium py-3 rounded-lg transition-colors relative overflow-hidden"
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.5 }}
        disabled={isSubmitting}
      >
        {isSubmitting ? (
          <motion.div
            className="flex items-center justify-center gap-2"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
          >
            <motion.div
              className="w-5 h-5 border-2 border-white border-t-transparent rounded-full"
              animate={{ rotate: 360 }}
              transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
            />
            Submitting...
          </motion.div>
        ) : (
          'Submit Report'
        )}
      </motion.button>
    </form>
  );
}