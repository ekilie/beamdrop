import React, { useState, useEffect, useRef } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  X,
  Download,
  ZoomIn,
  ZoomOut,
  FileText,
  Edit,
  Maximize,
  Minimize,
} from "lucide-react";
import { toast } from "@/hooks/use-toast";
import Prism from 'prismjs';
import 'prismjs/themes/prism-tomorrow.css';
import { getFileIcon } from "@/lib/utils";
import { useTheme } from "./ThemeProvider";
import { EnhancedVideoPlayer } from "./EnhancedVideoPlayer";
import { EnhancedAudioPlayer } from "./EnhancedAudioPlayer";
import { CodeEditorDialog } from "./CodeEditorDialog";

interface FilePreviewProps {
  fileName: string;
  isOpen: boolean;
  onClose: () => void;
  currentPath?: string;
}

export function FilePreview({ fileName, isOpen, onClose, currentPath = "." }: FilePreviewProps) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [zoom, setZoom] = useState(100);
  const [fileContent, setFileContent] = useState<string>("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const dialogRef = useRef<HTMLDivElement>(null);

  const fileExt = fileName.split(".").pop()?.toLowerCase() || "";

  const isImage = ["jpg", "jpeg", "png", "gif", "webp", "svg", "bmp"].includes(fileExt);
  const isPdf = fileExt === "pdf";
  const isVideo = ["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(fileExt);
  const isAudio = ["mp3", "wav", "ogg", "flac", "aac"].includes(fileExt);
  const isText = ["txt", "md", "json", "xml", "csv", "log"].includes(fileExt);
  const isCode = ["js", "ts", "tsx", "jsx", "py", "java", "vint", "go", "php", "rb", "html", "css", "scss"].includes(fileExt);

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

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, []);

  const toggleFullscreen = () => {
    const dialog = dialogRef.current?.closest('[role="dialog"]') as HTMLElement;
    if (!dialog) return;

    if (!document.fullscreenElement) {
      dialog.requestFullscreen().catch((err) => {
        console.error('Error attempting to enable fullscreen:', err);
      });
    } else {
      document.exitFullscreen().catch((err) => {
        console.error('Error attempting to exit fullscreen:', err);
      });
    }
  };

  const handleDownload = () => {
    try {
      const link = document.createElement('a');
      const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      link.href = `/download?file=${encodeURIComponent(filePath)}`;
      link.download = fileName;
      link.style.display = 'none';
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

  const renderPreviewContent = () => {
    const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
    const previewUrl = `/files?path=${encodeURIComponent(filePath)}`;

    if (isImage) {
      return (
        <div className="flex flex-col items-center space-y-4 animate-fade-in">
          <div className="flex items-center gap-2 bg-secondary/50 rounded-lg px-4 py-2 border border-border">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setZoom(Math.max(25, zoom - 25))}
              disabled={zoom <= 25}
              className="h-8"
            >
              <ZoomOut className="w-4 h-4" />
            </Button>
            <Badge variant="secondary" className="font-mono text-xs min-w-[4rem] text-center">
              {zoom}%
            </Badge>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setZoom(Math.min(200, zoom + 25))}
              disabled={zoom >= 200}
              className="h-8"
            >
              <ZoomIn className="w-4 h-4" />
            </Button>
          </div>
          <div className="max-h-[65vh] overflow-auto rounded-lg border border-border bg-muted/30 p-4 scrollbar-thin">
            <img
              src={previewUrl}
              alt={fileName}
              style={{ transform: `scale(${zoom / 100})` }}
              className="max-w-full h-auto transition-transform origin-center rounded"
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
        <div className="w-full h-[70vh] animate-fade-in">
          <iframe
            src={previewUrl}
            className="w-full h-full border-2 border-border rounded-lg shadow-lg"
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
          fileName={fileName}
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

    if (isText || isCode) {
      return (
        <div className="w-full">
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
        </div>
      );
    }

    // Unsupported file type
    setTimeout(() => setLoading(false), 0);
    return (
      <div className="text-center py-12">
        <div className="w-16 h-16 mx-auto mb-4 bg-muted rounded-full flex items-center justify-center">
          <FileText className="w-8 h-8 text-muted-foreground" />
        </div>
        <h3 className="font-mono font-bold text-foreground mb-2">
          PREVIEW NOT AVAILABLE
        </h3>
        <p className="text-muted-foreground font-mono text-sm mb-4">
          This file type cannot be previewed in the browser
        </p>
        <Button onClick={handleDownload} variant="default">
          <Download className="w-4 h-4 mr-2" />
          Download File
        </Button>
      </div>
    );
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent
        ref={dialogRef}
        className="w-[calc(100%-1rem)] max-w-5xl max-h-[90vh] bg-card border-2 border-border overflow-hidden [&>button]:hidden sm:max-h-[95vh]"
      >
        <DialogHeader className="border-b border-border pb-3 sm:pb-4">
          <div className="flex items-center justify-between gap-2">
            <DialogTitle className="font-mono font-bold text-foreground truncate flex items-center gap-2 text-sm sm:text-base">
              {getFileIcon(fileName, "w-4 h-4 sm:w-5 sm:h-5")}
              <span className="truncate">{fileName}</span>
            </DialogTitle>
            <div className="flex items-center gap-1 sm:gap-2 flex-shrink-0">
              <Button
                variant="outline"
                size="sm"
                onClick={toggleFullscreen}
                className="shrink-0 hidden sm:flex"
              >
                {isFullscreen ? (
                  <>
                    <Minimize className="w-4 h-4 mr-2" />
                    Exit
                  </>
                ) : (
                  <>
                    <Maximize className="w-4 h-4 mr-2" />
                    Fullscreen
                  </>
                )}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={toggleFullscreen}
                className="shrink-0 sm:hidden p-2"
                aria-label={isFullscreen ? "Exit fullscreen mode" : "Toggle fullscreen mode"}
              >
                {isFullscreen ? (
                  <Minimize className="w-4 h-4" />
                ) : (
                  <Maximize className="w-4 h-4" />
                )}
              </Button>
              {(isText || isCode) && (
                <CodeEditorDialog
                  currentPath={currentPath}
                  initialFileName={fileName}
                  initialContent={fileContent}
                  mode="edit"
                  onSaveSuccess={() => {
                    // Refresh the preview by updating the refresh key
                    setRefreshKey(prev => prev + 1);
                    setLoading(true);
                    setError(null);
                    setFileContent("");
                  }}
                  triggerButton={
                    <>
                      <Button
                        variant="outline"
                        size="sm"
                        className="shrink-0 hidden sm:flex"
                      >
                        <Edit className="w-4 h-4 mr-2" />
                        Edit
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="shrink-0 sm:hidden p-2"
                        aria-label="Edit file"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                    </>
                  }
                />
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={handleDownload}
                className="shrink-0 hidden sm:flex"
              >
                <Download className="w-4 h-4 mr-2" />
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
            <div className="flex items-center justify-center py-16">
              <div className="text-center space-y-4 animate-fade-in">
                <div className="relative">
                  <div className="w-12 h-12 border-4 border-primary/30 border-t-primary animate-spin rounded-full mx-auto"></div>
                  <div className="w-12 h-12 border-4 border-accent/30 border-b-accent animate-spin rounded-full mx-auto absolute top-0 left-1/2 -translate-x-1/2" style={{ animationDirection: 'reverse', animationDuration: '0.8s' }}></div>
                </div>
                <p className="font-mono text-sm text-muted-foreground font-semibold">
                  LOADING PREVIEW...
                </p>
              </div>
            </div>
          )}

          {error && (
            <div className="text-center py-16 animate-fade-in">
              <div className="w-20 h-20 mx-auto mb-4 bg-destructive/10 rounded-full flex items-center justify-center border-2 border-destructive/30 animate-bounce-slow">
                <X className="w-10 h-10 text-destructive" />
              </div>
              <h3 className="font-mono font-bold text-foreground mb-2 text-lg">
                PREVIEW ERROR
              </h3>
              <p className="text-muted-foreground font-mono text-sm mb-6 max-w-md mx-auto">
                {error}
              </p>
              <Button onClick={handleDownload} variant="default" className="hover-lift">
                <Download className="w-4 h-4 mr-2" />
                Download File
              </Button>
            </div>
          )}

          {!error && (
            <div style={{ display: loading ? 'none' : 'block' }}>
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

// Component for text file previews
function TextFilePreview({ fileName, currentPath = ".", onLoad, onError, onContentLoaded }: {
  fileName: string;
  currentPath?: string;
  onLoad: () => void;
  onError: (error: string) => void;
  onContentLoaded?: (content: string) => void;
}) {
  const [content, setContent] = useState<string>("");
  const { theme } = useTheme();

  useEffect(() => {
    const fetchContent = async () => {
      try {
        const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
        const response = await fetch(`/files?path=${encodeURIComponent(filePath)}`);
        if (!response.ok) {
          throw new Error("Failed to fetch file content");
        }
        const text = await response.text();
        setContent(text);
        if (onContentLoaded) {
          onContentLoaded(text);
        }
        onLoad();
      } catch (error) {
        onError("Failed to load text file");
      }
    };

    fetchContent();
  }, [fileName, currentPath, onLoad, onError]);

  const fileExt = fileName.split(".").pop()?.toLowerCase() || "";
  const isCode = ["js", "ts", "tsx", "jsx", "py", "java", "go", "php", "rb", "html", "css", "scss", "json", "xml", "yml", "yaml", "md", "sh", "bash", "c", "cpp", "cc", "cxx", "h", "hpp", "rs", "vue", "swift", "kt", "kts", "sql", "dockerfile"].includes(fileExt);

  // Get the appropriate icon from the getFileIcon function
  const fileIcon = getFileIcon(fileName);

  const language = getLanguageFromExtension(fileExt);

  return (
    <div className="bg-secondary/30 border-2 border-border rounded-lg p-4 animate-fade-in">
      <div className="flex items-center gap-2 mb-4 pb-3 border-b border-border">
        <div className="flex items-center justify-center w-5 h-5">
          {fileIcon}
        </div>
        {/* <Badge variant="outline" className="font-mono text-xs">
          {fileExt.toUpperCase()}
        </Badge> */}
        {isCode && (
          <Badge variant="secondary" className="font-mono text-xs">
            {language.toUpperCase()}
          </Badge>
        )}
      </div>
      <div className="bg-background border-2 border-border rounded-lg overflow-hidden shadow-sm">
        <div className="p-4 max-h-[50vh] overflow-auto scrollbar-thin bg-muted/30">
          <pre className={`text-sm font-mono ${isCode ? 'language-' + language : ''} whitespace-pre-wrap`}>
            <code className={isCode ? `language-${language}` : undefined}>
              {content}
            </code>
          </pre>
        </div>
      </div>
    </div>
  );
}