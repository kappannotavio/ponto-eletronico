export interface Employee {
  readonly id: number;
  fullName: string;
  jobTitle: string;
  whatsapp: string;
  startTime: string;
  dailyHours: number;
  lunchBreak: number;
  cardTag: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

/** Corpo de criação/edição. Na edição, `pin` vazio mantém a senha atual. */
export interface EmployeeInput {
  fullName: string;
  jobTitle: string;
  whatsapp: string;
  startTime: string;
  dailyHours: number;
  lunchBreak: number;
  cardTag: string;
  pin: string;
}

export interface TimeRecord {
  readonly id: number;
  employeeId: number;
  cardTag: string;
  recordedAt: string;
  source: 'device' | 'manual';
  createdAt: string;
}

export type AttendanceStatus = 'normal' | 'in_progress' | 'late' | 'overtime' | 'incomplete';

export interface Attendance {
  employeeId: number;
  employeeName: string;
  date: string;
  entry: string | null;
  exit: string | null;
  workedMinutes: number;
  late: boolean;
  status: AttendanceStatus;
  times: string[];
  records: TimeRecord[];
}

export interface Dashboard {
  date: string;
  activeCount: number;
  presentCount: number;
  lateCount: number;
  absentCount: number;
  today: Attendance[];
}

export type CardReadStatus = 'waiting' | 'completed' | 'cancelled' | 'expired';

export interface CardRead {
  id: string;
  status: CardReadStatus;
  cardTag?: string;
  assignedTo?: { id: number; fullName: string };
  createdAt: string;
  expiresAt: string;
}

export interface AttendanceFilter {
  employeeId?: number | null;
  date?: string | null;
  from?: string | null;
  to?: string | null;
}
