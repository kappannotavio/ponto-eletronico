import { afterNextRender, Component, ElementRef, inject, Injector, signal, viewChild } from '@angular/core';

import { FormsModule } from '@angular/forms';

import { CardReadButton } from '../../components/card-read-button/card-read-button';
import { apiErrorMessage } from '../../core/api';
import { EmployeeService } from '../../core/employee.service';
import { Employee, EmployeeInput } from '../../core/models';

type EmployeeForm = EmployeeInput & { readonly id: number };

@Component({
  imports: [FormsModule, CardReadButton],
  host: {
    '(document:keydown.escape)': 'closeEditModal()',
    '(document:keydown)': 'trapModalFocus($event)',
  },
  selector: 'app-employees',
  styleUrl: './employees.css',
  templateUrl: './employees.html',
})
export class Employees {
  readonly employees = signal<Employee[]>([]);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly editingEmployee = signal<EmployeeForm | null>(null);
  readonly editError = signal<string | null>(null);
  readonly saving = signal(false);
  searchTerm = '';

  get filteredEmployees(): Employee[] {
    const search = this.searchTerm.toLowerCase();

    return this.employees().filter(employee =>
      employee.fullName.toLowerCase().includes(search) ||
      employee.jobTitle.toLowerCase().includes(search) ||
      employee.cardTag.toLowerCase().includes(search),
    );
  }

  private readonly employeeService = inject(EmployeeService);
  private readonly injector = inject(Injector);
  private readonly editDialog = viewChild<ElementRef<HTMLElement>>('editDialog');
  private editTrigger: HTMLElement | null = null;

  constructor() {
    this.loadEmployees();
  }

  loadEmployees(): void {
    this.loading.set(true);
    this.employeeService.list().subscribe({
      next: employees => {
        this.employees.set(employees);
        this.error.set(null);
        this.loading.set(false);
      },
      error: error => {
        this.error.set(apiErrorMessage(error));
        this.loading.set(false);
      },
    });
  }

  toggleEmployeeStatus(employee: Employee): void {
    this.employeeService.setActive(employee.id, !employee.isActive).subscribe({
      next: updated => this.replaceEmployee(updated),
      error: error => this.error.set(apiErrorMessage(error)),
    });
  }

  openEditModal(event: MouseEvent, employee: Employee): void {
    const { id, fullName, jobTitle, whatsapp, startTime, dailyHours, lunchBreak, cardTag } = employee;
    this.editingEmployee.set({ id, fullName, jobTitle, whatsapp, startTime, dailyHours, lunchBreak, cardTag, pin: '' });
    this.editError.set(null);
    this.editTrigger = event.currentTarget as HTMLElement;

    afterNextRender(() => {
      this.editDialog()?.nativeElement
        .querySelector<HTMLElement>('input:not([disabled]):not([readonly]), select:not([disabled]), textarea:not([disabled]):not([readonly])')
        ?.focus();
    }, { injector: this.injector });
  }

  saveEmployee(): void {
    const form = this.editingEmployee();
    if (!form || this.saving()) return;

    const { id, ...input } = form;
    this.saving.set(true);
    this.editError.set(null);

    this.employeeService.update(id, input).subscribe({
      next: updated => {
        this.saving.set(false);
        this.replaceEmployee(updated);
        this.closeEditModal();
      },
      error: error => {
        this.saving.set(false);
        this.editError.set(apiErrorMessage(error));
      },
    });
  }

  closeEditModal(): void {
    if (!this.editingEmployee()) return;

    this.editingEmployee.set(null);
    this.editTrigger?.focus();
    this.editTrigger = null;
  }

  trapModalFocus(event: KeyboardEvent): void {
    if (event.key !== 'Tab') return;
    const dialog = this.editDialog()?.nativeElement;
    if (!this.editingEmployee() || !dialog) return;

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

  private replaceEmployee(updated: Employee): void {
    this.employees.update(employees => employees.map(employee => employee.id === updated.id ? updated : employee));
  }
}
