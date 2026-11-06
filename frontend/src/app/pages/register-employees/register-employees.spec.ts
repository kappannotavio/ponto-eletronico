import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { RegisterEmployees } from './register-employees';

describe('RegisterEmployees', () => {
  let component: RegisterEmployees;
  let fixture: ComponentFixture<RegisterEmployees>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RegisterEmployees],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(RegisterEmployees);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
