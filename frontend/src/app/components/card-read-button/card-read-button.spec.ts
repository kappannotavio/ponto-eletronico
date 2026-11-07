import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';

import { CardReadButton } from './card-read-button';

describe('CardReadButton', () => {
  let fixture: ComponentFixture<CardReadButton>;
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CardReadButton],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();

    fixture = TestBed.createComponent(CardReadButton);
    http = TestBed.inject(HttpTestingController);
    await fixture.whenStable();
  });

  it('emits the tag read by the device', async () => {
    const emitted: string[] = [];
    fixture.componentInstance.cardRead.subscribe(tag => emitted.push(tag));

    fixture.componentInstance.start();
    http.expectOne({ method: 'POST', url: '/api/v1/card-reads' })
      .flush({ id: 'abc', status: 'waiting', createdAt: '', expiresAt: '' });
    await new Promise(resolve => setTimeout(resolve));
    http.expectOne('/api/v1/card-reads/abc')
      .flush({ id: 'abc', status: 'completed', cardTag: 'A1:B2:C3:D4', createdAt: '', expiresAt: '' });
    await fixture.whenStable();

    expect(emitted).toEqual(['A1:B2:C3:D4']);
    expect((fixture.nativeElement as HTMLElement).textContent).toContain('lido com sucesso');
  });
});
