import { apiRequest } from "./client";
import type {
  Folder,
  ApiSuccess,
  CreateFolderInput,
  UpdateFolderInput,
} from "./types";

export const foldersAPI = {
  create: async (data: CreateFolderInput): Promise<Folder> => {
    return apiRequest<Folder>("/folders", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  getAll: async (): Promise<Folder[]> => {
    return apiRequest<Folder[]>("/folders");
  },

  getById: async (id: number): Promise<Folder> => {
    return apiRequest<Folder>(`/folders/${id}`);
  },

  update: async (id: number, data: UpdateFolderInput): Promise<Folder> => {
    return apiRequest<Folder>(`/folders/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  delete: async (id: number): Promise<ApiSuccess> => {
    return apiRequest<ApiSuccess>(`/folders/${id}`, {
      method: "DELETE",
    });
  },
};
