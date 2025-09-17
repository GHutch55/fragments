import { apiRequest } from "./client";
import type { PublicUser, ApiSuccess, UpdateUserInput } from "./types";

export const userAPI = {
  getCurrentUser: async (): Promise<PublicUser> => {
    return apiRequest<PublicUser>("/users/me");
  },

  updateCurrentUser: async (data: UpdateUserInput): Promise<PublicUser> => {
    return apiRequest<PublicUser>("/users/me", {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  deleteCurrentUser: async (): Promise<ApiSuccess> => {
    return apiRequest<ApiSuccess>("/users/me", {
      method: "DELETE",
    });
  },
};
