import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import Editor from "@monaco-editor/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Save, X, Loader2, Plus as PlusIcon, XCircle } from "lucide-react";
import { useSnippets } from "@/hooks/useSnippets";
import { useFolders } from "@/hooks/useFolders";
import { snippetsAPI } from "@/api/snippets";
import type { Snippet } from "@/api/types";

export function EditorPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { createSnippet, updateSnippet } = useSnippets();
  const { folders } = useFolders();

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [content, setContent] = useState("// Start coding...");
  const [language, setLanguage] = useState("javascript");
  const [folderId, setFolderId] = useState<number | null>(null);
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState("");

  const addTag = () => {
    const trimmed = tagInput.trim();
    if (trimmed && !tags.includes(trimmed)) {
      setTags([...tags, trimmed]);
      setTagInput("");
    }
  };

  const removeTag = (tagToRemove: string) => {
    setTags(tags.filter((tag) => tag !== tagToRemove));
  };

  // Load snippet if editing
  useEffect(() => {
    if (id) {
      setLoading(true);
      snippetsAPI
        .getById(parseInt(id))
        .then((snippet: Snippet) => {
          setTitle(snippet.title);
          setDescription(snippet.description ?? "");
          setContent(snippet.content);
          setLanguage(snippet.language);
          setFolderId(snippet.folder_id ?? null);
          setTags(snippet.tags ?? []);
        })
        .catch((err) => {
          setError(
            err instanceof Error ? err.message : "Failed to load snippet",
          );
        })
        .finally(() => setLoading(false));
    }
  }, [id]);

  const handleSave = async () => {
    if (!title.trim()) {
      setError("Title is required");
      return;
    }

    if (!content.trim()) {
      setError("Content is required");
      return;
    }

    setSaving(true);
    setError(null);

    try {
      if (id) {
        // Update existing snippet (including folder and tags)
        await updateSnippet(parseInt(id), {
          title,
          description: description || undefined,
          content,
          language,
          folder_id: folderId,
          tags: tags.length > 0 ? tags : undefined,
        });
      } else {
        // Create new snippet
        await createSnippet({
          title,
          description: description || undefined,
          content,
          language,
          folder_id: folderId,
          tags: tags.length > 0 ? tags : undefined,
        });
      }
      navigate("/dashboard");
    } catch (err) {
      console.error("Save error:", err);
      const errorMessage =
        err instanceof Error ? err.message : "Failed to save snippet";
      setError(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-background">
      {/* Header */}
      <header className="border-b bg-card">
        <div className="flex flex-col gap-4 px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4 flex-1">
              <div className="flex-1 max-w-2xl space-y-2">
                <Input
                  placeholder="Snippet title..."
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="text-lg font-semibold"
                />
                <Input
                  placeholder="Description (optional)..."
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="text-sm"
                />

                {/* Tags */}
                <div className="flex flex-wrap gap-2 items-center">
                  {tags.map((tag) => (
                    <span
                      key={tag}
                      className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-primary/10 text-primary rounded-md"
                    >
                      {tag}
                      <button
                        onClick={() => removeTag(tag)}
                        className="hover:text-primary/80"
                      >
                        <XCircle className="h-3 w-3" />
                      </button>
                    </span>
                  ))}
                  <div className="flex items-center gap-1">
                    <Input
                      placeholder="Add tag..."
                      value={tagInput}
                      onChange={(e) => setTagInput(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          addTag();
                        }
                      }}
                      className="text-xs w-32 h-7"
                    />
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={addTag}
                      className="h-7 w-7 p-0"
                    >
                      <PlusIcon className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <Label htmlFor="language" className="text-sm">
                  Language:
                </Label>
                <Select value={language} onValueChange={setLanguage}>
                  <SelectTrigger id="language" className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="javascript">JavaScript</SelectItem>
                    <SelectItem value="typescript">TypeScript</SelectItem>
                    <SelectItem value="python">Python</SelectItem>
                    <SelectItem value="java">Java</SelectItem>
                    <SelectItem value="cpp">C++</SelectItem>
                    <SelectItem value="go">Go</SelectItem>
                    <SelectItem value="rust">Rust</SelectItem>
                    <SelectItem value="html">HTML</SelectItem>
                    <SelectItem value="css">CSS</SelectItem>
                    <SelectItem value="sql">SQL</SelectItem>
                    <SelectItem value="json">JSON</SelectItem>
                    <SelectItem value="markdown">Markdown</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center gap-2">
                <Label htmlFor="folder" className="text-sm">
                  Folder:
                </Label>
                <Select
                  value={folderId?.toString() ?? "none"}
                  onValueChange={(value) =>
                    setFolderId(value === "none" ? null : parseInt(value))
                  }
                >
                  <SelectTrigger id="folder" className="w-40">
                    <SelectValue placeholder="No folder" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">No folder</SelectItem>
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

              <Button onClick={handleSave} disabled={saving} className="gap-2">
                {saving ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save className="h-4 w-4" />
                    Save
                  </>
                )}
              </Button>

              <Button
                variant="ghost"
                onClick={() => navigate("/dashboard")}
                className="gap-2"
              >
                <X className="h-4 w-4" />
                Cancel
              </Button>
            </div>
          </div>
        </div>

        {error && (
          <div className="px-6 pb-4">
            <p className="text-sm text-destructive">{error}</p>
          </div>
        )}
      </header>

      {/* Editor */}
      <main className="flex-1 overflow-hidden">
        <Editor
          height="100%"
          language={language}
          value={content}
          onChange={(value) => setContent(value || "")}
          theme="vs-dark"
          options={{
            fontSize: 14,
            minimap: { enabled: true },
            scrollBeyondLastLine: false,
            wordWrap: "on",
            automaticLayout: true,
          }}
        />
      </main>
    </div>
  );
}
