export interface User {
  id: number;
  username: string;
  created_at: string;
  updated_at: string;
}

export type PublicUser = Omit<User, "password_hash">;

export interface AuthResponse {
  token: string;
  user: PublicUser;
}

export interface Folder {
  id: number;
  user_id: number;
  name: string;
  description?: string | null;
  parent_id?: number | null;
  created_at: string;
}

export interface Snippet {
  id: number;
  user_id: number;
  folder_id?: number | null;
  title: string;
  description?: string | null;
  content: string;
  language: string;
  is_favorite: boolean;
  created_at: string;
  updated_at: string;
}

export interface Tag {
  id: number;
  user_id: number;
  name: string;
  color?: string | null;
  created_at: string;
}

export interface SnippetTag {
  snippet_id: number;
  tag_id: number;
}

export interface Paginated<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
}

export interface ApiSuccess {
  success: true;
}

// Auth
export interface RegisterInput {
  username: string;
  password: string;
  display_name?: string | null;
}

export interface LoginInput {
  username: string;
  password: string;
}

export interface ChangePasswordInput {
  current_password: string;
  new_password: string;
}

// Users
export interface UpdateUserInput {
  username?: string;
}

// Folders
export interface CreateFolderInput {
  name: string;
  description?: string;
  parent_id?: number | null;
}

export interface UpdateFolderInput {
  name?: string;
  description?: string;
  parent_id?: number | null;
}

// Snippets
export interface CreateSnippetInput {
  title: string;
  content: string;
  language: string;
  folder_id?: number | null;
}

export interface UpdateSnippetInput {
  title?: string;
  content?: string;
  language?: string;
  is_favorite?: boolean;
}

// Tags
export interface CreateTagInput {
  name: string;
  color?: string | null;
}

export interface UpdateTagInput {
  name?: string;
  color?: string | null;
}
