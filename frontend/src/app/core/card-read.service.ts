import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable, switchMap, takeWhile, timer } from 'rxjs';

import { API_URL } from './api';
import { CardRead } from './models';

@Injectable({ providedIn: 'root' })
export class CardReadService {
  private readonly http = inject(HttpClient);
  private readonly url = `${API_URL}/card-reads`;

  /** Pede ao relógio de ponto que leia a próxima tag aproximada. */
  create(): Observable<CardRead> {
    return this.http.post<CardRead>(this.url, {});
  }

  get(id: string): Observable<CardRead> {
    return this.http.get<CardRead>(`${this.url}/${id}`);
  }

  cancel(id: string): Observable<CardRead> {
    return this.http.delete<CardRead>(`${this.url}/${id}`);
  }

  /** Consulta o pedido a cada segundo e emite até ele sair de "waiting" (inclusive). */
  watch(id: string): Observable<CardRead> {
    return timer(0, 1000).pipe(
      switchMap(() => this.get(id)),
      takeWhile(read => read.status === 'waiting', true),
    );
  }
}
