import { afterNextRender, Component, ElementRef, inject, Injector, viewChild } from '@angular/core';

import { FormsModule } from '@angular/forms';
import { Employee } from '../../models/employee';
import { TimeRecord, TimeRecordStatus } from '../../models/time-record';

@Component({
  imports: [FormsModule],
  host: {
    '(document:keydown.escape)': 'closeEditModal()',
    '(document:keydown)': 'trapModalFocus($event)',
  },
  selector: 'app-time-records',
  styleUrl: './time-records.css',
  templateUrl: './time-records.html',
})
export class TimeRecords {
  readonly employees: Pick<Employee, 'id' | 'fullName'>[] = [
    { id: 1, fullName: 'Ana Souza' },
    { id: 2, fullName: 'Bruno Lima' },
    { id: 3, fullName: 'Carla Mendes' },
  ];
  // Dates assigned to the five original HTML fixtures for filtering and sorting.
  readonly records: TimeRecord[] = [
    { id: 1, employeeId: 1, date: '2026-11-06', entry: '08:00', demonstrationStatus: 'andamento' },
    { id: 2, employeeId: 3, date: '2026-11-05', entry: '08:30', exit: '17:15', demonstrationStatus: 'hora-extra' },
    { id: 3, employeeId: 2, date: '2026-11-05', entry: '09:12', exit: '15:00', demonstrationStatus: 'atraso' },
    { id: 4, employeeId: 1, date: '2026-11-04', entry: '08:02', exit: '17:00', demonstrationStatus: 'normal' },
    { id: 5, employeeId: 3, date: '2026-11-04', entry: '08:28', exit: '17:00', demonstrationStatus: 'normal' },
  ];
  readonly statusLabels: Record<TimeRecordStatus, string> = {
    andamento: 'Em andamento', normal: 'Normal', atraso: 'Atraso', 'hora-extra': 'Hora extra',
  };
  employeeFilter: number | null = null;
  dateFilter = '';
  editingRecord: TimeRecord | null = null;
  editSubmitted = false;
  isEditModalOpen = false;

  get filteredRecords(): TimeRecord[] {
    return this.records.filter(record =>
      (this.employeeFilter === null || record.employeeId === this.employeeFilter) &&
      (!this.dateFilter || record.date === this.dateFilter),
    ).sort((a, b) => b.date.localeCompare(a.date) || b.entry.localeCompare(a.entry) || b.id - a.id);
  }

  employeeName(id: number): string {
    return this.employees.find(employee => employee.id === id)?.fullName ?? 'Funcionário indisponível';
  }

  formatDate(date: string): string {
    return date.split('-').reverse().join('/');
  }

  clearFilters(): void {
    this.employeeFilter = null;
    this.dateFilter = '';
  }

  private minutes(time: string): number | null {
    if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(time)) return null;
    const [hours, minutes] = time.split(':').map(Number);
    return hours * 60 + minutes;
  }

  duration(record: TimeRecord): string {
    const entry = this.minutes(record.entry);
    const exit = this.minutes(record.exit ?? '');
    if (entry === null || exit === null || exit <= entry) return '—';
    const difference = exit - entry;
    return `${Math.floor(difference / 60).toString().padStart(2, '0')}:${(difference % 60).toString().padStart(2, '0')}`;
  }

  get editErrors(): { date?: string; entry?: string; exit?: string } {
    const record = this.editingRecord;
    if (!record) return {};
    const errors: { date?: string; entry?: string; exit?: string } = {};
    const date = new Date(`${record.date}T00:00:00Z`);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(record.date) || record.date.startsWith('0000') ||
        Number.isNaN(date.getTime()) || date.toISOString().slice(0, 10) !== record.date) {
      errors.date = 'Informe uma data válida.';
    }
    const entry = this.minutes(record.entry);
    const exit = this.minutes(record.exit ?? '');
    if (entry === null) errors.entry = 'Informe uma entrada válida (HH:mm).';
    if (record.exit && exit === null) errors.exit = 'Informe uma saída válida (HH:mm).';
    else if (exit !== null && entry !== null && exit <= entry) {
      errors.exit = 'A saída deve ser posterior à entrada, no mesmo dia.';
    }
    return errors;
  }

  saveRecord(): void {
    this.editSubmitted = true;
    const draft = this.editingRecord;
    if (!draft || Object.keys(this.editErrors).length) return;
    const index = this.records.findIndex(record => record.id === draft.id);
    if (index < 0) return;
    this.records[index] = { ...this.records[index], date: draft.date, entry: draft.entry, exit: draft.exit || undefined };
    this.closeEditModal();
    // If a date edit removes the row from the current filter, focus a stable control.
    afterNextRender(() => {
      if (this.editTrigger === null && !document.activeElement?.closest('.time-records-page')) {
        this.clearButton()?.nativeElement.focus();
      }
    }, { injector: this.injector });
  }

  private readonly injector = inject(Injector);
  private readonly editDialog = viewChild<ElementRef<HTMLElement>>('editDialog');
  private readonly clearButton = viewChild<ElementRef<HTMLButtonElement>>('clearButton');
  private editTrigger: HTMLElement | null = null;

  openEditModal(event: MouseEvent, record: TimeRecord): void {
    this.editingRecord = { ...record };
    this.editSubmitted = false;
    this.editTrigger = event.currentTarget as HTMLElement;
    this.isEditModalOpen = true;

    afterNextRender(() => {
      this.editDialog()?.nativeElement
        .querySelector<HTMLElement>('input:not([disabled]):not([readonly]), select:not([disabled]), textarea:not([disabled]):not([readonly])')
        ?.focus();
    }, { injector: this.injector });
  }

  closeEditModal(): void {
    if (!this.isEditModalOpen) return;

    this.isEditModalOpen = false;
    this.editingRecord = null;
    this.editSubmitted = false;
    this.editTrigger?.focus();
    this.editTrigger = null;
  }

  trapModalFocus(event: KeyboardEvent): void {
    if (event.key !== 'Tab') return;
    const dialog = this.editDialog()?.nativeElement;
    if (!this.isEditModalOpen || !dialog) return;

    const focusable = Array.from(dialog.querySelectorAll<HTMLElement>(
      'input:not([type="hidden"]), select, textarea, button, a[href], [tabindex]',
    )).filter(element => element.tabIndex >= 0 && !element.matches(':disabled, [hidden], [inert], [hidden] *, [inert] *'));
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const active = dialog.ownerDocument.activeElement;

    // Keep native Tab navigation except when focus would leave the dialog.
    if (event.shiftKey && (active === first || !dialog.contains(active))) {
      event.preventDefault();
      last?.focus();
    } else if (!event.shiftKey && (active === last || !dialog.contains(active))) {
      event.preventDefault();
      first?.focus();
    }
  }
}
