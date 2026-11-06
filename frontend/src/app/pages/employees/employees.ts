import { afterNextRender, Component, ElementRef, inject, Injector, viewChild } from '@angular/core';

import { FormsModule } from '@angular/forms';

import { Employee } from '../../models/employee';

@Component({
  imports: [FormsModule],
  host: {
    '(document:keydown.escape)': 'closeEditModal()',
    '(document:keydown)': 'trapModalFocus($event)',
  },
  selector: 'app-employees',
  styleUrl: './employees.css',
  templateUrl: './employees.html',
})
export class Employees {
  readonly employees: Employee[] = [
    {
      id: 1, fullName: 'Ana Souza', jobTitle: 'Recepcionista',
      whatsapp: '11900000001', startTime: '08:00', dailyHours: 8,
      lunchBreak: 60, cardTag: 'TAG-001', pin: '1001', isActive: true,
    },
    {
      id: 2, fullName: 'Bruno Lima', jobTitle: 'Auxiliar administrativo',
      whatsapp: '11900000002', startTime: '09:00', dailyHours: 6,
      lunchBreak: 30, cardTag: 'TAG-002', pin: '1002', isActive: false,
    },
    {
      id: 3, fullName: 'Carla Mendes', jobTitle: 'Assistente de loja',
      whatsapp: '11900000003', startTime: '08:30', dailyHours: 8,
      lunchBreak: 60, cardTag: 'TAG-003', pin: '1003', isActive: true,
    },
  ];
  editingEmployee: Employee | null = null;
  isEditModalOpen = false;
  searchTerm = '';

  get filteredEmployees(): Employee[] {
    const search = this.searchTerm.toLowerCase();

    return this.employees.filter(employee =>
      employee.fullName.toLowerCase().includes(search) ||
      employee.jobTitle.toLowerCase().includes(search),
    );
  }

  private readonly injector = inject(Injector);
  private readonly editDialog = viewChild<ElementRef<HTMLElement>>('editDialog');
  private editTrigger: HTMLElement | null = null;

  toggleEmployeeStatus(employee: Employee): void {
    employee.isActive = !employee.isActive;
  }

  openEditModal(event: MouseEvent, employee: Employee): void {
    this.editingEmployee = { ...employee };
    this.editTrigger = event.currentTarget as HTMLElement;
    this.isEditModalOpen = true;

    afterNextRender(() => {
      this.editDialog()?.nativeElement
        .querySelector<HTMLElement>('input:not([disabled]):not([readonly]), select:not([disabled]), textarea:not([disabled]):not([readonly])')
        ?.focus();
    }, { injector: this.injector });
  }

  saveEmployee(): void {
    if (!this.editingEmployee) return;

    const { id, ...changes } = this.editingEmployee;
    const employee = this.employees.find(employee => employee.id === id);
    if (!employee) return;

    Object.assign(employee, changes);
    this.closeEditModal();
  }

  closeEditModal(): void {
    if (!this.isEditModalOpen) return;

    this.isEditModalOpen = false;
    this.editingEmployee = null;
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
