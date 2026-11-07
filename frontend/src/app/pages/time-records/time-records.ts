import { afterNextRender, Component, computed, ElementRef, inject, Injector, signal, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { map, Observable, of, switchMap } from 'rxjs';

import { apiErrorMessage } from '../../core/api';
import { EmployeeService } from '../../core/employee.service';
import { formatDate, formatMinutes, STATUS_CLASSES, STATUS_LABELS, todayIso } from '../../core/format';
import { Attendance, Employee } from '../../core/models';
import { TimeRecordService } from '../../core/time-record.service';

interface DayForm {
  mode: 'edit' | 'create';
  employeeId: number | null;
  employeeName: string;
  date: string;
  entry: string;
  exit: string;
  /** Batidas entre a entrada e a saída (almoço), preservadas na edição. */
  middle: string[];
}

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
  readonly employees = signal<Employee[]>([]);
  readonly days = signal<Attendance[]>([]);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly editingDay = signal<DayForm | null>(null);
  readonly editError = signal<string | null>(null);
  readonly saving = signal(false);
  editSubmitted = false;

  readonly filterEmployeeId = signal<number | null>(null);
  readonly filterDate = signal('');

  readonly csvUrl = computed(() => this.timeRecords.reportCsvUrl(this.currentFilter()));
  readonly pdfUrl = computed(() => this.timeRecords.reportPdfUrl(this.currentFilter()));

  protected readonly statusLabels = STATUS_LABELS;
  protected readonly statusClasses = STATUS_CLASSES;
  protected readonly formatMinutes = formatMinutes;
  protected readonly formatDate = formatDate;

  private readonly timeRecords = inject(TimeRecordService);
  private readonly employeeService = inject(EmployeeService);
  private readonly injector = inject(Injector);
  private readonly editDialog = viewChild<ElementRef<HTMLElement>>('editDialog');
  private editTrigger: HTMLElement | null = null;

  constructor() {
    this.employeeService.list().subscribe({
      next: employees => this.employees.set(employees),
      error: error => this.error.set(apiErrorMessage(error)),
    });
    this.loadDays();
  }

  setEmployeeFilter(employeeId: number | null): void {
    this.filterEmployeeId.set(employeeId);
    this.loadDays();
  }

  setDateFilter(date: string): void {
    this.filterDate.set(date);
    this.loadDays();
  }

  clearFilters(): void {
    this.filterEmployeeId.set(null);
    this.filterDate.set('');
    this.loadDays();
  }

  loadDays(): void {
    this.loading.set(true);
    this.timeRecords.attendance(this.currentFilter()).subscribe({
      next: days => {
        this.days.set(days);
        this.error.set(null);
        this.loading.set(false);
      },
      error: error => {
        this.error.set(apiErrorMessage(error));
        this.loading.set(false);
      },
    });
  }

  openEditModal(event: MouseEvent, day: Attendance): void {
    const middle = day.exit ? day.times.slice(1, -1) : day.times.slice(1);

    this.openModal(event, {
      mode: 'edit',
      employeeId: day.employeeId,
      employeeName: day.employeeName,
      date: day.date,
      entry: day.entry ?? '',
      exit: day.exit ?? '',
      middle,
    });
  }

  openCreateModal(event: MouseEvent): void {
    this.openModal(event, {
      mode: 'create',
      employeeId: this.filterEmployeeId(),
      employeeName: '',
      date: this.filterDate() || todayIso(),
      entry: '',
      exit: '',
      middle: [],
    });
  }

  /** Erros do formulário de marcação, exibidos depois da primeira tentativa de salvar. */
  get editErrors(): { employee?: string; date?: string; entry?: string; exit?: string } {
    const form = this.editingDay();
    if (!form) return {};

    const errors: { employee?: string; date?: string; entry?: string; exit?: string } = {};
    const date = new Date(`${form.date}T00:00:00Z`);
    if (!form.employeeId) errors.employee = 'Escolha o funcionário.';
    if (!/^\d{4}-\d{2}-\d{2}$/.test(form.date) || form.date.startsWith('0000') ||
        Number.isNaN(date.getTime()) || date.toISOString().slice(0, 10) !== form.date) {
      errors.date = 'Informe uma data válida.';
    }

    const entry = minutes(form.entry);
    const exit = minutes(form.exit);
    if (entry === null) errors.entry = 'Informe uma entrada válida (HH:mm).';
    if (form.exit && exit === null) errors.exit = 'Informe uma saída válida (HH:mm).';
    else if (exit !== null && entry !== null && exit <= entry) {
      errors.exit = 'A saída deve ser posterior à entrada, no mesmo dia.';
    }
    return errors;
  }

  saveDay(): void {
    const form = this.editingDay();
    if (!form || this.saving()) return;

    this.editSubmitted = true;
    if (Object.keys(this.editErrors).length) return;

    const { employeeId, date } = form;
    if (!employeeId) return;

    const typed = [form.entry, ...form.middle, form.exit].filter(time => time);

    // Na criação, as batidas digitadas se somam às que o dia já tiver.
    const times$: Observable<string[]> = form.mode === 'edit'
      ? of(typed)
      : this.timeRecords.attendance({ employeeId, date }).pipe(
        map(existing => [...(existing[0]?.times ?? []), ...typed]),
      );

    this.persistDay(employeeId, date, times$);
  }

  deleteDay(): void {
    const form = this.editingDay();
    if (!form?.employeeId || this.saving()) return;
    if (!window.confirm(`Apagar todas as batidas de ${form.employeeName} em ${formatDate(form.date)}?`)) return;

    this.persistDay(form.employeeId, form.date, of([]));
  }

  closeEditModal(): void {
    if (!this.editingDay()) return;

    this.editingDay.set(null);
    this.editTrigger?.focus();
    this.editTrigger = null;
  }

  trapModalFocus(event: KeyboardEvent): void {
    if (event.key !== 'Tab') return;
    const dialog = this.editDialog()?.nativeElement;
    if (!this.editingDay() || !dialog) return;

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

  private openModal(event: MouseEvent, form: DayForm): void {
    this.editingDay.set(form);
    this.editError.set(null);
    this.editSubmitted = false;
    this.editTrigger = event.currentTarget as HTMLElement;

    afterNextRender(() => {
      this.editDialog()?.nativeElement
        .querySelector<HTMLElement>('input:not([disabled]):not([readonly]), select:not([disabled]), textarea:not([disabled]):not([readonly])')
        ?.focus();
    }, { injector: this.injector });
  }

  private persistDay(employeeId: number, date: string, times$: Observable<string[]>): void {
    this.saving.set(true);
    this.editError.set(null);

    times$.pipe(switchMap(times => this.timeRecords.replaceDay(employeeId, date, times))).subscribe({
      next: () => {
        this.saving.set(false);
        this.closeEditModal();
        this.loadDays();
      },
      error: error => {
        this.saving.set(false);
        this.editError.set(apiErrorMessage(error));
      },
    });
  }

  private currentFilter() {
    return { employeeId: this.filterEmployeeId(), date: this.filterDate() };
  }
}

function minutes(time: string): number | null {
  if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(time)) return null;
  const [hours, mins] = time.split(':').map(Number);
  return hours * 60 + mins;
}
