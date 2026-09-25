import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { LoginRequest, LoginResponse, ProvisionRequest, ProvisionResult, User } from './auth.models';

@Injectable({ providedIn: 'root' })
export class AuthApi {
  private readonly http = inject(HttpClient);

  login(body: LoginRequest) {
    return this.http.post<LoginResponse>('/auth/login', body);
  }

  logout() {
    return this.http.post<void>('/auth/logout', {});
  }

  me() {
    return this.http.get<User>('/auth/me');
  }

  provision(body: ProvisionRequest) {
    return this.http.post<ProvisionResult>('/auth/users/batch', body);
  }
}
