import { useFileTree } from "./useFileTree";
import { foldersAPI } from "@/api/folders";
import type { CreateFolderInput, UpdateFolderInput } from "@/api/types";

export function useFolders() {
  const { folders, setFolders } = useFileTree();

  const createFolder = async (data: CreateFolderInput) => {
    const created = await foldersAPI.create(data);
    setFolders((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      return [...current, created];
    });
    return created;
  };

  const updateFolder = async (id: number, data: UpdateFolderInput) => {
    const updated = await foldersAPI.update(id, data);
    setFolders((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      return current.map((f) => (f.id === id ? updated : f));
    });
    return updated;
  };

  const deleteFolder = async (id: number) => {
    await foldersAPI.delete(id);
    setFolders((prev) => {
      const current = Array.isArray(prev) ? prev : [];
      return current.filter((f) => f.id !== id);
    });
  };

  return {
    folders: Array.isArray(folders) ? folders : [],
    createFolder,
    updateFolder,
    deleteFolder,
  };
}
