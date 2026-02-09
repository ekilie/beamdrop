import React, { useState } from "react";
import { motion } from "framer-motion";
import { Download, Trash2, Star, Eye, Folder, MoreVertical, Edit, Copy, Move, Share2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { getFileIcon } from "@/lib/utils";
import { cn } from "@/lib/utils";
import { RenameDialog } from "./RenameDialog";
import { MoveDialog } from "./MoveDialog";
import { ShareLinkDialog } from "./ShareLinkDialog";

interface FileItem {
  name: string;
  size: string;
  modTime: string;
  isDir: boolean;
  path?: string;
  isStarred?: boolean;
}

interface FileGridViewProps {
  files: FileItem[];
  onNavigate: (path: string) => void;
  onPreview: (fileName: string) => void;
  onDownload: (fileName: string, event: React.MouseEvent) => void;
  onDelete: (fileName: string, event: React.MouseEvent) => void;
  onStar: (fileName: string, event: React.MouseEvent) => void;
  starredFiles: Set<string>;
  currentPath: string;
  onRefresh: () => void;
}

export const FileGridView: React.FC<FileGridViewProps> = ({
  files,
  onNavigate,
  onPreview,
  onDownload,
  onDelete,
  onStar,
  starredFiles,
  currentPath,
  onRefresh,
}) => {
  const [renameDialog, setRenameDialog] = useState<{ open: boolean; fileName: string }>({
    open: false,
    fileName: "",
  });
  const [moveDialog, setMoveDialog] = useState<{
    open: boolean;
    fileName: string;
    mode: "move" | "copy";
  }>({
    open: false,
    fileName: "",
    mode: "move",
  });
  const [shareDialog, setShareDialog] = useState<{ open: boolean; fileName: string }>({
    open: false,
    fileName: "",
  });

  const handleFileClick = (file: FileItem) => {
    if (file.isDir) {
      const newPath = currentPath === "." ? file.name : `${currentPath}/${file.name}`;
      onNavigate(newPath);
    } else {
      onPreview(file.name);
    }
  };

  const getFilePreviewBg = (fileName: string) => {
    const ext = fileName.split(".").pop()?.toLowerCase();
    const imageExts = ["jpg", "jpeg", "png", "gif", "webp", "svg"];
    
    if (imageExts.includes(ext || "")) {
      const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      return `/preview?file=${encodeURIComponent(filePath)}`;
    }
    return null;
  };

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4 animate-fade-in">
      {files.map((file, index) => {
        const previewUrl = getFilePreviewBg(file.name);
        // Use isStarred from backend response, fallback to Set lookup for backwards compatibility
        const filePath = file.path || (currentPath === "." ? file.name : `${currentPath}/${file.name}`);
        const isStarred = file.isStarred ?? starredFiles.has(filePath);

        return (
          <motion.div
            key={`${file.name}-${file.modTime}`}
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: index * 0.05, duration: 0.2 }}
            className={cn(
              "group relative bg-card border border-border rounded-lg overflow-hidden hover:border-primary/50 hover:shadow-lg transition-all cursor-pointer hover-lift",
              isStarred && "ring-2 ring-primary/20"
            )}
            onClick={() => handleFileClick(file)}
          >
            {/* Preview Area */}
            <div className="aspect-square bg-muted/30 flex items-center justify-center relative overflow-hidden">
              {file.isDir ? (
                <div className="w-full h-full flex items-center justify-center bg-gradient-to-br from-primary/10 to-accent/10">
                  <Folder className="w-12 h-12 text-primary" />
                </div>
              ) : previewUrl ? (
                <img
                  src={previewUrl}
                  alt={file.name}
                  className="w-full h-full object-cover"
                  loading="lazy"
                />
              ) : (
                <div className="w-full h-full flex items-center justify-center">
                  <div className="w-12 h-12 text-muted-foreground">
                    {getFileIcon(file.name, "w-12 h-12")}
                  </div>
                </div>
              )}
              
              {/* Star Badge */}
              {isStarred && (
                <div className="absolute top-2 left-2">
                  <Star className="w-4 h-4 text-primary fill-primary" />
                </div>
              )}

              {/* Quick Actions Overlay */}
              {!file.isDir && (
                <div className="absolute inset-0 bg-background/80 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                  <Button
                    size="icon"
                    variant="secondary"
                    onClick={(e) => {
                      e.stopPropagation();
                      onPreview(file.name);
                    }}
                    className="h-8 w-8"
                  >
                    <Eye className="w-4 h-4" />
                  </Button>
                  <Button
                    size="icon"
                    variant="secondary"
                    onClick={(e) => onDownload(file.name, e)}
                    className="h-8 w-8"
                  >
                    <Download className="w-4 h-4" />
                  </Button>
                </div>
              )}
            </div>

            {/* File Info */}
            <div className="p-3 space-y-1">
              <div className="flex items-start justify-between gap-2">
                <p className="text-sm font-mono font-medium truncate flex-1" title={file.name}>
                  {file.name}
                </p>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <MoreVertical className="w-4 h-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={(e) => {
                      e.stopPropagation();
                      if (file.isDir) {
                        handleFileClick(file);
                      } else {
                        onPreview(file.name);
                      }
                    }}>
                      <Eye className="w-4 h-4 mr-2" />
                      {file.isDir ? "Open" : "Preview"}
                    </DropdownMenuItem>
                    {!file.isDir && (
                      <DropdownMenuItem onClick={(e) => onDownload(file.name, e)}>
                        <Download className="w-4 h-4 mr-2" />
                        Download
                      </DropdownMenuItem>
                    )}
                    <DropdownMenuItem onClick={(e) => onStar(file.name, e)}>
                      <Star className={cn("w-4 h-4 mr-2", isStarred && "fill-primary")} />
                      {isStarred ? "Unstar" : "Star"}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={(e) => {
                        e.stopPropagation();
                        setShareDialog({ open: true, fileName: file.name });
                      }}
                    >
                      <Share2 className="w-4 h-4 mr-2" />
                      Share Link
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={(e) => {
                        e.stopPropagation();
                        setRenameDialog({ open: true, fileName: file.name });
                      }}
                    >
                      <Edit className="w-4 h-4 mr-2" />
                      Rename
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={(e) => {
                        e.stopPropagation();
                        setMoveDialog({ open: true, fileName: file.name, mode: "move" });
                      }}
                    >
                      <Move className="w-4 h-4 mr-2" />
                      Move
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={(e) => {
                        e.stopPropagation();
                        setMoveDialog({ open: true, fileName: file.name, mode: "copy" });
                      }}
                    >
                      <Copy className="w-4 h-4 mr-2" />
                      Copy
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={(e) => onDelete(file.name, e)}
                      className="text-destructive focus:text-destructive"
                    >
                      <Trash2 className="w-4 h-4 mr-2" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <div className="flex items-center justify-between text-xs text-muted-foreground font-mono">
                <span>{file.size}</span>
              </div>
            </div>
          </motion.div>
        );
      })}

      <RenameDialog
        open={renameDialog.open}
        onOpenChange={(open) => setRenameDialog({ ...renameDialog, open })}
        fileName={renameDialog.fileName}
        currentPath={currentPath}
        onSuccess={onRefresh}
      />

      <MoveDialog
        open={moveDialog.open}
        onOpenChange={(open) => setMoveDialog({ ...moveDialog, open })}
        fileName={moveDialog.fileName}
        currentPath={currentPath}
        onSuccess={onRefresh}
        mode={moveDialog.mode}
      />

      <ShareLinkDialog
        open={shareDialog.open}
        onOpenChange={(open) => setShareDialog({ ...shareDialog, open })}
        fileName={shareDialog.fileName}
        currentPath={currentPath}
      />
    </div>
  );
};
