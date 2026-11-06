import { Component } from '@angular/core';
import { Employee } from '../../models/employee';
import { TimeRecord } from '../../models/time-record';

type DashboardFilter = 'present' | 'late' | 'missing';
type DashboardEmployee = Pick<Employee, 'id' | 'fullName' | 'startTime' | 'isActive'>;
type DashboardRecord = Omit<TimeRecord, 'demonstrationStatus'>;

@Component({
  imports: [],
  selector: 'app-dashboard',
  styleUrl: './dashboard.css',
  templateUrl: './dashboard.html',
})
export class Dashboard {
  // Fixed demonstration day; all active employees are provisionally expected.
  // A backend calendar will replace this assumption and the zero-tolerance rule.
  readonly consideredDate = '2026-11-06';
  readonly dateLabel = '06/11/2026';
  readonly employees: DashboardEmployee[] = [
    { id: 1, fullName: 'Ana Souza', startTime: '08:00', isActive: true },
    { id: 2, fullName: 'Bruno Lima', startTime: '09:00', isActive: false },
    { id: 3, fullName: 'Carla Mendes', startTime: '08:30', isActive: true },
    { id: 4, fullName: 'Lucas Santos', startTime: '08:00', isActive: true },
    { id: 5, fullName: 'Maria Oliveira', startTime: '12:00', isActive: true },
  ];
  // Local fixtures, not synchronized with edits in other pages.
  readonly records: DashboardRecord[] = [
    { id: 1, employeeId: 1, date: '2026-11-06', entry: '08:00' },
    { id: 6, employeeId: 4, date: '2026-11-06', entry: '08:15', exit: '17:00' },
    { id: 7, employeeId: 5, date: '2026-11-06', entry: '12:00', exit: '17:00' },
  ];
  selectedFilter: DashboardFilter | null = null;

  private minutes(time: string): number | null {
    if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(time)) return null;
    const [hours, minutes] = time.split(':').map(Number);
    return hours * 60 + minutes;
  }

  duration(record?: DashboardRecord): string {
    const entry = this.minutes(record?.entry ?? '');
    const exit = this.minutes(record?.exit ?? '');
    if (entry === null || exit === null || exit < entry) return '—';
    const total = exit - entry;
    return `${Math.floor(total / 60).toString().padStart(2, '0')}:${(total % 60).toString().padStart(2, '0')}`;
  }

  get rows() {
    return this.employees.filter(employee => employee.isActive).map(employee => {
      // One row/count per employee; use the first valid entry on the selected day.
      // Multiple shifts and daily aggregation await backend rules.
      const record = this.records.filter(record => record.employeeId === employee.id &&
        record.date === this.consideredDate && this.minutes(record.entry) !== null,
      ).sort((a, b) => a.entry.localeCompare(b.entry) || a.id - b.id)[0];
      const present = !!record;
      const planned = this.minutes(employee.startTime);
      const actual = this.minutes(record?.entry ?? '');
      const late = actual !== null && planned !== null && actual > planned;
      // Display precedence: missing > late > in progress > completed.
      const status = !present ? 'Sem marcação' : late ? 'Atraso' : !record.exit ? 'Em andamento' : 'Registrado';
      const statusClass = !present ? 'status-sem-marcacao' : late ? 'status-atraso' : !record.exit ? 'status-andamento' : 'status-registrado';
      return { employeeId: employee.id, fullName: employee.fullName, present, late,
        missing: !present, entry: record?.entry ?? '—', exit: record?.exit || '—',
        duration: this.duration(record), status, statusClass };
    });
  }

  get totals() {
    const rows = this.rows;
    return { present: rows.filter(row => row.present).length,
      late: rows.filter(row => row.late).length, missing: rows.filter(row => row.missing).length };
  }

  get filteredRows() {
    const filter = this.selectedFilter;
    return this.rows.filter(row => filter === null || row[filter]);
  }

  toggleFilter(filter: DashboardFilter): void {
    this.selectedFilter = this.selectedFilter === filter ? null : filter;
  }
}
