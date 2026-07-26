import { useState, useEffect, useRef } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  AlertCircle,
  Download,
  FileText,
  Loader2,
  Maximize,
  Minimize,
  Pencil,
  RotateCcw,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import { toast } from "@/hooks/use-toast";
import Prism from "prismjs";
import "prismjs/themes/prism-tomorrow.css";
import { getFileIcon } from "@/lib/utils";

import { EnhancedVideoPlayer } from "./EnhancedVideoPlayer";
import { EnhancedAudioPlayer } from "./EnhancedAudioPlayer";
import { CodeEditorDialog } from "./CodeEditorDialog";

interface FilePreviewProps {
  fileName: string;
  isOpen: boolean;
  onClose: () => void;
  currentPath?: string;
}

export function FilePreview({
  fileName,
  isOpen,
  onClose,
  currentPath = ".",
}: FilePreviewProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [zoom, setZoom] = useState(100);
  const [fileContent, setFileContent] = useState<string>("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);

  const fileExt = fileName.split(".").pop()?.toLowerCase() || "";

  const isImage = ["jpg", "jpeg", "png", "gif", "webp", "svg", "bmp"].includes(
    fileExt,
  );
  const isPdf = fileExt === "pdf";
  const isVideo = ["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(
    fileExt,
  );
  const isAudio = ["mp3", "wav", "ogg", "flac", "aac"].includes(fileExt);
  const isText = [
    "txt",
    "md",
    "json",
    "xml",
    "csv",
    "log",
    "env",
    "ini",
    "cfg",
    "conf",
    "toml",
  ].includes(fileExt);
  const isCode = [
    "js",
    "ts",
    "tsx",
    "jsx",
    "py",
    "java",
    "go",
    "php",
    "rb",
    "html",
    "css",
    "scss",
    "sass",
    "yml",
    "yaml",
    "sh",
    "bash",
    "c",
    "cpp",
    "cc",
    "cxx",
    "h",
    "hpp",
    "rs",
    "vue",
    "swift",
    "kt",
    "kts",
    "sql",
    "graphql",
    "gql",
    "r",
    "lua",
    "perl",
    "pl",
    "tf",
    "hcl",
    "zig",
    "nim",
    "ex",
    "exs",
    "erl",
    "clj",
    "scala",
    "groovy",
    "dart",
  ].includes(fileExt);
  const isConfig =
    [
      "gitignore",
      "dockerignore",
      "editorconfig",
      "prettierrc",
      "eslintrc",
      "babelrc",
      "npmrc",
      "yarnrc",
    ].includes(fileExt) ||
    [
      "Dockerfile",
      "Makefile",
      "Caddyfile",
      "Vagrantfile",
      "Procfile",
      "Gemfile",
      "Rakefile",
      ".gitignore",
      ".dockerignore",
      ".editorconfig",
      ".prettierrc",
      ".eslintrc",
      ".babelrc",
      ".npmrc",
      ".yarnrc",
      "docker-compose.yml",
      "docker-compose.yaml",
    ].includes(fileName);

  const isTextLike = isText || isCode || isConfig;
  const language = getLanguageFromExtension(fileExt);

  useEffect(() => {
    if (isCode && fileContent) {
      setTimeout(() => {
        Prism.highlightAll();
      }, 0);
    }
  }, [fileContent, isCode]);

  useEffect(() => {
    if (isOpen) {
      setLoading(true);
      setError(null);
      setZoom(100);
      setIsFullscreen(false);
    }
  }, [isOpen, fileName]);

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => {
      document.removeEventListener("fullscreenchange", handleFullscreenChange);
    };
  }, []);

  const toggleFullscreen = () => {
    const dialog = dialogRef.current?.closest('[role="dialog"]') as HTMLElement;
    if (!dialog) return;

    if (!document.fullscreenElement) {
      dialog.requestFullscreen().catch((err) => {
        console.error("Error attempting to enable fullscreen:", err);
      });
    } else {
      document.exitFullscreen().catch((err) => {
        console.error("Error attempting to exit fullscreen:", err);
      });
    }
  };

  const handleDownload = () => {
    try {
      const link = document.createElement("a");
      const filePath =
        currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      link.href = `/download?file=${encodeURIComponent(filePath)}`;
      link.download = fileName;
      link.style.display = "none";
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);

      toast({
        title: "Download Started",
        description: `${fileName} download initiated.`,
      });
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to download file",
        variant: "destructive",
      });
    }
  };

  const typeLabel = isImage
    ? "Image"
    : isPdf
      ? "PDF"
      : isVideo
        ? "Video"
        : isAudio
          ? "Audio"
          : isCode
            ? "Code"
            : isText
              ? "Text"
              : isConfig
                ? "Config"
                : fileExt
                  ? fileExt.toUpperCase()
                  : "File";

  const renderPreviewContent = () => {
    const filePath =
      currentPath === "." ? fileName : `${currentPath}/${fileName}`;
    const previewUrl = `/files?path=${encodeURIComponent(filePath)}`;

    if (isImage) {
      return (
        <div className="flex flex-col items-center gap-4 animate-fade-in">
          <div className="flex items-center gap-1 bg-secondary/60 rounded-md border border-border p-1 shadow-subtle">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setZoom(Math.max(25, zoom - 25))}
              disabled={zoom <= 25}
              className="h-8 w-8 p-0"
              aria-label="Zoom out"
            >
              <ZoomOut className="w-4 h-4" />
            </Button>
            <Badge
              variant="secondary"
              className="font-mono text-xs min-w-[3.5rem] text-center justify-center"
            >
              {zoom}%
            </Badge>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setZoom(Math.min(200, zoom + 25))}
              disabled={zoom >= 200}
              className="h-8 w-8 p-0"
              aria-label="Zoom in"
            >
              <ZoomIn className="w-4 h-4" />
            </Button>
            <div className="w-px h-5 bg-border mx-1" />
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setZoom(100)}
              disabled={zoom === 100}
              className="h-8 w-8 p-0"
              aria-label="Reset zoom"
            >
              <RotateCcw className="w-3.5 h-3.5" />
            </Button>
          </div>
          <div className="max-h-[65vh] overflow-auto rounded-lg border border-border bg-[repeating-conic-gradient(hsl(var(--muted))_0%_25%,transparent_0%_50%)] bg-[length:16px_16px] p-4 scrollbar-thin">
            <img
              src={previewUrl}
              alt={fileName}
              style={{ transform: `scale(${zoom / 100})` }}
              className="max-w-full h-auto transition-transform origin-center rounded shadow-medium"
              onLoad={() => setLoading(false)}
              onError={() => {
                setError("Failed to load image");
                setLoading(false);
              }}
            />
          </div>
        </div>
      );
    }

    if (isPdf) {
      return (
        <div className="flex flex-col gap-3 animate-fade-in">
          <div className="flex items-center justify-between gap-2 px-3 py-2 rounded-md bg-secondary/50 border border-border text-xs font-mono text-muted-foreground">
            <span className="truncate">{fileName}</span>
            <Badge variant="outline" className="font-mono text-[0.65rem]">
              PDF
            </Badge>
          </div>
          <iframe
            src={previewUrl}
            className="w-full h-[70vh] border border-border rounded-lg shadow-subtle"
            onLoad={() => setLoading(false)}
            onError={() => {
              setError("Failed to load PDF");
              setLoading(false);
            }}
            title={`PDF preview of ${fileName}`}
          />
        </div>
      );
    }

    if (isVideo) {
      return (
        <EnhancedVideoPlayer
          src={previewUrl}
          onLoadedData={() => setLoading(false)}
          onError={() => {
            setError("Failed to load video");
            setLoading(false);
          }}
        />
      );
    }

    if (isAudio) {
      return (
        <EnhancedAudioPlayer
          src={previewUrl}
          fileName={fileName}
          onLoadedData={() => setLoading(false)}
          onError={() => {
            setError("Failed to load audio");
            setLoading(false);
          }}
        />
      );
    }

    if (isTextLike) {
      return (
        <TextFilePreview
          key={refreshKey}
          fileName={fileName}
          currentPath={currentPath}
          onLoad={() => setLoading(false)}
          onError={(err) => {
            setError(err);
            setLoading(false);
          }}
          onContentLoaded={(content) => setFileContent(content)}
        />
      );
    }

    // Unsupported file type
    setTimeout(() => setLoading(false), 0);
    return (
      <div className="flex flex-col items-center text-center py-16 px-4 animate-fade-in">
        <div className="w-20 h-20 mx-auto mb-5 rounded-2xl bg-secondary/70 border border-border flex items-center justify-center shadow-subtle">
          <FileText className="w-9 h-9 text-muted-foreground" />
        </div>
        <h3 className="font-mono font-bold text-foreground mb-1.5 tracking-tight">
          Preview not available
        </h3>
        <p className="text-muted-foreground font-mono text-sm mb-6 max-w-sm">
          This file type can't be rendered in the browser. Download it to view
          its contents.
        </p>
        <Button onClick={handleDownload} variant="default" className="gap-2">
          <Download className="w-4 h-4" />
          Download File
        </Button>
      </div>
    );
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent
        ref={dialogRef}
        className="w-[calc(100%-2rem)] max-w-5xl max-h-[85vh] bg-card border-2 border-border overflow-y-auto [&>button]:hidden sm:max-h-[90vh]"
      >
        <DialogHeader className="border-b border-border pb-3 sm:pb-4">
          <div className="flex items-center justify-between gap-3">
            <DialogTitle className="font-mono font-bold text-foreground truncate flex items-center gap-2.5 text-sm sm:text-base min-w-0">
              <span className="flex items-center justify-center w-8 h-8 rounded-md bg-secondary/70 border border-border shrink-0">
                {getFileIcon(fileName, "w-4 h-4 text-primary")}
              </span>
              <span className="truncate">{fileName}</span>
              <Badge
                variant="outline"
                className="font-mono text-[0.65rem] uppercase tracking-wide shrink-0 hidden sm:inline-flex"
              >
                {typeLabel}
              </Badge>
            </DialogTitle>
            <div className="flex items-center gap-1 flex-shrink-0">
              <Button
                variant="outline"
                size="sm"
                onClick={toggleFullscreen}
                className="shrink-0 hidden sm:flex gap-2"
              >
                {isFullscreen ? (
                  <>
                    <Minimize className="w-4 h-4" />
                    Exit
                  </>
                ) : (
                  <>
                    <Maximize className="w-4 h-4" />
                    Fullscreen
                  </>
                )}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={toggleFullscreen}
                className="shrink-0 sm:hidden p-2"
                aria-label={
                  isFullscreen
                    ? "Exit fullscreen mode"
                    : "Toggle fullscreen mode"
                }
              >
                {isFullscreen ? (
                  <Minimize className="w-4 h-4" />
                ) : (
                  <Maximize className="w-4 h-4" />
                )}
              </Button>
              {isTextLike && (
                <CodeEditorDialog
                  currentPath={currentPath}
                  initialFileName={fileName}
                  initialContent={fileContent}
                  mode="edit"
                  onSaveSuccess={() => {
                    setRefreshKey((prev) => prev + 1);
                    setLoading(true);
                    setError(null);
                    setFileContent("");
                  }}
                  triggerButton={
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        className="shrink-0 hidden sm:flex gap-2"
                      >
                        <Pencil className="w-4 h-4" />
                        Edit
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="shrink-0 sm:hidden p-2"
                        aria-label="Edit file"
                      >
                        <Pencil className="w-4 h-4" />
                      </Button>
                    </>
                  }
                />
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={handleDownload}
                className="shrink-0 hidden sm:flex gap-2"
              >
                <Download className="w-4 h-4" />
                Download
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleDownload}
                className="shrink-0 sm:hidden p-2"
                aria-label="Download file"
              >
                <Download className="w-4 h-4" />
              </Button>
            </div>
          </div>
        </DialogHeader>

        <div className="mt-4 overflow-auto max-h-[calc(90vh-8rem)] sm:max-h-[calc(95vh-8rem)] scrollbar-thin">
          {loading && !error && (
            <div className="flex flex-col items-center justify-center py-16 animate-fade-in">
              <Loader2 className="w-10 h-10 animate-spin text-primary" />
              <p className="font-mono text-xs text-muted-foreground mt-4 uppercase tracking-widest">
                Loading preview
              </p>
            </div>
          )}

          {error && (
            <div className="flex flex-col items-center text-center py-16 px-4 animate-fade-in">
              <div className="w-20 h-20 mx-auto mb-5 rounded-2xl bg-destructive/10 border border-destructive/30 flex items-center justify-center">
                <AlertCircle className="w-10 h-10 text-destructive" />
              </div>
              <h3 className="font-mono font-bold text-foreground mb-1.5 tracking-tight text-lg">
                Preview error
              </h3>
              <p className="text-muted-foreground font-mono text-sm mb-6 max-w-md mx-auto">
                {error}
              </p>
              <Button
                onClick={handleDownload}
                variant="default"
                className="gap-2"
              >
                <Download className="w-4 h-4" />
                Download File
              </Button>
            </div>
          )}

          {!error && (
            <div
              className={loading ? "" : "animate-fade-in"}
              style={{ display: loading ? "none" : "block" }}
            >
              {renderPreviewContent()}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

// Language mapping for syntax highlighting
function getLanguageFromExtension(ext: string): string {
  const languageMap: { [key: string]: string } = {
    js: "javascript",
    jsx: "jsx",
    ts: "typescript",
    tsx: "tsx",
    py: "python",
    java: "java",
    go: "go",
    php: "php",
    rb: "ruby",
    html: "html",
    css: "css",
    scss: "scss",
    sass: "scss",
    json: "json",
    xml: "xml",
    yml: "yaml",
    yaml: "yaml",
    md: "markdown",
    sh: "bash",
    bash: "bash",
    c: "c",
    cpp: "cpp",
    cc: "cpp",
    cxx: "cpp",
    h: "c",
    hpp: "cpp",
    rs: "rust",
    swift: "swift",
    kt: "kotlin",
    kts: "kotlin",
    sql: "sql",
    dockerfile: "dockerfile",
    toml: "toml",
    ini: "ini",
    graphql: "graphql",
    gql: "graphql",
    r: "r",
    lua: "lua",
    perl: "perl",
    pl: "perl",
    tf: "hcl",
    hcl: "hcl",
    zig: "zig",
    dart: "dart",
    scala: "scala",
    groovy: "groovy",
    ex: "elixir",
    exs: "elixir",
    erl: "erlang",
    clj: "clojure",
    nim: "nim",
    vue: "markup",
    gitignore: "git",
    dockerignore: "docker",
    env: "bash",
    cfg: "ini",
    conf: "ini",
  };

  return languageMap[ext] || "text";
}

// Component for text file previews
function TextFilePreview({
  fileName,
  currentPath = ".",
  onLoad,
  onError,
  onContentLoaded,
}: {
  fileName: string;
  currentPath?: string;
  onLoad: () => void;
  onError: (error: string) => void;
  onContentLoaded?: (content: string) => void;
}) {
  const [content, setContent] = useState<string>("");
  const [contentError, setContentError] = useState<boolean>(false);
  const [loadingContent, setLoadingContent] = useState<boolean>(true);

  useEffect(() => {
    let cancelled = false;
    const fetchContent = async () => {
      try {
        const filePath =
          currentPath === "." ? fileName : `${currentPath}/${fileName}`;
        const response = await fetch(
          `/files?path=${encodeURIComponent(filePath)}`,
        );
        if (!response.ok) {
          throw new Error("Failed to fetch file content");
        }
        const text = await response.text();
        if (cancelled) return;
        setContent(text);
        setLoadingContent(false);
        if (onContentLoaded) {
          onContentLoaded(text);
        }
        onLoad();
      } catch (error) {
        if (cancelled) return;
        setContentError(true);
        setLoadingContent(false);
        onError("Failed to load text file");
      }
    };

    fetchContent();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fileName, currentPath]);

  const fileExt = fileName.split(".").pop()?.toLowerCase() || "";
  const isCode = [
    "js",
    "ts",
    "tsx",
    "jsx",
    "py",
    "java",
    "go",
    "php",
    "rb",
    "html",
    "css",
    "scss",
    "sass",
    "json",
    "xml",
    "yml",
    "yaml",
    "md",
    "sh",
    "bash",
    "c",
    "cpp",
    "cc",
    "cxx",
    "h",
    "hpp",
    "rs",
    "vue",
    "swift",
    "kt",
    "kts",
    "sql",
    "dockerfile",
    "toml",
    "ini",
    "graphql",
    "gql",
    "r",
    "lua",
    "perl",
    "pl",
    "tf",
    "hcl",
    "zig",
    "dart",
    "scala",
    "groovy",
    "ex",
    "exs",
    "erl",
    "clj",
    "nim",
    "gitignore",
    "dockerignore",
    "editorconfig",
    "env",
    "cfg",
    "conf",
  ].includes(fileExt);

  const fileIcon = getFileIcon(fileName, "w-4 h-4 text-primary");
  const language = getLanguageFromExtension(fileExt);

  const lineCount =
    content && !loadingContent && !contentError
      ? content.split("\n").length
      : 0;

  return (
    <div className="rounded-lg border border-border bg-secondary/20 overflow-hidden animate-fade-in">
      <div className="flex items-center justify-between gap-2 px-4 py-2 bg-secondary/50 border-b border-border">
        <div className="flex items-center gap-2 min-w-0">
          <span className="flex items-center justify-center w-5 h-5 shrink-0">
            {fileIcon}
          </span>
          <span className="font-mono text-xs text-muted-foreground truncate">
            {fileName}
          </span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {isCode && (
            <Badge variant="secondary" className="font-mono text-[0.65rem]">
              {language.toUpperCase()}
            </Badge>
          )}
          {!loadingContent && !contentError && (
            <Badge
              variant="outline"
              className="font-mono text-[0.65rem] text-muted-foreground"
            >
              {lineCount} {lineCount === 1 ? "line" : "lines"}
            </Badge>
          )}
        </div>
      </div>
      {loadingContent ? (
        <div className="p-4 space-y-2 bg-background">
          {[...Array(8)].map((_, i) => (
            <div
              key={i}
              className="h-3 rounded bg-muted animate-pulse"
              style={{ width: `${85 - (i % 5) * 12}%` }}
            />
          ))}
        </div>
      ) : contentError ? (
        <div className="p-4 bg-background text-sm font-mono text-destructive">
          Failed to load file content.
        </div>
      ) : (
        <div className="bg-background">
          <div className="max-h-[55vh] overflow-auto scrollbar-thin">
            <pre
              className={`text-sm font-mono p-4 whitespace-pre-wrap break-words leading-relaxed ${isCode ? "language-" + language : ""}`}
            >
              <code className={isCode ? `language-${language}` : undefined}>
                {content}
              </code>
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}