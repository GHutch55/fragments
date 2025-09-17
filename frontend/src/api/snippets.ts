import { apiRequest } from "./client";
import type {
  Snippet,
  Paginated,
  ApiSuccess,
  CreateSnippetInput,
  UpdateSnippetInput,
} from "./types";

export const snippetsAPI = {
  create: async (data: CreateSnippetInput): Promise<Snippet> => {
    return apiRequest<Snippet>("/snippets", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  getAll: async (
    params: { page?: string; limit?: string; search?: string } = {},
  ): Promise<Paginated<Snippet>> => {
    const searchParams = new URLSearchParams();

    if (params.page) searchParams.append("page", params.page);
    if (params.limit) searchParams.append("limit", params.limit);
    if (params.search) searchParams.append("search", params.search);

    const queryString = searchParams.toString();
    const endpoint = queryString ? `/snippets?${queryString}` : "/snippets";

    return apiRequest<Paginated<Snippet>>(endpoint);
  },

  getById: async (id: number): Promise<Snippet> => {
    return apiRequest<Snippet>(`/snippets/${id}`);
  },

  update: async (id: number, data: UpdateSnippetInput): Promise<Snippet> => {
    return apiRequest<Snippet>(`/snippets/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  },

  delete: async (id: number): Promise<ApiSuccess> => {
    return apiRequest<ApiSuccess>(`/snippets/${id}`, {
      method: "DELETE",
    });
  },
};
