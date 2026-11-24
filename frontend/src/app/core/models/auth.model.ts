export type CEFRCode = 'A1' | 'A2' | 'B1' | 'B2' | 'C1' | 'C2';
export type UserRole = 'student' | 'teacher' | 'admin';

export interface User {
  id: string;
  email: string;
  full_name: string;
  role: UserRole;
  cefr_level: CEFRCode;
  created_at: string;
  updated_at: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  full_name: string;
  cefr_level?: CEFRCode;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface AuthResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: User;
}

export interface ApiErrorResponse {
  error: string;
}
