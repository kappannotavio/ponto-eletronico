import { HttpErrorResponse } from '@angular/common/http';

// Em desenvolvimento o `ng serve` encaminha /api para o backend (proxy.conf.json).
export const API_URL = '/api/v1';

export function apiErrorMessage(error: unknown): string {
  if (error instanceof HttpErrorResponse) {
    if (error.status === 0) return 'Não foi possível conectar ao servidor. O backend está rodando?';
    if (typeof error.error?.error === 'string') return error.error.error;
  }

  return 'Ocorreu um erro inesperado. Tente novamente.';
}
