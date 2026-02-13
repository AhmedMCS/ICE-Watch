import { useState } from 'react';
import { IceReport } from '../types';
import { AlertTriangle, MapPin, User } from 'lucide-react';

interface IceReportFormProps {
  onSubmit: (report: Omit<IceReport, 'id' | 'timestamp'>) => void;
}

export function IceReportForm({ onSubmit }: IceReportFormProps) {
  const [location, setLocation] = useState('');
  const [severity, setSeverity] = useState<'low' | 'medium' | 'high'>('medium');
  const [description, setDescription] = useState('');
  const [reportedBy, setReportedBy] = useState('Anonymous');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!location.trim()) return;

    onSubmit({
      location: location.trim(),
      severity,
      description: description.trim(),
      reportedBy: reportedBy.trim() || 'Anonymous'
    });

    // Reset form
    setLocation('');
    setSeverity('medium');
    setDescription('');
    setReportedBy('Anonymous');
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {/* Location */}
      <div>
        <label htmlFor="location" className="block text-sm font-medium text-gray-700 mb-2">
          <MapPin className="w-4 h-4 inline mr-1" />
          Location *
        </label>
        <input
          type="text"
          id="location"
          value={location}
          onChange={(e) => setLocation(e.target.value)}
          placeholder="e.g., Main Quad, Library Steps"
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          required
        />
      </div>

      {/* Severity */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          <AlertTriangle className="w-4 h-4 inline mr-1" />
          Severity Level
        </label>
        <div className="grid grid-cols-3 gap-2">
          <button
            type="button"
            onClick={() => setSeverity('low')}
            className={`px-4 py-2 rounded-lg border-2 transition-all ${
              severity === 'low'
                ? 'bg-yellow-100 border-yellow-400 text-yellow-800'
                : 'bg-white border-gray-200 text-gray-600 hover:border-gray-300'
            }`}
          >
            Low
          </button>
          <button
            type="button"
            onClick={() => setSeverity('medium')}
            className={`px-4 py-2 rounded-lg border-2 transition-all ${
              severity === 'medium'
                ? 'bg-orange-100 border-orange-400 text-orange-800'
                : 'bg-white border-gray-200 text-gray-600 hover:border-gray-300'
            }`}
          >
            Medium
          </button>
          <button
            type="button"
            onClick={() => setSeverity('high')}
            className={`px-4 py-2 rounded-lg border-2 transition-all ${
              severity === 'high'
                ? 'bg-red-100 border-red-400 text-red-800'
                : 'bg-white border-gray-200 text-gray-600 hover:border-gray-300'
            }`}
          >
            High
          </button>
        </div>
      </div>

      {/* Description */}
      <div>
        <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-2">
          Description
        </label>
        <textarea
          id="description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Additional details about the ice hazard..."
          rows={3}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
        />
      </div>

      {/* Reporter Name */}
      <div>
        <label htmlFor="reportedBy" className="block text-sm font-medium text-gray-700 mb-2">
          <User className="w-4 h-4 inline mr-1" />
          Your Name (Optional)
        </label>
        <input
          type="text"
          id="reportedBy"
          value={reportedBy}
          onChange={(e) => setReportedBy(e.target.value)}
          placeholder="Anonymous"
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      {/* Submit Button */}
      <button
        type="submit"
        className="w-full bg-blue-500 hover:bg-blue-600 text-white font-medium py-3 rounded-lg transition-colors"
      >
        Submit Report
      </button>
    </form>
  );
}
