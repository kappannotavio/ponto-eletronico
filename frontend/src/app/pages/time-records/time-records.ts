import { afterNextRender, Component, ElementRef, inject, Injector, viewChild } from '@angular/core';

@Component({
  imports: [],
  host: {
    '(document:keydown.escape)': 'closeEditModal()',
    '(document:keydown)': 'trapModalFocus($event)',
  },
  selector: 'app-time-records',
  styleUrl: './time-records.css',
  templateUrl: './time-records.html',
})
export class TimeRecords {
  isEditModalOpen = false;

  private readonly injector = inject(Injector);
  private readonly editDialog = viewChild<ElementRef<HTMLElement>>('editDialog');
  private editTrigger: HTMLElement | null = null;

  openEditModal(event: MouseEvent): void {
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
