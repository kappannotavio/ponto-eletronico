import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Attendance } from '../../core/models';
import { TimeRecords } from './time-records';

const days: Attendance[] = [{
  employeeId: 1, employeeName: 'Pessoa 1', date: '2026-11-06', entry: '08:00', exit: '17:00',
  workedMinutes: 480, late: false, status: 'normal', times: ['08:00', '17:00'], records: [],
}];

describe('TimeRecords', () => {
  let component: TimeRecords;
  let fixture: ComponentFixture<TimeRecords>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimeRecords],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(TimeRecords);
    component = fixture.componentInstance;
    const http = TestBed.inject(HttpTestingController);
    http.expectOne('/api/v1/employees').flush([]);
    http.expectOne(request => request.url === '/api/v1/attendance').flush(days);
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  async function openModal(index = 0): Promise<HTMLButtonElement> {
    const trigger = fixture.nativeElement.querySelectorAll('.button-edit')[index] as HTMLButtonElement;
    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    return trigger;
  }

  it('focuses the first editable field after rendering', async () => {
    await openModal();
    expect(document.activeElement?.id).toBe('edit-employee');
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

});
