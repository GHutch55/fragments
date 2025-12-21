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

  // Get token from localStorage to trigger reload when it changes
  const authToken = localStorage.getItem("authToken");

  useEffect(() => {
    if (!authToken) {
      // If no token (logged out), reset state immediately
      setFolders([]);
      setSnippets([]);
      setError(null);
      setLoading(false);
      return;
    }

    async function loadAll() {
      try {
        setLoading(true);

        const [foldersData, snippetsResponse] = await Promise.all([
          foldersAPI.getAll(),
          snippetsAPI.getAll({ limit: "1000" }),
        ]);

        setFolders(Array.isArray(foldersData.data) ? foldersData.data : []);
        setSnippets(
          Array.isArray(snippetsResponse.data) ? snippetsResponse.data : [],
        );
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load data");
        setFolders([]);
        setSnippets([]);
      } finally {
        setLoading(false);
      }
    }

    loadAll();
  }, [authToken]); // Re-run when authToken changes (login/logout)

  function reset() {
    setFolders([]);
    setSnippets([]);
    setError(null);
    setLoading(false);
  }

  return (
    <FileTreeContext.Provider
      value={{
        folders,
        snippets,
        loading,
        error,
        setFolders,
        setSnippets,
        reset,
      }}
    >
      {props.children}
    </FileTreeContext.Provider>
  );
}
