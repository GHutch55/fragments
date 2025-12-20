const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1";

const getAuthToken = (): string | null => {
  return localStorage.getItem("authToken");
};

export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`;
  const token = getAuthToken();

  const config: RequestInit = {
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
    ...options,
  };

  if (token) {
    (config.headers as Record<string, string>).Authorization =
      `Bearer ${token}`;
  }

  const response = await fetch(url, config);

  if (response.status === 204) {
    return { success: true } as T;
  }

  const data = (await response.json()) as T;

  if (!response.ok) {
    throw new Error(
      (data as { message?: string; error?: string }).message ||
      (data as { message?: string; error?: string }).error ||
      "Request failed",
    );
  }

  return data;
}
