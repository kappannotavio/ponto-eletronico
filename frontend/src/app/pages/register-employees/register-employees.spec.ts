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
});
