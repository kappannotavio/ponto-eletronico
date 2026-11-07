import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { API_URL } from './api';
import { Employee, EmployeeInput } from './models';

@Injectable({ providedIn: 'root' })
export class EmployeeService {
  private readonly http = inject(HttpClient);
  private readonly url = `${API_URL}/employees`;

  list(filter: { search?: string; active?: boolean } = {}): Observable<Employee[]> {
    let params = new HttpParams();
    if (filter.search) params = params.set('search', filter.search);
    if (filter.active !== undefined) params = params.set('active', filter.active);

    return this.http.get<Employee[]>(this.url, { params });
  }

  get(id: number): Observable<Employee> {
    return this.http.get<Employee>(`${this.url}/${id}`);
  }

  create(input: EmployeeInput): Observable<Employee> {
    return this.http.post<Employee>(this.url, input);
  }

  update(id: number, input: EmployeeInput): Observable<Employee> {
    return this.http.put<Employee>(`${this.url}/${id}`, input);
  }

  setActive(id: number, isActive: boolean): Observable<Employee> {
    return this.http.patch<Employee>(`${this.url}/${id}/status`, { isActive });
  }
}
