import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { API_URL } from './api';
import { Attendance, AttendanceFilter, Dashboard, TimeRecord } from './models';

@Injectable({ providedIn: 'root' })
export class TimeRecordService {
  private readonly http = inject(HttpClient);

  dashboard(): Observable<Dashboard> {
    return this.http.get<Dashboard>(`${API_URL}/dashboard`);
  }

  /** Resumo por funcionário e dia, do mais recente ao mais antigo. */
  attendance(filter: AttendanceFilter = {}): Observable<Attendance[]> {
    return this.http.get<Attendance[]>(`${API_URL}/attendance`, { params: toParams(filter) });
  }

  /** Define todas as batidas do dia ("HH:MM"). Lista vazia apaga o dia. */
  replaceDay(employeeId: number, date: string, times: string[]): Observable<Attendance> {
    return this.http.put<Attendance>(`${API_URL}/attendance/${employeeId}/${date}`, { times });
  }

  listRecords(filter: AttendanceFilter = {}): Observable<TimeRecord[]> {
    return this.http.get<TimeRecord[]>(`${API_URL}/time-records`, { params: toParams(filter) });
  }

  createRecord(employeeId: number, recordedAt: Date): Observable<TimeRecord> {
    return this.http.post<TimeRecord>(`${API_URL}/time-records`, {
      employeeId,
      recordedAt: recordedAt.toISOString(),
    });
  }

  deleteRecord(id: number): Observable<void> {
    return this.http.delete<void>(`${API_URL}/time-records/${id}`);
  }

  /** Relatório para o Excel (CSV com ";"). */
  reportCsvUrl(filter: AttendanceFilter = {}): string {
    return reportUrl('attendance.csv', filter);
  }

  /** Espelho de ponto em PDF, uma folha por funcionário. */
  reportPdfUrl(filter: AttendanceFilter = {}): string {
    return reportUrl('attendance.pdf', filter);
  }
}

function toParams(filter: AttendanceFilter): HttpParams {
  let params = new HttpParams();
  for (const [key, value] of Object.entries(filter)) {
    if (value !== null && value !== undefined && value !== '') params = params.set(key, value);
  }
  return params;
}

function reportUrl(file: string, filter: AttendanceFilter): string {
  const query = toParams(filter).toString();
  return `${API_URL}/reports/${file}${query ? `?${query}` : ''}`;
}
