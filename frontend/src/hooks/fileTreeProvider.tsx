import { useEffect, useState } from "react";
import { foldersAPI } from "@/api/folders";
import { snippetsAPI } from "@/api/snippets";
import type { Folder, Snippet } from "@/api/types";
import { FileTreeContext } from "./fileTreeContext";

export function FileTreeProvider(props: { children: React.ReactNode }) {
  const [folders, setFolders] = useState<Folder[]>([]);
  const [snippets, setSnippets] = useState<Snippet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadAll() {
      try {
        setLoading(true);

        const [foldersData, snippetsResponse] = await Promise.all([
          foldersAPI.getAll(),
          snippetsAPI.getAll({ limit: "1000" }),
        ]);

        // Extract data from paginated response
        setFolders(Array.isArray(foldersData.data) ? foldersData.data : []);
        setSnippets(
          Array.isArray(snippetsResponse.data) ? snippetsResponse.data : [],
        );
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load data");
        setFolders([]);
        setSnippets([]);
      } finally {
        setLoading(false);
      }
    }
    loadAll();
  }, []);

  return (
    <FileTreeContext.Provider
      value={{
        folders,
        snippets,
        loading,
        error,
        setFolders,
        setSnippets,
      }}
    >
      {props.children}
    </FileTreeContext.Provider>
  );
}
