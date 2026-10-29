import { Routes } from '@angular/router';
import { Dashboard } from './pages/dashboard/dashboard';
import { RegisterEmployees } from './pages/register-employees/register-employees';
import { Employees } from './pages/employees/employees';
import { TimeRecords } from './pages/time-records/time-records';

export const routes: Routes = [
  {
    path: '',
    redirectTo: 'dashboard',
    pathMatch: 'full'
  },
  {
    path: 'dashboard',
    component: Dashboard
  },
  {
    path: 'employees/new',
    component: RegisterEmployees
  },
  {
    path: 'employees',
    component: Employees
  },
  {
    path: 'time-records',
    component: TimeRecords
  }
];
