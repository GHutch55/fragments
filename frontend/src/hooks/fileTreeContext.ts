import { createContext } from "react";
import type { Folder, Snippet } from "@/api/types";

export interface FileTreeContextType {
  folders: Folder[];
  snippets: Snippet[];
  loading: boolean;
  error: string | null;
  setFolders: React.Dispatch<React.SetStateAction<Folder[]>>;
  setSnippets: React.Dispatch<React.SetStateAction<Snippet[]>>;
  reset: () => void;
  loadAll: () => Promise<void>;
}

export const FileTreeContext = createContext<FileTreeContextType | null>(null);
