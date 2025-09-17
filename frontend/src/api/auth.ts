import { apiRequest } from "./client";
import type {
  AuthResponse,
  ApiSuccess,
  PublicUser,
  RegisterInput,
  LoginInput,
  ChangePasswordInput,
} from "./types";

export const authAPI = {
  register: async (data: RegisterInput): Promise<AuthResponse> => {
    return apiRequest<AuthResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  login: async (data: LoginInput): Promise<AuthResponse> => {
    return apiRequest<AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  getMe: async (): Promise<PublicUser> => {
    return apiRequest<PublicUser>("/auth/me");
  },

  changePassword: async (data: ChangePasswordInput): Promise<ApiSuccess> => {
    return apiRequest<ApiSuccess>("/auth/change-password", {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },
};

export const authHelpers = {
  setToken: (token: string): void => {
    localStorage.setItem("authToken", token);
  },

  removeToken: (): void => {
    localStorage.removeItem("authToken");
  },

  isAuthenticated: (): boolean => {
    return !!localStorage.getItem("authToken");
  },

  logout: (): void => {
    localStorage.removeItem("authToken");
    window.location.href = "/login";
  },
};
