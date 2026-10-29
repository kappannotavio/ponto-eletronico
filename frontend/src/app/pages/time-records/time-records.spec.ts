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
});
