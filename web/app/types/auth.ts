export type UserRole = 'admin' | 'teacher' | 'student';

export interface User {
  id: string;
  login: string;
  role: UserRole;
  full_name: string;
}

export interface LoginRequest {
  login: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface ProvisionUserInput {
  full_name: string;
  login: string;
}

export interface ProvisionRequest {
  group_id: string;
  users: ProvisionUserInput[];
}

export interface ProvisionedUser {
  id: string;
  full_name: string;
  login: string;
  password: string;
}

export interface ProvisionFailure {
  index: number;
  login: string;
  error: string;
}

export interface ProvisionResult {
  created: ProvisionedUser[];
  failed: ProvisionFailure[];
}
