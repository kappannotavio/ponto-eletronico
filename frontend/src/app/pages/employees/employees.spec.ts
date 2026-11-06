import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Employee } from '../../core/models';
import { Employees } from './employees';

const employees: Employee[] = [1, 2].map(id => ({
  id, fullName: `Pessoa ${id}`, jobTitle: 'Cargo', whatsapp: '', startTime: '08:00', dailyHours: 8,
  lunchBreak: 60, cardTag: `TAG-${id}`, isActive: true, createdAt: '', updatedAt: '',
}));

describe('Employees', () => {
  let component: Employees;
  let fixture: ComponentFixture<Employees>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [Employees],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(Employees);
    component = fixture.componentInstance;
    TestBed.inject(HttpTestingController).expectOne('/api/v1/employees').flush(employees);
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
    expect(document.activeElement?.id).toBe('edit-full-name');
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
