import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Save, FileText, Loader2 } from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { useTheme } from "./ThemeProvider";

interface CodeEditorProps {
  initialFileName?: string;
  initialContent?: string;
  currentPath?: string;
  onSave?: (fileName: string, content: string) => void;
  onClose?: () => void;
}

// Language mapping for syntax highlighting
function getLanguageFromExtension(ext: string): string {
  const languageMap: { [key: string]: string } = {
    'js': 'javascript',
    'jsx': 'jsx',
    'ts': 'typescript',
    'tsx': 'tsx',
    'py': 'python',
    'java': 'java',
    'go': 'go',
    'php': 'php',
    'rb': 'ruby',
    'html': 'html',
    'css': 'css',
    'scss': 'scss',
    'sass': 'scss',
    'json': 'json',
    'xml': 'xml',
    'yml': 'yaml',
    'yaml': 'yaml',
    'md': 'markdown',
    'sh': 'bash',
    'bash': 'bash',
    'c': 'c',
    'cpp': 'cpp',
    'cc': 'cpp',
    'cxx': 'cpp',
    'h': 'c',
    'hpp': 'cpp',
    'rs': 'rust',
    'vue': 'vue',
    'swift': 'swift',
    'kt': 'kotlin',
    'kts': 'kotlin',
    'sql': 'sql',
    'dockerfile': 'dockerfile',
  };

  return languageMap[ext] || 'text';
}

export function CodeEditor({
  initialFileName = "",
  initialContent = "",
  currentPath = ".",
  onSave,
  onClose,
}: CodeEditorProps) {
  const [fileName, setFileName] = useState(initialFileName);
  const [content, setContent] = useState(initialContent);
  const [isSaving, setIsSaving] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { theme } = useTheme();

  useEffect(() => {
    setFileName(initialFileName);
    setContent(initialContent);
    setHasChanges(false);
  }, [initialFileName, initialContent]);

  useEffect(() => {
    // Auto-resize textarea
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
    }
  }, [content]);

  const handleContentChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setContent(e.target.value);
    setHasChanges(true);
  };

  const handleFileNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFileName(e.target.value);
    setHasChanges(true);
  };

  const handleSave = async () => {
    if (!fileName.trim()) {
      toast({
        title: "Error",
        description: "Please enter a file name",
        variant: "destructive",
      });
      return;
    }

    setIsSaving(true);
    try {
      const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      
      const response = await fetch("/write", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          filePath,
          content,
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to save file");
      }

      toast({
        title: "Success",
        description: `File "${fileName}" saved successfully`,
      });

      setHasChanges(false);
      
      if (onSave) {
        onSave(fileName, content);
      }
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to save file",
        variant: "destructive",
      });
    } finally {
      setIsSaving(false);
    }
  };

  const fileExt = fileName.split(".").pop()?.toLowerCase() || "";
  const language = getLanguageFromExtension(fileExt);
  const isCode = language !== 'text';

  // Handle keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        handleSave();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="flex flex-col h-full bg-background">
      {/* Header */}
      <div className="flex items-center justify-between gap-4 p-4 border-b border-border bg-card">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="flex items-center justify-center w-8 h-8 bg-primary/10 rounded border border-primary/20">
            <FileText className="w-4 h-4 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <Label htmlFor="fileName" className="text-xs text-muted-foreground font-mono uppercase">
              File Name
            </Label>
            <Input
              id="fileName"
              value={fileName}
              onChange={handleFileNameChange}
              placeholder="example.js"
              className="font-mono text-sm mt-1"
            />
          </div>
          {isCode && (
            <Badge variant="secondary" className="font-mono text-xs shrink-0">
              {language.toUpperCase()}
            </Badge>
          )}
        </div>
        <Button
          onClick={handleSave}
          disabled={isSaving || !hasChanges}
          className="shrink-0"
        >
          {isSaving ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Saving...
            </>
          ) : (
            <>
              <Save className="w-4 h-4 mr-2" />
              Save
            </>
          )}
        </Button>
      </div>

      {/* Editor */}
      <div className="flex-1 overflow-hidden relative">
        <Textarea
          ref={textareaRef}
          value={content}
          onChange={handleContentChange}
          placeholder="Start writing your code here..."
          className="w-full h-full resize-none border-0 rounded-none font-mono text-sm leading-relaxed focus-visible:ring-0 focus-visible:ring-offset-0 p-4"
          style={{
            fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
          }}
        />
        {hasChanges && (
          <div className="absolute top-2 right-2">
            <Badge variant="outline" className="text-xs">
              Unsaved changes
            </Badge>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between px-4 py-2 border-t border-border bg-muted/30 text-xs text-muted-foreground font-mono">
        <div className="flex items-center gap-4">
          <span>Lines: {content.split('\n').length}</span>
          <span>Chars: {content.length}</span>
        </div>
        <div className="flex items-center gap-2">
          <kbd className="px-2 py-1 bg-background border border-border rounded text-xs">
            Ctrl+S
          </kbd>
          <span>to save</span>
        </div>
      </div>
    </div>
  );
}

