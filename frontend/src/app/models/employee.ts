export interface Employee {
  readonly id: number;
  fullName: string;
  jobTitle: string;
  whatsapp: string;
  startTime: string;
  dailyHours: number;
  lunchBreak: number;
  cardTag: string;
  pin: string;
  isActive: boolean;
}

export type EmployeeRegistration = Omit<Employee, 'id' | 'isActive'>;
