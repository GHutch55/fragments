import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { FileTree } from "@/components/ui/filetree";
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
import { Plus, FolderPlus } from "lucide-react";
import { useFolders } from "@/hooks/useFolders";
import type { CreateFolderInput } from "@/api/types";

export function Dashboard() {
  const navigate = useNavigate();
  const { folders, createFolder } = useFolders();

  const [folderDialogOpen, setFolderDialogOpen] = useState(false);

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
          </div>
        </div>
      </header>

      {/* File Tree */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-6xl mx-auto p-6">
          <FileTree
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
