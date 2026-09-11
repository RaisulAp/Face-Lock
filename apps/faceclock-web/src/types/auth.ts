export interface UserRole {
  id: string;
  name: string;
  is_system?: boolean;
}

export interface AuthUser {
  id: string;
  email: string;
  employee_id?: string | null;
  employee_number?: string | null;
  full_name?: string;
  must_change_password?: boolean;
  roles: UserRole[];
  permissions: string[];
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: AuthUser;
}

export interface RefreshResponse {
  access_token: string;
  refresh_token: string;
}

export interface AuthMeResponse {
  user: AuthUser;
}
