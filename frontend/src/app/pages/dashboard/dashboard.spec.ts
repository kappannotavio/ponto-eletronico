import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';

import { Attendance, Dashboard as DashboardData, Employee } from '../../core/models';
import { Dashboard } from './dashboard';

function employee(id: number, fullName: string): Employee {
  return {
    id, fullName, jobTitle: '', whatsapp: '', startTime: '08:00', dailyHours: 8, lunchBreak: 60,
    cardTag: `TAG-${id}`, isActive: true, createdAt: '', updatedAt: '',
  };
}

function day(employeeId: number, changes: Partial<Attendance>): Attendance {
  return {
    employeeId, employeeName: '', date: '2026-11-06', entry: null, exit: null, workedMinutes: 0,
    late: false, status: 'normal', times: [], records: [], ...changes,
  };
}

// Funcionários ativos (a API já filtra os inativos com ?active=true).
const employees = [employee(1, 'Ana Souza'), employee(3, 'Carla Mendes'), employee(4, 'Lucas Santos'), employee(5, 'Maria Oliveira')];

const today: Attendance[] = [
  day(1, { entry: '08:00', status: 'in_progress', times: ['08:00'] }),
  day(4, { entry: '08:15', exit: '17:00', workedMinutes: 465, late: true, status: 'late' }),
  day(5, { entry: '12:00', exit: '17:00', workedMinutes: 300, status: 'normal' }),
];

describe('Dashboard', () => {
  let component: Dashboard;
  let fixture: ComponentFixture<Dashboard>;
  let http: HttpTestingController;

  async function respond(days: Attendance[] = today, active: Employee[] = employees): Promise<void> {
    // O primeiro carregamento sai do timer(0), depois de um tick.
    await new Promise(resolve => setTimeout(resolve));
    const dashboard: DashboardData = {
      date: '2026-11-06', activeCount: active.length, presentCount: days.length,
      lateCount: days.filter(item => item.late).length, absentCount: active.length - days.length, today: days,
    };
    http.expectOne('/api/v1/dashboard').flush(dashboard);
    http.expectOne(request => request.url === '/api/v1/employees' && request.params.get('active') === 'true').flush(active);
    await fixture.whenStable();
  }

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Dashboard],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(Dashboard);
    component = fixture.componentInstance;
    http = TestBed.inject(HttpTestingController);
    await respond();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('counts active employees, with late employees also present', () => {
    expect(component.totals).toEqual({ present: 3, late: 1, missing: 1 });
    expect(component.rows.find(row => row.employeeId === 4)).toMatchObject({ present: true, late: true });
  });

  it('displays each situation, including employees without entries', () => {
    expect(component.rows.map(row => row.status)).toEqual(['Em andamento', 'Sem marcação', 'Atraso', 'Normal']);
    const table = fixture.nativeElement.querySelector('tbody').textContent;
    for (const status of ['Em andamento', 'Sem marcação', 'Atraso', 'Normal']) {
      expect(table).toContain(status);
    }
  });

  it('shows the worked hours calculated by the backend', () => {
    expect(component.rows.map(row => row.duration)).toEqual(['—', '—', '07:45', '05:00']);
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
  });

  it('handles a day without entries and shows an empty filter', async () => {
    fixture = TestBed.createComponent(Dashboard);
    component = fixture.componentInstance;
    await respond([]);
    expect(component.totals).toEqual({ present: 0, late: 0, missing: 4 });
    expect(component.rows.every(row => row.status === 'Sem marcação' && row.duration === '—')).toBe(true);
    fixture.nativeElement.querySelector('.indicador').click();
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('tbody').textContent).toContain('Nenhum funcionário');
  });

  it('shows the API error message', async () => {
    fixture = TestBed.createComponent(Dashboard);
    await new Promise(resolve => setTimeout(resolve));
    http.expectOne('/api/v1/dashboard').flush({ error: 'erro interno do servidor' }, { status: 500, statusText: 'Error' });
    http.match(request => request.url === '/api/v1/employees');
    await fixture.whenStable();
    expect(fixture.nativeElement.querySelector('[role="alert"]').textContent).toContain('erro interno do servidor');
  });
});
