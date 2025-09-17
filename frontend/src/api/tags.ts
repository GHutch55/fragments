import { apiRequest } from "./client";
import type { Tag, ApiSuccess, CreateTagInput, UpdateTagInput } from "./types";

export const tagsAPI = {
  create: async (data: CreateTagInput): Promise<Tag> => {
    return apiRequest<Tag>("/tags", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  getAll: async (): Promise<Tag[]> => {
    return apiRequest<Tag[]>("/tags");
  },

  getById: async (id: number): Promise<Tag> => {
    return apiRequest<Tag>(`/tags/${id}`);
  },

  update: async (id: number, data: UpdateTagInput): Promise<Tag> => {
    return apiRequest<Tag>(`/tags/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  delete: async (id: number): Promise<ApiSuccess> => {
    return apiRequest<ApiSuccess>(`/tags/${id}`, {
      method: "DELETE",
    });
  },
};
