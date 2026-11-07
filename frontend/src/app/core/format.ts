import { AttendanceStatus } from './models';

export const STATUS_LABELS: Record<AttendanceStatus, string> = {
  normal: 'Normal',
  in_progress: 'Em andamento',
  late: 'Atraso',
  overtime: 'Hora extra',
  incomplete: 'Incompleto',
};

export const STATUS_CLASSES: Record<AttendanceStatus, string> = {
  normal: 'status-normal',
  in_progress: 'status-andamento',
  late: 'status-atraso',
  overtime: 'status-hora-extra',
  incomplete: 'status-incompleto',
};

/** 485 -> "08:05" */
export function formatMinutes(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  return `${String(hours).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}`;
}

/** "2026-11-06" -> "06/11/2026" */
export function formatDate(date: string): string {
  const [year, month, day] = date.split('-');
  return `${day}/${month}/${year}`;
}

/** Data local no formato da API ("YYYY-MM-DD"). */
export function todayIso(): string {
  const now = new Date();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${now.getFullYear()}-${month}-${day}`;
}
