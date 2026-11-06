import { Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { AbstractControl, FormBuilder, ReactiveFormsModule, ValidatorFn, Validators } from '@angular/forms';
import { EmployeeRegistration } from '../../models/employee';

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
  imports: [ReactiveFormsModule],
  selector: 'app-register-employees',
  styleUrl: './register-employees.css',
  templateUrl: './register-employees.html',
})
export class RegisterEmployees {
  private readonly builder = inject(FormBuilder).nonNullable;
  private readonly destroyRef = inject(DestroyRef);
  readonly validated = signal(false);
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
      this.validated.set(false);
    });
  }

  isInvalid(field: keyof EmployeeRegistration): boolean {
    const control = this.form.controls[field];
    return control.invalid && control.touched;
  }

  onSubmit(): EmployeeRegistration | null {
    this.validated.set(false);
    this.form.markAllAsTouched();
    if (this.form.invalid) return null;

    const values = this.form.getRawValue();
    const employee: EmployeeRegistration = {
      ...values,
      fullName: values.fullName.trim(),
      jobTitle: values.jobTitle.trim(),
      whatsapp: values.whatsapp.trim().replace(/[()\s-]/g, ''),
      cardTag: values.cardTag.trim(),
    };
    // Future integration: pass employee to an Angular service here.
    // Do not persist or log the payload: it contains a PIN.
    this.validated.set(true);
    return employee;
  }

  onCancel(): void {
    this.form.reset();
    this.validated.set(false);
  }
}
