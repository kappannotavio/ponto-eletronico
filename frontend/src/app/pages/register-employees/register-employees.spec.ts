import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { RegisterEmployees } from './register-employees';

describe('RegisterEmployees', () => {
  let component: RegisterEmployees;
  let fixture: ComponentFixture<RegisterEmployees>;
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RegisterEmployees],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(RegisterEmployees);
    component = fixture.componentInstance;
    http = TestBed.inject(HttpTestingController);
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  const validValues = {
    fullName: '  Ana Souza  ', jobTitle: ' Analista ', whatsapp: '+55 (11) 99999-9999',
    startTime: '08:00', dailyHours: 8, lunchBreak: 60, cardTag: ' 00123 ', pin: '0012',
  };

  it('rejects empty and whitespace-only required fields without calling the API', () => {
    component.form.patchValue({ fullName: '   ', cardTag: '  ' });
    expect(component.onSubmit()).toBeNull();
    expect(component.isInvalid('fullName')).toBe(true);
    expect(component.isInvalid('cardTag')).toBe(true);
    expect(component.isInvalid('pin')).toBe(true);
  });

  it('requires exactly four ASCII digits for the PIN', () => {
    for (const pin of ['', '123', '12345', 'abcd', '12 3', ' 1234']) {
      component.form.controls.pin.setValue(pin);
      expect(component.form.controls.pin.invalid).toBe(true);
    }
    component.form.controls.pin.setValue('0012');
    expect(component.form.controls.pin.valid).toBe(true);
  });

  it('validates optional WhatsApp and time', () => {
    for (const value of ['', '   ', '11999999999', '(11) 99999-9999', '+55 11 99999-9999']) {
      component.form.controls.whatsapp.setValue(value);
      expect(component.form.controls.whatsapp.valid).toBe(true);
    }
    for (const value of ['abc', '99999-9999', '123', '+1 99999999999']) {
      component.form.controls.whatsapp.setValue(value);
      expect(component.form.controls.whatsapp.invalid).toBe(true);
    }
    component.form.controls.startTime.setValue('25:00');
    expect(component.form.controls.startTime.invalid).toBe(true);
  });

  it('only accepts the existing numeric options', () => {
    for (const value of [4, 6, 8]) {
      component.form.controls.dailyHours.setValue(value);
      expect(component.form.controls.dailyHours.valid).toBe(true);
    }
    for (const value of [0, 15, 30, 60, 90, 120]) {
      component.form.controls.lunchBreak.setValue(value);
      expect(component.form.controls.lunchBreak.valid).toBe(true);
    }
    component.form.patchValue({ dailyHours: 7, lunchBreak: -1 });
    expect(component.onSubmit()).toBeNull();
    expect(component.form.controls.dailyHours.invalid).toBe(true);
    expect(component.form.controls.lunchBreak.invalid).toBe(true);
  });

  it('sends a normalized payload to the API and resets the form on success', async () => {
    component.form.setValue(validValues);
    const payload = component.onSubmit();
    const expected = { ...validValues, fullName: 'Ana Souza', jobTitle: 'Analista', whatsapp: '+5511999999999', cardTag: '00123' };
    expect(payload).toEqual(expected);

    const request = http.expectOne({ method: 'POST', url: '/api/v1/employees' });
    expect(request.request.body).toEqual(expected);
    request.flush({ ...expected, id: 1, isActive: true, createdAt: '', updatedAt: '' });
    await fixture.whenStable();

    expect(component.success()).toContain('Ana Souza');
    expect(component.form.controls.fullName.value).toBe('');
    component.form.controls.fullName.setValue('Outro nome');
    expect(component.success()).toBeNull();
  });

  it('keeps the data and shows the API error when saving fails', async () => {
    component.form.setValue(validValues);
    component.onSubmit();
    http.expectOne('/api/v1/employees')
      .flush({ error: 'a tag 00123 ja pertence a Carla Mendes' }, { status: 409, statusText: 'Conflict' });
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain('ja pertence a Carla Mendes');
    expect(component.form.controls.fullName.value).toBe(validValues.fullName);
  });

  it('wires inputs, numeric selects, save and cancel to the form', async () => {
    const element = fixture.nativeElement as HTMLElement;
    element.querySelector<HTMLButtonElement>('button[type="submit"]')!.click();
    await fixture.whenStable();
    expect(element.querySelector('#full-name')?.parentElement?.querySelector('#fullName-error')).not.toBeNull();
    expect(element.querySelector('#full-name')?.getAttribute('aria-invalid')).toBe('true');
    component.form.setValue(validValues);
    const name = element.querySelector<HTMLInputElement>('#full-name')!;
    name.value = 'Maria Souza';
    name.dispatchEvent(new Event('input'));
    const hours = element.querySelector<HTMLSelectElement>('#daily-hours')!;
    hours.selectedIndex = 1;
    hours.dispatchEvent(new Event('change'));
    expect(component.form.controls.fullName.value).toBe('Maria Souza');
    expect(component.form.controls.dailyHours.value).toBe(6);
    element.querySelector<HTMLButtonElement>('button[type="submit"]')!.click();
    http.expectOne('/api/v1/employees').flush({ error: 'erro' }, { status: 400, statusText: 'Bad Request' });
    await fixture.whenStable();
    element.querySelector<HTMLButtonElement>('.button-secondary')!.click();
    await fixture.whenStable();
    expect(component.form.getRawValue()).toEqual({ fullName: '', jobTitle: '', whatsapp: '',
      startTime: '', dailyHours: 4, lunchBreak: 0, cardTag: '', pin: '' });
    expect(component.form.pristine).toBe(true);
    expect(component.form.untouched).toBe(true);
    expect(element.querySelector('.field-error')).toBeNull();
    expect(element.querySelector('.alert')).toBeNull();
  });

  it('fills the card field with the tag read by the time clock', () => {
    component.setCardTag('A1:B2:C3:D4');
    expect(component.form.controls.cardTag.value).toBe('A1:B2:C3:D4');
  });
});
