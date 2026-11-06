import { Employee } from './employee';

export type TimeRecordStatus = 'andamento' | 'normal' | 'atraso' | 'hora-extra';

export interface TimeRecord {
  readonly id: number;
  readonly employeeId: Employee['id'];
  date: string; // Local calendar date, YYYY-MM-DD; no timezone conversion.
  entry: string; // HH:mm, same-day records only in this demonstration.
  exit?: string;
  // Fixture only: the backend will determine status using business rules.
  readonly demonstrationStatus: TimeRecordStatus;
}
