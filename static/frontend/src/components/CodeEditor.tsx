import React, { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Save, FileText, Loader2, Maximize, Minimize } from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { useTheme } from "./ThemeProvider";
import Editor from "react-simple-code-editor";
import Prism from 'prismjs';

// Import Prism languages - import in dependency order
// NOTE: We Don't import prism-core separately, it's included in prismjs
import 'prismjs/components/prism-clike';
import 'prismjs/components/prism-markup';
import 'prismjs/components/prism-css';
import 'prismjs/components/prism-javascript';
import 'prismjs/components/prism-json';
import 'prismjs/components/prism-bash';
import 'prismjs/components/prism-c';
import 'prismjs/components/prism-cpp';
import 'prismjs/components/prism-go';
import 'prismjs/components/prism-java';
import 'prismjs/components/prism-php';
import 'prismjs/components/prism-python';
import 'prismjs/components/prism-ruby';
import 'prismjs/components/prism-rust';
import 'prismjs/components/prism-sql';
import 'prismjs/components/prism-yaml';
import 'prismjs/components/prism-markdown';
// JSX/TSX must come after JavaScript (they extend it)
import 'prismjs/components/prism-jsx';
import 'prismjs/components/prism-typescript';
import 'prismjs/components/prism-tsx';

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
  const [isFullscreen, setIsFullscreen] = useState(false);
  const editorRef = useRef<HTMLDivElement>(null);
  const { theme } = useTheme();

  // Determine the current effective theme
  const currentTheme = theme === "system" 
    ? (typeof window !== 'undefined' && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
    : theme;

  useEffect(() => {
    setFileName(initialFileName);
    setContent(initialContent);
    setHasChanges(false);
  }, [initialFileName, initialContent]);

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, []);

  const toggleFullscreen = () => {
    const editor = editorRef.current;
    if (!editor) return;

    if (!document.fullscreenElement) {
      editor.requestFullscreen().catch((err) => {
        console.error('Error attempting to enable fullscreen:', err);
      });
    } else {
      document.exitFullscreen().catch((err) => {
        console.error('Error attempting to exit fullscreen:', err);
      });
    }
  };

  const handleContentChange = (value: string) => {
    setContent(value);
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

  const highlightCode = (code: string): string => {
    if (!isCode || !code || !code.trim()) {
      return code;
    }
    
    // Map language names to Prism language identifiers
    const prismLanguageMap: { [key: string]: string } = {
      'javascript': 'javascript',
      'jsx': 'jsx',
      'typescript': 'typescript',
      'tsx': 'tsx',
      'python': 'python',
      'java': 'java',
      'go': 'go',
      'php': 'php',
      'ruby': 'ruby',
      'html': 'markup',
      'css': 'css',
      'scss': 'css',
      'json': 'json',
      'xml': 'markup',
      'yaml': 'yaml',
      'markdown': 'markdown',
      'bash': 'bash',
      'c': 'c',
      'cpp': 'cpp',
      'rust': 'rust',
      'sql': 'sql',
    };
    
    const prismLang = prismLanguageMap[language];
    
    if (!prismLang) {
      return code;
    }
    
    const prismLanguage = Prism.languages[prismLang];
    if (!prismLanguage || typeof prismLanguage !== 'object') {
      return code;
    }
    
    try {
      return Prism.highlight(code, prismLanguage, prismLang);
    } catch (e) {
      // If highlighting fails, return unhighlighted code
      console.warn('Prism highlighting error for language:', prismLang, e);
      return code;
    }
  };

  return (
    <div ref={editorRef} className="flex flex-col h-full bg-background">
      {/* Header */}
      <div className="flex items-center gap-4 px-6 py-4 border-b border-border bg-card">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="flex items-center justify-center w-10 h-10 bg-primary/10 rounded-lg border border-primary/20 shrink-0">
            <FileText className="w-5 h-5 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <Label htmlFor="fileName" className="text-xs text-muted-foreground font-mono uppercase mb-1.5 block">
              File Name
            </Label>
            <Input
              id="fileName"
              value={fileName}
              onChange={handleFileNameChange}
              placeholder="example.js"
              className="font-mono text-sm h-9"
            />
          </div>
          {isCode && (
            <Badge variant="secondary" className="font-mono text-xs shrink-0 h-fit">
              {language.toUpperCase()}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {hasChanges && (
            <Badge variant="outline" className="text-xs font-normal">
              Unsaved
            </Badge>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={toggleFullscreen}
            className="shrink-0"
          >
            {isFullscreen ? (
              <Minimize className="w-4 h-4" />
            ) : (
              <Maximize className="w-4 h-4" />
            )}
          </Button>
          <Button
            onClick={handleSave}
            disabled={isSaving || !hasChanges}
            size="sm"
            className="gap-2"
          >
            {isSaving ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="w-4 h-4" />
                Save
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Editor */}
      <div className="flex-1 overflow-auto relative bg-background">
        <div className="absolute inset-0">
          <Editor
            value={content}
            onValueChange={handleContentChange}
            highlight={highlightCode}
            placeholder="Start writing your code here..."
            padding={16}
            style={{
              fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
              fontSize: 14,
              lineHeight: 1.6,
              minHeight: '100%',
              outline: 'none',
              backgroundColor: 'transparent',
              color: 'hsl(var(--foreground))',
            }}
            textareaClassName="editor-textarea"
            preClassName="editor-pre"
          />
        </div>
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

