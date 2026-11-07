import { Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { forkJoin, switchMap, timer } from 'rxjs';

import { apiErrorMessage } from '../../core/api';
import { EmployeeService } from '../../core/employee.service';
import { formatDate, formatMinutes, STATUS_CLASSES, STATUS_LABELS } from '../../core/format';
import { Dashboard as DashboardData, Employee } from '../../core/models';
import { TimeRecordService } from '../../core/time-record.service';

type DashboardFilter = 'present' | 'late' | 'missing';

const REFRESH_INTERVAL_MS = 15_000;

@Component({
  imports: [],
  selector: 'app-dashboard',
  styleUrl: './dashboard.css',
  templateUrl: './dashboard.html',
})
export class Dashboard {
  readonly data = signal<DashboardData | null>(null);
  readonly employees = signal<Employee[]>([]);
  readonly error = signal<string | null>(null);
  readonly greeting = greetingFor(new Date());
  selectedFilter: DashboardFilter | null = null;

  protected readonly formatDate = formatDate;

  constructor() {
    const timeRecords = inject(TimeRecordService);
    const employees = inject(EmployeeService);

    // Atualiza sozinho para mostrar as batidas que chegam do relógio.
    timer(0, REFRESH_INTERVAL_MS)
      .pipe(
        switchMap(() => forkJoin({ dashboard: timeRecords.dashboard(), employees: employees.list({ active: true }) })),
        takeUntilDestroyed(),
      )
      .subscribe({
        next: ({ dashboard, employees }) => {
          this.data.set(dashboard);
          this.employees.set(employees);
          this.error.set(null);
        },
        error: error => this.error.set(apiErrorMessage(error)),
      });
  }

  // Uma linha por funcionário ativo; quem ainda não bateu ponto aparece como "Sem marcação".
  get rows() {
    const today = this.data()?.today ?? [];

    return this.employees().map(employee => {
      const day = today.find(attendance => attendance.employeeId === employee.id);
      const present = !!day;

      return {
        employeeId: employee.id,
        fullName: employee.fullName,
        present,
        late: !!day?.late,
        missing: !present,
        entry: day?.entry ?? '—',
        exit: day?.exit ?? '—',
        duration: day?.workedMinutes ? formatMinutes(day.workedMinutes) : '—',
        status: day ? STATUS_LABELS[day.status] : 'Sem marcação',
        statusClass: day ? STATUS_CLASSES[day.status] : 'status-sem-marcacao',
      };
    });
  }

  get totals() {
    const rows = this.rows;
    return {
      present: rows.filter(row => row.present).length,
      late: rows.filter(row => row.late).length,
      missing: rows.filter(row => row.missing).length,
    };
  }

  get filteredRows() {
    const filter = this.selectedFilter;
    return this.rows.filter(row => filter === null || row[filter]);
  }

  toggleFilter(filter: DashboardFilter): void {
    this.selectedFilter = this.selectedFilter === filter ? null : filter;
  }
}

function greetingFor(now: Date): string {
  const hour = now.getHours();
  if (hour < 12) return 'Bom dia!';
  if (hour < 18) return 'Boa tarde!';
  return 'Boa noite!';
}
