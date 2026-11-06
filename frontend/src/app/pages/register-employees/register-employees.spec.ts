import { ComponentFixture, TestBed } from '@angular/core/testing';
import { RegisterEmployees } from './register-employees';

describe('RegisterEmployees', () => {
  let component: RegisterEmployees;
  let fixture: ComponentFixture<RegisterEmployees>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RegisterEmployees],
    }).compileComponents();

    fixture = TestBed.createComponent(RegisterEmployees);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  const validValues = {
    fullName: '  Ana Souza  ', jobTitle: ' Analista ', whatsapp: '+55 (11) 99999-9999',
    startTime: '08:00', dailyHours: 8, lunchBreak: 60, cardTag: ' 00123 ', pin: '0012',
  };

  it('rejects empty and whitespace-only required fields', () => {
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

  it('builds a JSON-ready payload preserving string identifiers and numeric options', () => {
    component.form.setValue(validValues);
    const payload = component.onSubmit();
    expect(payload).toEqual({ ...validValues, fullName: 'Ana Souza', jobTitle: 'Analista',
      whatsapp: '+5511999999999', cardTag: '00123' });
    expect(JSON.parse(JSON.stringify(payload))).toEqual(payload);
    expect(component.validated()).toBe(true);
    component.form.controls.fullName.setValue('Outro nome');
    expect(component.validated()).toBe(false);
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
    await fixture.whenStable();
    expect(element.querySelector('[role="status"]')?.textContent).toContain('ainda não enviados');
    element.querySelector<HTMLButtonElement>('button[type="button"]')!.click();
    await fixture.whenStable();
    expect(component.form.getRawValue()).toEqual({ fullName: '', jobTitle: '', whatsapp: '',
      startTime: '', dailyHours: 4, lunchBreak: 0, cardTag: '', pin: '' });
    expect(component.form.pristine).toBe(true);
    expect(component.form.untouched).toBe(true);
    expect(element.querySelector('.field-error')).toBeNull();
    expect(element.querySelector('[role="status"]')).toBeNull();
  });
});
