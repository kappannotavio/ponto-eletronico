import { Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { switchMap, timer } from 'rxjs';

import { apiErrorMessage } from '../../core/api';
import { formatDate, formatMinutes, STATUS_CLASSES, STATUS_LABELS } from '../../core/format';
import { Dashboard as DashboardData } from '../../core/models';
import { TimeRecordService } from '../../core/time-record.service';

const REFRESH_INTERVAL_MS = 15_000;

@Component({
  imports: [],
  selector: 'app-dashboard',
  styleUrl: './dashboard.css',
  templateUrl: './dashboard.html',
})
export class Dashboard {
  protected readonly data = signal<DashboardData | null>(null);
  protected readonly error = signal<string | null>(null);
  protected readonly greeting = greetingFor(new Date());

  protected readonly statusLabels = STATUS_LABELS;
  protected readonly statusClasses = STATUS_CLASSES;
  protected readonly formatMinutes = formatMinutes;
  protected readonly formatDate = formatDate;

  constructor() {
    const timeRecords = inject(TimeRecordService);

    // Atualiza sozinho para mostrar as batidas que chegam do relógio.
    timer(0, REFRESH_INTERVAL_MS)
      .pipe(switchMap(() => timeRecords.dashboard()), takeUntilDestroyed())
      .subscribe({
        next: data => {
          this.data.set(data);
          this.error.set(null);
        },
        error: error => this.error.set(apiErrorMessage(error)),
      });
  }
}

function greetingFor(now: Date): string {
  const hour = now.getHours();
  if (hour < 12) return 'Bom dia!';
  if (hour < 18) return 'Boa tarde!';
  return 'Boa noite!';
}
