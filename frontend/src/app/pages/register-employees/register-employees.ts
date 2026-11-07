import { Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { AbstractControl, FormBuilder, ReactiveFormsModule, ValidatorFn, Validators } from '@angular/forms';

import { CardReadButton } from '../../components/card-read-button/card-read-button';
import { apiErrorMessage } from '../../core/api';
import { EmployeeService } from '../../core/employee.service';
import { EmployeeInput } from '../../core/models';

const nonBlank: ValidatorFn = (control: AbstractControl) =>
  typeof control.value === 'string' && control.value.trim() ? null : { required: true };

const oneOf = (options: readonly number[]): ValidatorFn => control =>
  options.includes(control.value) ? null : { option: true };

// Brazilian number with DDD, optionally formatted and prefixed with +55.
const whatsapp: ValidatorFn = control => {
  const value = control.value.trim();
  return !value || /^(?:\+55\s?)?(?:[1-9]\d|\([1-9]\d\))\s?\d{4,5}-?\d{4}$/.test(value)
    ? null : { whatsapp: true };
};

@Component({
  imports: [ReactiveFormsModule, CardReadButton],
  selector: 'app-register-employees',
  styleUrl: './register-employees.css',
  templateUrl: './register-employees.html',
})
export class RegisterEmployees {
  private readonly builder = inject(FormBuilder).nonNullable;
  private readonly destroyRef = inject(DestroyRef);
  private readonly employees = inject(EmployeeService);
  readonly saving = signal(false);
  readonly error = signal<string | null>(null);
  readonly success = signal<string | null>(null);
  readonly form = this.builder.group({
    fullName: ['', nonBlank],
    jobTitle: [''],
    whatsapp: ['', whatsapp],
    startTime: ['', Validators.pattern(/^([01]\d|2[0-3]):[0-5]\d$/)],
    dailyHours: [4, oneOf([4, 6, 8])],
    lunchBreak: [0, oneOf([0, 15, 30, 60, 90, 120])],
    cardTag: ['', nonBlank],
    pin: ['', [Validators.required, Validators.pattern(/^[0-9]{4}$/)]],
  });

  constructor() {
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroyRef)).subscribe(() => {
      this.success.set(null);
    });
  }

  isInvalid(field: keyof EmployeeInput): boolean {
    const control = this.form.controls[field];
    return control.invalid && control.touched;
  }

  setCardTag(cardTag: string): void {
    this.form.controls.cardTag.setValue(cardTag);
    this.form.controls.cardTag.markAsTouched();
  }

  /** Valida e envia o cadastro. Devolve o payload enviado, ou null se o formulário for inválido. */
  onSubmit(): EmployeeInput | null {
    this.error.set(null);
    this.success.set(null);
    this.form.markAllAsTouched();
    if (this.form.invalid || this.saving()) return null;

    const values = this.form.getRawValue();
    const employee: EmployeeInput = {
      ...values,
      fullName: values.fullName.trim(),
      jobTitle: values.jobTitle.trim(),
      whatsapp: values.whatsapp.trim().replace(/[()\s-]/g, ''),
      cardTag: values.cardTag.trim(),
    };

    // Do not persist or log the payload: it contains a PIN.
    this.saving.set(true);
    this.employees.create(employee).subscribe({
      next: created => {
        this.saving.set(false);
        this.form.reset();
        this.success.set(`${created.fullName} foi cadastrado(a). Já pode bater o ponto com o cartão ${created.cardTag}.`);
      },
      error: error => {
        this.saving.set(false);
        this.error.set(apiErrorMessage(error));
      },
    });

    return employee;
  }

  onCancel(): void {
    this.form.reset();
    this.error.set(null);
    this.success.set(null);
  }
}
