import { useState } from 'react';
import { ActivityReportForm } from './components/ActivityReportForm';
import { ActivityReportList } from './components/ActivityReportList';
import { ActivityReport } from './types';
import { Shield, AlertTriangle, TrendingUp, MapPin, Activity } from 'lucide-react';
import { motion } from 'motion/react';

function App() {
  const [reports, setReports] = useState<ActivityReport[]>([
    {
      id: '1',
      location: 'Near Campus Parking Lot D',
      activityType: 'vehicle',
      description: 'Unmarked vehicle spotted, left after 15 minutes',
      timestamp: new Date(Date.now() - 1000 * 60 * 45).toISOString(),
      verified: false
    },
    {
      id: '2',
      location: 'Student Housing Complex',
      activityType: 'agents',
      description: 'Two individuals in plain clothes asking questions',
      timestamp: new Date(Date.now() - 1000 * 60 * 180).toISOString(),
      verified: true
    }
  ]);

  const [showSuccess, setShowSuccess] = useState(false);

  const addReport = (report: Omit<ActivityReport, 'id' | 'timestamp'>) => {
    const newReport: ActivityReport = {
      ...report,
      id: Date.now().toString(),
      timestamp: new Date().toISOString()
    };
    setReports([newReport, ...reports]);
    setShowSuccess(true);
    setTimeout(() => setShowSuccess(false), 3000);
  };

  const deleteReport = (id: string) => {
    setReports(reports.filter(report => report.id !== id));
  };

  const toggleVerified = (id: string) => {
    setReports(reports.map(report => 
      report.id === id ? { ...report, verified: !report.verified } : report
    ));
  };

  const verifiedCount = reports.filter(r => r.verified).length;
  const recentCount = reports.filter(r => {
    const hoursSince = (Date.now() - new Date(r.timestamp).getTime()) / (1000 * 60 * 60);
    return hoursSince < 24;
  }).length;

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100">
      {/* Header */}
      <motion.header 
        className="bg-slate-900 text-white shadow-lg"
        initial={{ y: -100 }}
        animate={{ y: 0 }}
        transition={{ type: 'spring', stiffness: 100 }}
      >
        <div className="max-w-6xl mx-auto px-4 py-6">
          <div className="flex items-center gap-3">
            <motion.div 
              className="bg-red-600 p-2 rounded-lg"
              whileHover={{ scale: 1.1, rotate: 5 }}
              whileTap={{ scale: 0.9 }}
            >
              <Shield className="w-6 h-6 text-white" />
            </motion.div>
            <div>
              <h1 className="text-2xl font-bold">Campus ICE Activity Tracker</h1>
              <p className="text-sm text-slate-300">Community awareness and safety reporting</p>
            </div>
          </div>
        </div>
      </motion.header>

      {/* Alert Banner */}
      <motion.div 
        className="bg-red-50 border-b border-red-200"
        initial={{ opacity: 0, height: 0 }}
        animate={{ opacity: 1, height: 'auto' }}
        transition={{ delay: 0.2 }}
      >
        <div className="max-w-6xl mx-auto px-4 py-3">
          <div className="flex items-start gap-3">
            <motion.div
              animate={{ rotate: [0, -10, 10, -10, 0] }}
              transition={{ duration: 0.5, repeat: Infinity, repeatDelay: 3 }}
            >
              <AlertTriangle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            </motion.div>
            <div className="text-sm text-red-800">
              <strong>Important:</strong> If you witness ICE activity, prioritize your safety first. Do not interfere with enforcement actions. Contact campus resources or legal support organizations if needed.
            </div>
          </div>
        </div>
      </motion.div>

      {/* Success Message */}
      {showSuccess && (
        <motion.div
          className="fixed top-24 right-4 bg-green-600 text-white px-6 py-3 rounded-lg shadow-lg z-50"
          initial={{ x: 400, opacity: 0 }}
          animate={{ x: 0, opacity: 1 }}
          exit={{ x: 400, opacity: 0 }}
          transition={{ type: 'spring', stiffness: 200, damping: 20 }}
        >
          <div className="flex items-center gap-2">
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ delay: 0.2, type: 'spring', stiffness: 300 }}
            >
              ✓
            </motion.div>
            Report submitted successfully!
          </div>
        </motion.div>
      )}

      {/* Stats Bar */}
      <motion.div 
        className="bg-white border-b border-slate-200 shadow-sm"
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.3 }}
      >
        <div className="max-w-6xl mx-auto px-4 py-4">
          <div className="grid grid-cols-3 gap-4">
            <motion.div 
              className="flex items-center gap-3 p-3 bg-slate-50 rounded-lg cursor-pointer"
              whileHover={{ scale: 1.05, backgroundColor: 'rgb(241, 245, 249)' }}
              whileTap={{ scale: 0.95 }}
            >
              <div className="bg-blue-500 p-2 rounded-lg">
                <Activity className="w-5 h-5 text-white" />
              </div>
              <div>
                <div className="text-2xl font-bold text-slate-900">{reports.length}</div>
                <div className="text-xs text-slate-600">Total Reports</div>
              </div>
            </motion.div>

            <motion.div 
              className="flex items-center gap-3 p-3 bg-slate-50 rounded-lg cursor-pointer"
              whileHover={{ scale: 1.05, backgroundColor: 'rgb(241, 245, 249)' }}
              whileTap={{ scale: 0.95 }}
            >
              <div className="bg-green-500 p-2 rounded-lg">
                <TrendingUp className="w-5 h-5 text-white" />
              </div>
              <div>
                <div className="text-2xl font-bold text-slate-900">{recentCount}</div>
                <div className="text-xs text-slate-600">Last 24 Hours</div>
              </div>
            </motion.div>

            <motion.div 
              className="flex items-center gap-3 p-3 bg-slate-50 rounded-lg cursor-pointer"
              whileHover={{ scale: 1.05, backgroundColor: 'rgb(241, 245, 249)' }}
              whileTap={{ scale: 0.95 }}
            >
              <div className="bg-purple-500 p-2 rounded-lg">
                <MapPin className="w-5 h-5 text-white" />
              </div>
              <div>
                <div className="text-2xl font-bold text-slate-900">{verifiedCount}</div>
                <div className="text-xs text-slate-600">Verified</div>
              </div>
            </motion.div>
          </div>
        </div>
      </motion.div>

      {/* Main Content */}
      <main className="max-w-6xl mx-auto px-4 py-8">
        <div className="grid gap-6 lg:grid-cols-2">
          {/* Report Form */}
          <motion.div 
            className="bg-white rounded-xl shadow-sm p-6 border border-slate-200 h-fit"
            initial={{ opacity: 0, x: -50 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.4 }}
            whileHover={{ boxShadow: '0 10px 25px rgba(0,0,0,0.1)' }}
          >
            <h2 className="text-xl font-semibold text-gray-900 mb-4">Report Activity</h2>
            <ActivityReportForm onSubmit={addReport} />
          </motion.div>

          {/* Reports List */}
          <motion.div 
            className="bg-white rounded-xl shadow-sm p-6 border border-slate-200"
            initial={{ opacity: 0, x: 50 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.5 }}
            whileHover={{ boxShadow: '0 10px 25px rgba(0,0,0,0.1)' }}
          >
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900">Recent Reports</h2>
              <motion.span 
                className="bg-slate-100 text-slate-700 px-3 py-1 rounded-full text-sm font-medium"
                key={reports.length}
                initial={{ scale: 1.3 }}
                animate={{ scale: 1 }}
                transition={{ type: 'spring', stiffness: 300 }}
              >
                {reports.length} {reports.length === 1 ? 'report' : 'reports'}
              </motion.span>
            </div>
            <ActivityReportList 
              reports={reports} 
              onDelete={deleteReport}
              onToggleVerified={toggleVerified}
            />
          </motion.div>
        </div>

        {/* Resources Section */}
        <motion.div 
          className="mt-8 grid gap-4 md:grid-cols-3"
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.6 }}
        >
          <motion.div 
            className="bg-blue-50 border border-blue-200 rounded-xl p-4 cursor-pointer"
            whileHover={{ scale: 1.05, y: -5 }}
            whileTap={{ scale: 0.98 }}
            transition={{ type: 'spring', stiffness: 300 }}
          >
            <h3 className="font-semibold text-blue-900 mb-2">Know Your Rights</h3>
            <p className="text-sm text-blue-800">
              You have the right to remain silent and the right to speak to a lawyer. You do not have to open your door or answer questions.
            </p>
          </motion.div>
          <motion.div 
            className="bg-purple-50 border border-purple-200 rounded-xl p-4 cursor-pointer"
            whileHover={{ scale: 1.05, y: -5 }}
            whileTap={{ scale: 0.98 }}
            transition={{ type: 'spring', stiffness: 300 }}
          >
            <h3 className="font-semibold text-purple-900 mb-2">Campus Resources</h3>
            <p className="text-sm text-purple-800">
              Contact campus legal aid, student services, or designated support offices for assistance and guidance.
            </p>
          </motion.div>
          <motion.div 
            className="bg-green-50 border border-green-200 rounded-xl p-4 cursor-pointer"
            whileHover={{ scale: 1.05, y: -5 }}
            whileTap={{ scale: 0.98 }}
            transition={{ type: 'spring', stiffness: 300 }}
          >
            <h3 className="font-semibold text-green-900 mb-2">Community Support</h3>
            <p className="text-sm text-green-800">
              Share information responsibly and support community members. Report suspicious activity to help keep everyone informed.
            </p>
          </motion.div>
        </motion.div>
      </main>
    </div>
  );
}

export default App;