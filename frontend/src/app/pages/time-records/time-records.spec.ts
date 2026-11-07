import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting, TestRequest } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Attendance, Employee } from '../../core/models';
import { TimeRecords } from './time-records';

const employees = [1, 2, 3].map(id => ({
  id, fullName: ['Ana Souza', 'Bruno Lima', 'Carla Mendes'][id - 1], jobTitle: '', whatsapp: '',
  startTime: '08:00', dailyHours: 8, lunchBreak: 60, cardTag: `TAG-${id}`, isActive: true, createdAt: '', updatedAt: '',
})) satisfies Employee[];

function day(employeeId: number, date: string, times: string[], changes: Partial<Attendance> = {}): Attendance {
  const complete = times.length % 2 === 0;
  return {
    employeeId, employeeName: employees[employeeId - 1].fullName, date, times, records: [],
    entry: times[0] ?? null, exit: complete ? times[times.length - 1] : null,
    workedMinutes: 0, late: false, status: complete ? 'normal' : 'in_progress', ...changes,
  };
}

// A API devolve do dia mais recente para o mais antigo.
const days: Attendance[] = [
  day(1, '2026-11-06', ['08:00']),
  day(3, '2026-11-05', ['08:30', '12:00', '13:00', '17:15'], { workedMinutes: 525, status: 'overtime' }),
  day(2, '2026-11-05', ['09:12', '15:00'], { workedMinutes: 318, late: true, status: 'late' }),
];

describe('TimeRecords', () => {
  let component: TimeRecords;
  let fixture: ComponentFixture<TimeRecords>;
  let http: HttpTestingController;

  const attendanceRequest = (): TestRequest => http.expectOne(request => request.url === '/api/v1/attendance');

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimeRecords],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(TimeRecords);
    component = fixture.componentInstance;
    http = TestBed.inject(HttpTestingController);
    http.expectOne('/api/v1/employees').flush(employees);
    attendanceRequest().flush(days);
    await fixture.whenStable();
  });

  afterEach(() => http.verify());

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  // Botão 0 é "Nova marcação"; os seguintes são o "Editar" de cada linha.
  async function openModal(index = 0): Promise<HTMLButtonElement> {
    const trigger = fixture.nativeElement.querySelectorAll('.button-edit')[index] as HTMLButtonElement;
    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    return trigger;
  }

  function type(selector: string, value: string): void {
    const input = fixture.nativeElement.querySelector(selector) as HTMLInputElement;
    input.value = value;
    input.dispatchEvent(new Event('input'));
  }

  it('focuses the first editable field after rendering', async () => {
    await openModal();
    expect(document.activeElement?.id).toBe('edit-employee');
  });

  it('skips read-only fields when editing an existing day', async () => {
    await openModal(1);
    const element = fixture.nativeElement as HTMLElement;
    expect(element.querySelector<HTMLInputElement>('#edit-employee')!.readOnly).toBe(true);
    expect(element.querySelector<HTMLInputElement>('#edit-date')!.readOnly).toBe(true);
    expect(document.activeElement?.id).toBe('edit-entry');
  });

  it('wraps Tab at both ends and preserves native keys inside', async () => {
    await openModal();
    const dialog = fixture.nativeElement.querySelector('[role="dialog"]') as HTMLElement;
    const first = dialog.querySelector('button')!;
    const last = dialog.querySelector('.button-secondary') as HTMLButtonElement;
    const press = (key: string, shiftKey = false) => {
      const event = new KeyboardEvent('keydown', { key, shiftKey, bubbles: true, cancelable: true });
      document.activeElement!.dispatchEvent(event);
      return event;
    };

    last.focus();
    expect(press('Tab').defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(first);
    expect(press('Tab', true).defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(last);

    dialog.querySelector('input')!.focus();
    expect(press('Tab').defaultPrevented).toBe(false);
    expect(press('Tab', true).defaultPrevented).toBe(false);
    expect(press('Enter').defaultPrevented).toBe(false);
  });

  it('restores the original trigger on Cancel, Close and Escape', async () => {
    for (const action of ['.button-secondary', '.close-button', 'Escape']) {
      const trigger = await openModal(1);
      if (action === 'Escape') {
        document.activeElement!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
      } else {
        (fixture.nativeElement.querySelector(action) as HTMLButtonElement).click();
      }
      fixture.detectChanges();
      expect(fixture.nativeElement.querySelector('[role="dialog"]')).toBeNull();
      expect(document.activeElement).toBe(trigger);
    }
  });

  it('renders one row per employee and day, with the date column', () => {
    const rows = fixture.nativeElement.querySelectorAll('tbody tr');
    expect(rows.length).toBe(3);
    expect(rows[0].textContent).toContain('06/11/2026');
    expect(rows[1].textContent).toContain('08:45');
    expect(rows[1].textContent).toContain('Hora extra');
    expect(rows[2].textContent).toContain('Atraso');
  });

  it('sends the filters to the API and clears them using the button', async () => {
    const element = fixture.nativeElement as HTMLElement;
    const employee = element.querySelector<HTMLSelectElement>('#employee-filter')!;
    employee.selectedIndex = 1;
    employee.dispatchEvent(new Event('change'));
    expect(attendanceRequest().request.params.get('employeeId')).toBe('1');

    type('#date-filter', '2026-11-04');
    const filtered = attendanceRequest();
    expect(filtered.request.params.get('employeeId')).toBe('1');
    expect(filtered.request.params.get('date')).toBe('2026-11-04');
    filtered.flush([]);
    await fixture.whenStable();
    expect(element.querySelector('tbody')?.textContent).toContain('Nenhum registro');

    element.querySelector<HTMLButtonElement>('.button-clear')!.click();
    const cleared = attendanceRequest();
    expect(cleared.request.params.keys()).toEqual([]);
    cleared.flush(days);
    await fixture.whenStable();
    expect(component.filterEmployeeId()).toBeNull();
    expect(component.filterDate()).toBe('');
    expect(element.querySelectorAll('tbody tr').length).toBe(3);
  });

  it('loads the selected day and keeps the break punches', async () => {
    await openModal(2);
    expect(component.editingDay()).toMatchObject({
      mode: 'edit', employeeId: 3, date: '2026-11-05', entry: '08:30', exit: '17:15', middle: ['12:00', '13:00'],
    });
    fixture.nativeElement.querySelector('.button-secondary').click();
    await fixture.whenStable();
    expect(component.editingDay()).toBeNull();
  });

  it('saves the edited day through the API and reloads the list', async () => {
    await openModal(2);
    type('#edit-exit', '18:00');
    fixture.nativeElement.querySelector('.button-primary').click();

    const save = http.expectOne({ method: 'PUT', url: '/api/v1/attendance/3/2026-11-05' });
    expect(save.request.body).toEqual({ times: ['08:30', '12:00', '13:00', '18:00'] });
    save.flush(days[1]);
    attendanceRequest().flush(days);
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('[role="dialog"]')).toBeNull();
  });

  it('rejects invalid dates, malformed times and non-increasing times', async () => {
    await openModal(1);
    const original = { ...component.editingDay()! };
    for (const changes of [
      { date: '' }, { date: '2026-02-30' }, { date: '0000-01-01' },
      { entry: '' }, { entry: '24:00' }, { exit: 'noon' },
      { exit: '08:00' }, { exit: '07:59' },
    ]) {
      component.editingDay.set({ ...original, ...changes });
      component.saveDay();
      expect(component.editingDay()).not.toBeNull();
      expect(Object.keys(component.editErrors).length).toBeGreaterThan(0);
    }
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('#exit-error')).not.toBeNull();
  });

  it('allows clearing the optional exit', async () => {
    await openModal(3);
    type('#edit-exit', '');
    fixture.nativeElement.querySelector('.button-primary').click();
    const save = http.expectOne({ method: 'PUT', url: '/api/v1/attendance/2/2026-11-05' });
    expect(save.request.body).toEqual({ times: ['09:12'] });
    save.flush(days[2]);
    attendanceRequest().flush(days);
  });

  it('adds new punches to the existing ones when creating', async () => {
    await openModal(0);
    component.editingDay.update(form => ({ ...form!, employeeId: 1, date: '2026-11-06', entry: '12:00', exit: '13:00' }));
    component.saveDay();

    const existing = attendanceRequest();
    expect(existing.request.params.get('employeeId')).toBe('1');
    existing.flush([days[0]]);
    const save = http.expectOne({ method: 'PUT', url: '/api/v1/attendance/1/2026-11-06' });
    expect(save.request.body).toEqual({ times: ['08:00', '12:00', '13:00'] });
    save.flush(days[0]);
    attendanceRequest().flush(days);
  });

  it('requires an employee when creating', async () => {
    await openModal(0);
    component.editingDay.update(form => ({ ...form!, entry: '08:00' }));
    component.saveDay();
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('#employee-error')).not.toBeNull();
  });

  it('builds the report links from the current filters', () => {
    component.setEmployeeFilter(2);
    attendanceRequest().flush([]);
    expect(component.csvUrl()).toBe('/api/v1/reports/attendance.csv?employeeId=2');
    expect(component.pdfUrl()).toBe('/api/v1/reports/attendance.pdf?employeeId=2');
  });
});
