import React, { useState, useEffect, useCallback } from "react";
import { foldersAPI } from "@/api/folders";
import { snippetsAPI } from "@/api/snippets";
import type { Folder, Snippet } from "@/api/types";
import { FileTreeContext } from "./fileTreeContext";

export function FileTreeProvider(props: { children: React.ReactNode }) {
  const [folders, setFolders] = useState<Folder[]>([]);
  const [snippets, setSnippets] = useState<Snippet[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Loads folders & snippets from API
  const loadAll = useCallback(async () => {
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
  }, []);

  // Load data once on mount
  useEffect(() => {
    loadAll();
  }, [loadAll]);

  // Resets context state to initial empty values
  const reset = useCallback(() => {
    setFolders([]);
    setSnippets([]);
    setError(null);
    setLoading(false);
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
        reset,
        loadAll,
      }}
    >
      {props.children}
    </FileTreeContext.Provider>
  );
}
