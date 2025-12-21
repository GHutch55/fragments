import { useState, useContext } from "react";
import { useNavigate } from "react-router-dom";
import { FileTree } from "@/components/ui/filetree";
import { FileTreeContext } from "@/hooks/fileTreeContext";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Plus, FolderPlus, LogOut, Search, X } from "lucide-react";
import { useFolders } from "@/hooks/useFolders";
import type { CreateFolderInput } from "@/api/types";

export function Dashboard() {
  const fileTree = useContext(FileTreeContext);
  const navigate = useNavigate();
  const { folders, createFolder } = useFolders();

  const [folderDialogOpen, setFolderDialogOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const handleLogout = () => {
    localStorage.removeItem("authToken");
    fileTree?.reset();
    navigate("/login");
  };

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [folderForm, setFolderForm] = useState<CreateFolderInput>({
    name: "",
    description: "",
    parent_id: null,
  });

  const handleCreateFolder = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      await createFolder(folderForm);
      setFolderDialogOpen(false);
      setFolderForm({
        name: "",
        description: "",
        parent_id: null,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create folder");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex h-screen flex-col bg-background">
      {/* Header */}
      <header className="border-b bg-card shadow-sm">
        <div className="flex items-center justify-between px-6 py-4">
          <div className="flex items-center gap-4">
            <h1 className="text-2xl font-semibold tracking-tight">Fragments</h1>

            {/* Search Bar */}
            <div className="relative w-96">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search snippets..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 pr-9"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery("")}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => setFolderDialogOpen(true)}
            >
              <FolderPlus className="h-4 w-4 mr-2" />
              New Folder
            </Button>
            <Button size="sm" onClick={() => navigate("/editor")}>
              <Plus className="h-4 w-4 mr-2" />
              New Snippet
            </Button>
            <Button size="sm" variant="ghost" onClick={handleLogout}>
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        </div>
      </header>

      {/* File Tree */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-6xl mx-auto p-6">
          <FileTree
            searchQuery={searchQuery}
            onCreateFolder={() => setFolderDialogOpen(true)}
            onCreateSnippet={() => navigate("/editor")}
          />
        </div>
      </main>

      {/* New Folder Dialog */}
      <Dialog open={folderDialogOpen} onOpenChange={setFolderDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FolderPlus className="h-5 w-5" />
              Create New Folder
            </DialogTitle>
            <DialogDescription>
              Organize your snippets with folders
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleCreateFolder}>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="folder-name">Folder Name *</Label>
                <Input
                  id="folder-name"
                  placeholder="My Project"
                  value={folderForm.name}
                  onChange={(e) =>
                    setFolderForm({ ...folderForm, name: e.target.value })
                  }
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="folder-description">
                  Description (Optional)
                </Label>
                <Textarea
                  id="folder-description"
                  placeholder="What's this folder for?"
                  value={folderForm.description}
                  onChange={(e) =>
                    setFolderForm({
                      ...folderForm,
                      description: e.target.value,
                    })
                  }
                  className="min-h-[80px]"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="parent-folder">Parent Folder (Optional)</Label>
                <Select
                  value={folderForm.parent_id?.toString() ?? "none"}
                  onValueChange={(value) =>
                    setFolderForm({
                      ...folderForm,
                      parent_id: value === "none" ? null : parseInt(value),
                    })
                  }
                >
                  <SelectTrigger id="parent-folder">
                    <SelectValue placeholder="No parent" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">No parent</SelectItem>
                    {folders &&
                      folders.length > 0 &&
                      folders.map((folder) => (
                        <SelectItem
                          key={folder.id}
                          value={folder.id.toString()}
                        >
                          {folder.name}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
              </div>

              {error && <p className="text-sm text-destructive">{error}</p>}
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setFolderDialogOpen(false)}
                disabled={loading}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={loading}>
                {loading ? "Creating..." : "Create Folder"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
