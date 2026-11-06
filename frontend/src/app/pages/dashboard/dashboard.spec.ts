import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Dashboard } from './dashboard';

describe('Dashboard', () => {
  let component: Dashboard;
  let fixture: ComponentFixture<Dashboard>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Dashboard],
    }).compileComponents();

    fixture = TestBed.createComponent(Dashboard);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('counts active employees, with late employees also present', () => {
    expect(component.totals).toEqual({ present: 3, late: 1, missing: 1 });
    expect(component.rows.find(row => row.employeeId === 4)).toMatchObject({ present: true, late: true });
  });

  it('excludes inactive employees both with and without an entry', () => {
    component.records.push({ id: 20, employeeId: 2, date: component.consideredDate, entry: '09:15' });
    expect(component.totals).toEqual({ present: 3, late: 1, missing: 1 });
    expect(component.rows.some(row => row.employeeId === 2)).toBe(false);
    component.employees.find(employee => employee.id === 1)!.isActive = false;
    expect(component.totals).toEqual({ present: 2, late: 1, missing: 1 });
  });

  it('uses only valid entries on the considered day and counts employees once', () => {
    component.records.push(
      { id: 21, employeeId: 3, date: '2026-11-05', entry: '08:30' },
      { id: 22, employeeId: 3, date: component.consideredDate, entry: '' },
      { id: 23, employeeId: 1, date: component.consideredDate, entry: '09:00' },
    );
    expect(component.totals).toEqual({ present: 3, late: 1, missing: 1 });
    expect(component.rows.find(row => row.employeeId === 1)?.entry).toBe('08:00');
  });

  it('uses zero tolerance without classifying missing entries as late', () => {
    component.records[0].entry = '08:01';
    expect(component.totals.late).toBe(2);
    component.records[0].entry = '07:59';
    expect(component.totals.late).toBe(1);
    expect(component.rows.find(row => row.employeeId === 3)?.late).toBe(false);
  });

  it('displays each situation with lateness taking precedence over completion', () => {
    expect(component.rows.map(row => row.status)).toEqual(['Em andamento', 'Sem marcação', 'Atraso', 'Registrado']);
    const table = fixture.nativeElement.querySelector('tbody').textContent;
    for (const status of ['Em andamento', 'Sem marcação', 'Atraso', 'Registrado']) {
      expect(table).toContain(status);
    }
  });

  it('toggles each card filter and preserves totals', async () => {
    const element = fixture.nativeElement as HTMLElement;
    const cards = element.querySelectorAll<HTMLButtonElement>('.indicador');
    for (const [index, count] of [3, 1, 1].entries()) {
      cards[index].click();
      await fixture.whenStable();
      expect(element.querySelectorAll('tbody tr').length).toBe(count);
      expect(cards[index].getAttribute('aria-pressed')).toBe('true');
      expect(cards[index].classList.contains('selected')).toBe(true);
      expect(Array.from(element.querySelectorAll('.indicator-value'), item => item.textContent?.trim())).toEqual(['3', '1', '1']);
      cards[index].click();
      await fixture.whenStable();
      expect(component.selectedFilter).toBeNull();
      expect(element.querySelectorAll('tbody tr').length).toBe(4);
    }
    cards[1].click();
    await fixture.whenStable();
    expect(element.querySelector('tbody')?.textContent).toContain('Lucas Santos');
    cards[1].click();
    await fixture.whenStable();
    expect(component.selectedFilter).toBeNull();
    expect(element.querySelectorAll('tbody tr').length).toBe(4);
  });

  it('handles no entries without inferring absence or lateness and shows an empty filter', async () => {
    component.records.splice(0);
    expect(component.totals).toEqual({ present: 0, late: 0, missing: 4 });
    expect(component.rows.every(row => row.status === 'Sem marcação' && row.duration === '—')).toBe(true);
    fixture.nativeElement.querySelector('.indicador').click();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('tbody').textContent).toContain('Nenhum funcionário');
    component.employees.splice(0);
    expect(component.totals).toEqual({ present: 0, late: 0, missing: 0 });
  });

  it('calculates gross same-day durations without subtracting breaks or assuming overnight work', () => {
    expect(component.duration(component.records[1])).toBe('08:45');
    expect(component.duration(component.records[2])).toBe('05:00');
    expect(component.duration(component.records[0])).toBe('—');
    expect(component.duration()).toBe('—');
    const record = component.records[1];
    expect(component.duration({ ...record, entry: '22:00', exit: '06:00' })).toBe('—');
    expect(component.duration({ ...record, exit: '25:00' })).toBe('—');
    expect(component.duration({ ...record, entry: '08:00', exit: '08:00' })).toBe('00:00');
  });
});
