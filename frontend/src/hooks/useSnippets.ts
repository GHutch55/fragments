import { useFileTree } from "./useFileTree";
import { snippetsAPI } from "@/api/snippets";
import type { CreateSnippetInput, UpdateSnippetInput } from "@/api/types";

export function useSnippets() {
  const { snippets, setSnippets } = useFileTree();

  const createSnippet = async (data: CreateSnippetInput) => {
    const created = await snippetsAPI.create(data);
    setSnippets((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      return [created, ...current];
    });
    return created;
  };

  const updateSnippet = async (id: number, data: UpdateSnippetInput) => {
    const updated = await snippetsAPI.update(id, data);
    setSnippets((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      return current.map((s) => (s.id === id ? updated : s));
    });
    return updated;
  };

  const deleteSnippet = async (id: number) => {
    await snippetsAPI.delete(id);
    setSnippets((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      return current.filter((s) => s.id !== id);
    });
  };

  const toggleFavorite = async (id: number) => {
    const current = Array.isArray(snippets) ? snippets : [];
    const snippet = current.find((s) => s.id === id);
    if (!snippet) return;

    const updated = await snippetsAPI.update(id, {
      is_favorite: !snippet.is_favorite,
    });
    setSnippets((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      return current.map((s) => (s.id === id ? updated : s));
    });
    return updated;
  };

  return {
    snippets: Array.isArray(snippets) ? snippets : [],
    createSnippet,
    updateSnippet,
    deleteSnippet,
    toggleFavorite,
  };
}
