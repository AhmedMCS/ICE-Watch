import { IceReport } from '../types';
import { IceReportCard } from './IceReportCard';

interface IceReportListProps {
  reports: IceReport[];
  onDelete: (id: string) => void;
}

export function IceReportList({ reports, onDelete }: IceReportListProps) {
  if (reports.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-green-100 rounded-full mb-4">
          <span className="text-3xl">✓</span>
        </div>
        <p className="text-gray-500">No active ice reports. Campus looks safe!</p>
      </div>
    );
  }

  return (
    <div className="space-y-3 max-h-[600px] overflow-y-auto pr-2">
      {reports.map((report) => (
        <IceReportCard key={report.id} report={report} onDelete={onDelete} />
      ))}
    </div>
  );
}
