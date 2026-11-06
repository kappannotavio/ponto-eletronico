import { ComponentFixture, TestBed } from '@angular/core/testing';
import { TimeRecords } from './time-records';

describe('TimeRecords', () => {
  let component: TimeRecords;
  let fixture: ComponentFixture<TimeRecords>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [TimeRecords],
    }).compileComponents();

    fixture = TestBed.createComponent(TimeRecords);
    component = fixture.componentInstance;
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
    expect(document.activeElement?.id).toBe('edit-date');
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


  it('renders five records in descending date and entry order with simple durations', () => {
    expect(component.filteredRecords.map(record => record.id)).toEqual([1, 3, 2, 5, 4]);
    expect(fixture.nativeElement.querySelectorAll('tbody tr').length).toBe(5);
    expect(component.duration(component.records[0])).toBe('—');
    expect(component.duration(component.records[1])).toBe('08:45');
    expect(component.duration(component.records[2])).toBe('05:48');
  });

  it('combines filters and clears them using the button', async () => {
    const element = fixture.nativeElement as HTMLElement;
    const employee = element.querySelector<HTMLSelectElement>('#employee-filter')!;
    employee.selectedIndex = 1;
    employee.dispatchEvent(new Event('change'));
    await fixture.whenStable();
    expect(component.filteredRecords.map(record => record.id)).toEqual([1, 4]);
    const date = element.querySelector<HTMLInputElement>('#date-filter')!;
    date.value = '2026-11-04';
    date.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    expect(component.filteredRecords.map(record => record.id)).toEqual([4]);
    date.value = '2026-11-05';
    date.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    expect(element.querySelector('tbody')?.textContent).toContain('Nenhum registro');
    element.querySelector<HTMLButtonElement>('.button-clear')!.click();
    await fixture.whenStable();
    expect(component.employeeFilter).toBeNull();
    expect(component.dateFilter).toBe('');
    expect(element.querySelectorAll('tbody tr').length).toBe(5);
    component.dateFilter = '2026-11-05';
    expect(component.filteredRecords.map(record => record.id)).toEqual([3, 2]);
  });

  it('loads the selected record and discards its draft on cancel', async () => {
    const original = { ...component.filteredRecords[1] };
    await openModal(1);
    expect(component.editingRecord).toEqual(original);
    expect(component.editingRecord).not.toBe(component.records.find(record => record.id === original.id));
    component.editingRecord!.entry = '10:00';
    fixture.nativeElement.querySelector('.button-secondary').click();
    await fixture.whenStable();
    expect(component.editingRecord).toBeNull();
    expect(component.records.find(record => record.id === original.id)).toEqual(original);
  });

  it('saves only the selected record through the form and recalculates duration', async () => {
    const originals = component.records.map(record => ({ ...record }));
    const id = component.filteredRecords[2].id;
    await openModal(2);
    const element = fixture.nativeElement as HTMLElement;
    expect(element.querySelector<HTMLInputElement>('#edit-employee')!.readOnly).toBe(true);
    expect(element.querySelector<HTMLInputElement>('#edit-status')!.readOnly).toBe(true);
    for (const [field, value] of [['date', '2026-11-07'], ['entry', '10:00'], ['exit', '18:00']]) {
      const input = element.querySelector<HTMLInputElement>(`#edit-${field}`)!;
      input.value = value;
      input.dispatchEvent(new Event('input'));
    }
    element.querySelector<HTMLButtonElement>('.button-primary')!.click();
    await fixture.whenStable();
    const updated = component.records.find(record => record.id === id)!;
    expect(updated).toEqual({ ...originals.find(record => record.id === id), date: '2026-11-07', entry: '10:00', exit: '18:00' });
    expect(component.records.filter(record => record.id !== id)).toEqual(originals.filter(record => record.id !== id));
    expect(component.duration(updated)).toBe('08:00');
    expect(component.filteredRecords[0].id).toBe(id);
    expect(element.querySelector('[role="dialog"]')).toBeNull();
  });

  it('rejects invalid dates, malformed times and non-increasing same-day times', async () => {
    await openModal(0);
    const original = { ...component.records[0] };
    for (const changes of [
      { date: '' }, { date: '2026-02-30' }, { date: '0000-01-01' },
      { entry: '' }, { entry: '24:00' }, { exit: 'noon' },
      { exit: '08:00' }, { exit: '07:59' },
    ]) {
      component.editingRecord = { ...original, ...changes };
      component.saveRecord();
      expect(component.isEditModalOpen).toBe(true);
      expect(component.records[0]).toEqual(original);
      expect(Object.keys(component.editErrors).length).toBeGreaterThan(0);
    }
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('#exit-error')).not.toBeNull();
  });

  it('allows clearing the optional exit while retaining the demonstration status', async () => {
    await openModal(1);
    const id = component.editingRecord!.id;
    const status = component.editingRecord!.demonstrationStatus;
    component.editingRecord!.exit = '';
    component.saveRecord();
    const updated = component.records.find(record => record.id === id)!;
    expect(updated.exit).toBeUndefined();
    expect(updated.demonstrationStatus).toBe(status);
    expect(component.duration(updated)).toBe('—');
  });

  it('focuses the clear button when saving removes a filtered row', async () => {
    component.dateFilter = '2026-11-06';
    fixture.detectChanges();
    await openModal();
    component.editingRecord!.date = '2026-11-07';
    fixture.nativeElement.querySelector('.button-primary').click();
    await fixture.whenStable();
    expect(component.filteredRecords).toEqual([]);
    expect(document.activeElement).toBe(fixture.nativeElement.querySelector('.button-clear'));
  });
});
