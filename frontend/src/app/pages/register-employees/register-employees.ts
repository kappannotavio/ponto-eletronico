import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';

import { CardReadButton } from '../../components/card-read-button/card-read-button';
import { apiErrorMessage } from '../../core/api';
import { EmployeeService } from '../../core/employee.service';
import { EmployeeInput } from '../../core/models';

function emptyForm(): EmployeeInput {
  return {
    fullName: '',
    jobTitle: '',
    whatsapp: '',
    startTime: '08:00',
    dailyHours: 8,
    lunchBreak: 60,
    cardTag: '',
    pin: '',
  };
}

@Component({
  imports: [FormsModule, RouterLink, CardReadButton],
  selector: 'app-register-employees',
  styleUrl: './register-employees.css',
  templateUrl: './register-employees.html',
})
export class RegisterEmployees {
  protected form = emptyForm();
  protected readonly saving = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly success = signal<string | null>(null);

  private readonly employees = inject(EmployeeService);

  save(): void {
    if (this.saving()) return;

    this.saving.set(true);
    this.error.set(null);
    this.success.set(null);

    this.employees.create(this.form).subscribe({
      next: employee => {
        this.saving.set(false);
        this.success.set(`${employee.fullName} foi cadastrado(a). Já pode bater o ponto com o cartão ${employee.cardTag}.`);
        this.form = emptyForm();
      },
      error: error => {
        this.saving.set(false);
        this.error.set(apiErrorMessage(error));
      },
    });
  }
}
