import { Component, DestroyRef, inject, input, output, signal } from '@angular/core';
import { Subscription } from 'rxjs';

import { apiErrorMessage } from '../../core/api';
import { CardReadService } from '../../core/card-read.service';
import { CardRead } from '../../core/models';

type Feedback = { kind: 'info' | 'success' | 'warning' | 'error'; text: string };

/**
 * Botão "Ler no relógio": pede ao ESP32 que leia a próxima tag aproximada
 * e emite o UID lido em `cardRead`.
 */
@Component({
  selector: 'app-card-read-button',
  styleUrl: './card-read-button.css',
  templateUrl: './card-read-button.html',
})
export class CardReadButton {
  /** Funcionário sendo editado, para não alertar quando a tag já é dele. */
  readonly employeeId = input<number | null>(null);
  readonly cardRead = output<string>();

  protected readonly waiting = signal(false);
  protected readonly feedback = signal<Feedback | null>(null);

  private readonly cardReads = inject(CardReadService);
  private currentRead: CardRead | null = null;
  private subscription: Subscription | null = null;

  constructor() {
    inject(DestroyRef).onDestroy(() => this.cancel());
  }

  start(): void {
    this.stopWatching();
    this.waiting.set(true);
    this.feedback.set({ kind: 'info', text: 'Aproxime o cartão do relógio de ponto…' });

    this.subscription = this.cardReads.create().subscribe({
      next: read => {
        this.currentRead = read;
        this.subscription = this.cardReads.watch(read.id).subscribe({
          next: update => this.handleUpdate(update),
          error: error => this.fail(error),
        });
      },
      error: error => this.fail(error),
    });
  }

  cancel(): void {
    if (!this.waiting()) return;

    if (this.currentRead) this.cardReads.cancel(this.currentRead.id).subscribe({ error: () => {} });
    this.stopWatching();
    this.currentRead = null;
    this.waiting.set(false);
    this.feedback.set(null);
  }

  private handleUpdate(read: CardRead): void {
    if (read.status === 'waiting') return;

    this.stopWatching();
    this.currentRead = null;
    this.waiting.set(false);

    if (read.status === 'completed' && read.cardTag) {
      this.cardRead.emit(read.cardTag);

      const owner = read.assignedTo;
      this.feedback.set(owner && owner.id !== this.employeeId()
        ? { kind: 'warning', text: `Cartão ${read.cardTag} lido, mas ele já pertence a ${owner.fullName}.` }
        : { kind: 'success', text: `Cartão ${read.cardTag} lido com sucesso.` });
    } else if (read.status === 'expired') {
      this.feedback.set({ kind: 'error', text: 'Nenhum cartão foi lido a tempo. Tente novamente.' });
    } else {
      this.feedback.set({ kind: 'error', text: 'A leitura foi cancelada.' });
    }
  }

  private fail(error: unknown): void {
    this.stopWatching();
    this.currentRead = null;
    this.waiting.set(false);
    this.feedback.set({ kind: 'error', text: apiErrorMessage(error) });
  }

  private stopWatching(): void {
    this.subscription?.unsubscribe();
    this.subscription = null;
  }
}
