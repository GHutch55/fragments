import { createContext } from "react";
import type { Folder, Snippet } from "@/api/types";

export interface FileTreeContextType {
  folders: Folder[];
  snippets: Snippet[];
  loading: boolean;
  error: string | null;
  setFolders: (value: Folder[] | ((prev: Folder[]) => Folder[])) => void;
  setSnippets: (value: Snippet[] | ((prev: Snippet[]) => Snippet[])) => void;
}

export const FileTreeContext = createContext<FileTreeContextType | null>(null);
