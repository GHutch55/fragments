import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useFileTree } from "@/hooks/useFileTree";
import {
  Folder,
  File,
  ChevronRight,
  ChevronDown,
  Star,
  FolderPlus,
  MoreVertical,
  Trash2,
  Edit,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSnippets } from "@/hooks/useSnippets";
import { useFolders } from "@/hooks/useFolders";
import type { Folder as FolderType, Snippet } from "@/api/types";

interface FileTreeProps {
  onCreateFolder?: () => void;
  onCreateSnippet?: () => void;
}

export function FileTree({ onCreateFolder, onCreateSnippet }: FileTreeProps) {
  const navigate = useNavigate();
  const { folders, snippets, loading, error } = useFileTree();
  const { deleteSnippet } = useSnippets();
  const { deleteFolder, updateFolder } = useFolders();
  const [expandedFolders, setExpandedFolders] = useState<Set<number>>(
    new Set(),
  );
  const [renamingFolder, setRenamingFolder] = useState<number | null>(null);
  const [renameValue, setRenameValue] = useState("");

  const toggleFolder = (folderId: number) => {
    const newExpanded = new Set(expandedFolders);
    if (newExpanded.has(folderId)) {
      newExpanded.delete(folderId);
    } else {
      newExpanded.add(folderId);
    }
    setExpandedFolders(newExpanded);
  };

  const getChildFolders = (parentId: number | null) => {
    if (!Array.isArray(folders)) return [];
    const result = folders.filter((f) => {
      // Check if parent_id matches - handle null, undefined, and actual values
      if (parentId === null) {
        return f.parent_id === null || f.parent_id === undefined;
      }
      return f.parent_id === parentId;
    });
    console.log(`getChildFolders(${parentId}):`, result);
    return result;
  };

  const getSnippetsInFolder = (folderId: number | null) => {
    if (!Array.isArray(snippets)) return [];
    const result = snippets.filter((s) => {
      // Check if folder_id matches - handle null, undefined, and actual values
      if (folderId === null) {
        return s.folder_id === null || s.folder_id === undefined;
      }
      return s.folder_id === folderId;
    });
    console.log(`getSnippetsInFolder(${folderId}):`, result);
    return result;
  };

  const renderSnippet = (snippet: Snippet, level: number) => {
    const handleSnippetClick = () => {
      navigate(`/editor/${snippet.id}`);
    };

    return (
      <div
        key={snippet.id}
        className="flex items-center gap-2 px-3 py-2 hover:bg-accent rounded-md cursor-pointer transition-colors group"
        style={{ paddingLeft: `${level * 20 + 12}px` }}
        onClick={handleSnippetClick}
      >
        <div className="w-4 shrink-0" />
        <File className="h-4 w-4 text-muted-foreground shrink-0" />
        <span className="text-sm flex-1 truncate">{snippet.title}</span>
        {snippet.is_favorite && (
          <Star className="h-3 w-3 fill-yellow-500 text-yellow-500 shrink-0" />
        )}
        <span className="text-xs text-muted-foreground shrink-0">
          {snippet.language}
        </span>

        <DropdownMenu>
          <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={async (e) => {
                e.stopPropagation();
                if (confirm(`Delete "${snippet.title}"?`)) {
                  await deleteSnippet(snippet.id);
                }
              }}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    );
  };

  const renderFolder = (folder: FolderType, level: number) => {
    const isExpanded = expandedFolders.has(folder.id);
    const isRenaming = renamingFolder === folder.id;
    const childFolders = getChildFolders(folder.id);
    const folderSnippets = getSnippetsInFolder(folder.id);
    const hasChildren = childFolders.length > 0 || folderSnippets.length > 0;

    const handleRename = async () => {
      if (renameValue.trim() && renameValue !== folder.name) {
        await updateFolder(folder.id, { name: renameValue.trim() });
      }
      setRenamingFolder(null);
      setRenameValue("");
    };

    return (
      <div key={folder.id}>
        <div
          className="flex items-center gap-2 px-3 py-2 hover:bg-accent rounded-md cursor-pointer transition-colors group"
          style={{ paddingLeft: `${level * 20 + 12}px` }}
          onClick={() => !isRenaming && toggleFolder(folder.id)}
        >
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
            )
          ) : (
            <div className="w-4 shrink-0" />
          )}
          <Folder className="h-4 w-4 text-blue-500 shrink-0" />

          {isRenaming ? (
            <Input
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onBlur={handleRename}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRename();
                if (e.key === "Escape") {
                  setRenamingFolder(null);
                  setRenameValue("");
                }
              }}
              className="h-6 text-sm flex-1"
              autoFocus
              onClick={(e) => e.stopPropagation()}
            />
          ) : (
            <span className="text-sm font-medium flex-1 truncate">
              {folder.name}
            </span>
          )}

          <span className="text-xs text-muted-foreground shrink-0">
            {folderSnippets.length}
          </span>

          <DropdownMenu>
            <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={(e) => {
                  e.stopPropagation();
                  setRenamingFolder(folder.id);
                  setRenameValue(folder.name);
                }}
              >
                <Edit className="h-4 w-4 mr-2" />
                Rename
              </DropdownMenuItem>
              <DropdownMenuItem
                className="text-destructive focus:text-destructive"
                onClick={async (e) => {
                  e.stopPropagation();
                  if (childFolders.length > 0) {
                    alert(
                      "Cannot delete folder with subfolders. Delete subfolders first.",
                    );
                    return;
                  }
                  if (
                    confirm(
                      `Delete "${folder.name}"? Any snippets inside will be moved to the root.`,
                    )
                  ) {
                    try {
                      await deleteFolder(folder.id);
                    } catch (err) {
                      console.error("Delete folder error:", err);
                      alert(
                        `Failed to delete folder: ${err instanceof Error ? err.message : "Unknown error"}`,
                      );
                    }
                  }
                }}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {isExpanded && (
          <div>
            {childFolders.map((child) => renderFolder(child, level + 1))}
            {folderSnippets.map((snippet) => renderSnippet(snippet, level + 1))}
          </div>
        )}
      </div>
    );
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading your snippets...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="text-center">
          <div className="bg-destructive/10 rounded-full p-4 inline-block mb-4">
            <File className="h-8 w-8 text-destructive" />
          </div>
          <h3 className="text-lg font-semibold mb-2">Error loading data</h3>
          <p className="text-destructive text-sm">{error}</p>
        </div>
      </div>
    );
  }

  const rootFolders = getChildFolders(null);
  const rootSnippets = getSnippetsInFolder(null);

  // DEBUG
  console.log("FileTree render:", {
    folders: folders.map((f) => ({
      id: f.id,
      name: f.name,
      parent_id: f.parent_id,
    })),
    snippets: snippets.map((s) => ({
      id: s.id,
      title: s.title,
      folder_id: s.folder_id,
    })),
    rootFolders,
    rootSnippets,
    foldersIsArray: Array.isArray(folders),
    snippetsIsArray: Array.isArray(snippets),
  });

  if (rootFolders.length === 0 && rootSnippets.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="bg-primary/10 rounded-full p-8 mb-6">
          <File className="h-16 w-16 text-primary" />
        </div>
        <h3 className="text-2xl font-bold mb-3">No snippets yet</h3>
        <p className="text-muted-foreground max-w-md mb-8 text-base">
          Get started by creating your first code snippet. Organize your code
          with folders and keep everything at your fingertips.
        </p>
        <div className="flex gap-3">
          <Button
            variant="outline"
            className="gap-2"
            onClick={() => onCreateFolder?.()}
          >
            <FolderPlus className="h-4 w-4" />
            Create Folder
          </Button>
          <Button className="gap-2" onClick={() => onCreateSnippet?.()}>
            <File className="h-4 w-4" />
            New Snippet
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto">
      <div className="space-y-1 pb-4">
        {rootFolders.map((folder) => renderFolder(folder, 0))}
        {rootSnippets.map((snippet) => renderSnippet(snippet, 0))}
      </div>
    </div>
  );
}
