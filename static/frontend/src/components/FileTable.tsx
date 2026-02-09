import React, { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Download,
  RefreshCw,
  MoreHorizontal,
  Eye,
  Trash2,
  Star,
  ArrowUpDown,
  FolderOpen,
  ChevronUp,
  ChevronDown,
  Edit,
  Copy,
  Move,
  Share2,
} from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { getFileIcon } from "@/lib/utils";
import { cn } from "@/lib/utils";
import { useSettings } from "@/context/settings";
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

interface FileTableProps {
  files: FileItem[];
  isLoading: boolean;
  onRefresh: () => void;
  onNavigate: (path: string) => void;
  onPreview: (fileName: string) => void;
  searchTerm?: string;
  currentPath?: string;
  starredFiles: Set<string>;
  onToggleStar: (fileName: string, event: React.MouseEvent) => Promise<void>;
}

type SortField = "name" | "size" | "modTime";
type SortOrder = "asc" | "desc";

const FileTable: React.FC<FileTableProps> = ({
  files,
  isLoading,
  onRefresh,
  onNavigate,
  onPreview,
  searchTerm,
  currentPath = ".",
  starredFiles,
  onToggleStar,
}) => {
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortOrder, setSortOrder] = useState<SortOrder>("asc");
  const { showHiddenFiles } = useSettings();
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

  if (!showHiddenFiles) {
    files = files.filter(file => !file.name.startsWith('.'));
  }
  const sortedFiles = useMemo(() => {
    return [...files].sort((a, b) => {
      let comparison = 0;

      // Always show directories first
      if (a.isDir && !b.isDir) return -1;
      if (!a.isDir && b.isDir) return 1;

      switch (sortField) {
        case "name":
          comparison = a.name.localeCompare(b.name);
          break;
        case "size":
          {
            const parseSize = (s: string) => {
              if (a.isDir || b.isDir) return 0; // Directories don't have meaningful size comparison
              const [num, unit] = s.split(" ");
              const n = parseFloat(num);
              if (unit === "KB") return n * 1024;
              if (unit === "MB") return n * 1024 * 1024;
              if (unit === "GB") return n * 1024 * 1024 * 1024;
              return n;
            };
            comparison = parseSize(a.size) - parseSize(b.size);
            break;
          }
        case "modTime":
          comparison = new Date(a.modTime).getTime() - new Date(b.modTime).getTime();
          break;
      }

      return sortOrder === "asc" ? comparison : -comparison;
    });
  }, [files, sortField, sortOrder]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortOrder("asc");
    }
  };

  const handleFileClick = (file: FileItem) => {
    if (file.isDir) {
      const newPath = currentPath === "." ? file.name : `${currentPath}/${file.name}`;
      onNavigate(newPath);
    } else {
      onPreview(file.name);
    }
  };

  const downloadFile = async (fileName: string, event: React.MouseEvent) => {
    event.stopPropagation();
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

  const deleteFile = async (fileName: string, event: React.MouseEvent) => {
    event.stopPropagation();
    try {
      const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      const response = await fetch('/trash', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ sourcePath: filePath }),
      });

      if (response.ok) {
        onRefresh();
        toast({
          title: "Moved to Trash",
          description: `${fileName} has been moved to trash.`,
        });
      } else {
        const error = await response.json();
        toast({
          title: "Error",
          description: error.error || "Failed to move file to trash",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to move file to trash",
        variant: "destructive",
      });
    }
  };


  const getSortIcon = (field: SortField) => {
    if (sortField !== field) {
      return <ArrowUpDown className="w-3 h-3 text-muted-foreground" />;
    }
    return sortOrder === "asc" ? (
      <ChevronUp className="w-3 h-3 text-primary" />
    ) : (
      <ChevronDown className="w-3 h-3 text-primary" />
    );
  };

  if (isLoading) {
    return (
      <div className="space-y-4 animate-fade-in">
        <div className="flex items-center justify-between">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-8 w-24" />
        </div>
        <div className="border border-border rounded-lg hover-lift">
          <Table>
            <TableHeader>
              <TableRow className="border-b border-border">
                <TableHead className="w-[50%]">Name</TableHead>
                <TableHead className="w-[20%]">Size</TableHead>
                <TableHead className="w-[25%]">Modified</TableHead>
                <TableHead className="w-[5%]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {[1, 2, 3, 4, 5].map((i) => (
                <TableRow key={i}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <Skeleton className="h-6 w-6 animate-pulse" />
                      <Skeleton className="h-4 w-32 animate-pulse" />
                    </div>
                  </TableCell>
                  <TableCell><Skeleton className="h-4 w-16 animate-pulse" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-24 animate-pulse" /></TableCell>
                  <TableCell><Skeleton className="h-8 w-8 animate-pulse" /></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-4 animate-fade-in">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="font-mono text-sm font-bold uppercase tracking-wide text-foreground">
            {sortedFiles.length} ITEM{sortedFiles.length !== 1 ? "S" : ""}
            {searchTerm && (
              <span className="text-muted-foreground ml-2">matching "{searchTerm}"</span>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
            className="gap-2 hover-lift transition-smooth"
          >
            <RefreshCw className="w-4 h-4" />
            <span className="font-mono text-xs font-bold uppercase tracking-wide">
              REFRESH
            </span>
          </Button>
        </div>

        {/* Table */}
        <div className="border-t border-b border-border hover-lift transition-smooth rounded-lg overflow-hidden shadow-sm">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent border-b border-border">
                <TableHead
                  className="cursor-pointer select-none font-mono font-bold uppercase tracking-wide"
                  onClick={() => handleSort("name")}
                >
                  <div className="flex items-center gap-2">
                    Name
                    {getSortIcon("name")}
                  </div>
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none font-mono font-bold uppercase tracking-wide hidden md:table-cell"
                  onClick={() => handleSort("size")}
                >
                  <div className="flex items-center gap-2">
                    Size
                    {getSortIcon("size")}
                  </div>
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none font-mono font-bold uppercase tracking-wide hidden lg:table-cell"
                  onClick={() => handleSort("modTime")}
                >
                  <div className="flex items-center gap-2">
                    Modified
                    {getSortIcon("modTime")}
                  </div>
                </TableHead>
                <TableHead className="w-[80px]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedFiles.map((file) => {
                // Use isStarred from backend response, fallback to Set lookup for backwards compatibility
                const filePath = file.path || (currentPath === "." ? file.name : `${currentPath}/${file.name}`);
                const isStarred = file.isStarred ?? starredFiles.has(filePath);
                return (
                  <TableRow
                    key={`${file.name}-${file.modTime}`}
                    className="cursor-pointer border-b border-border hover:bg-muted/50 transition-all group hover:shadow-sm"
                    onClick={() => handleFileClick(file)}
                  >
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-3">
                        <div className="flex-shrink-0">
                          {file.isDir ? (
                            <FolderOpen className="w-5 h-5 text-primary" />
                          ) : (
                            <div className="text-muted-foreground">
                              {getFileIcon(file.name, "w-5 h-5")}
                            </div>
                          )}
                        </div>
                        <div className="flex items-center gap-2 min-w-0 flex-1">
                          <span className="font-mono truncate">{file.name}</span>
                          {isStarred && (
                            <Star className="w-4 h-4 text-primary fill-primary flex-shrink-0" />
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground hidden md:table-cell">
                      {file.size}
                    </TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground hidden lg:table-cell">
                      {file.modTime}
                    </TableCell>
                    <TableCell onClick={(e) => e.stopPropagation()}>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                          >
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={(e) => {
                              e.stopPropagation();
                              if (file.isDir) {
                                const newPath = currentPath === "." ? file.name : `${currentPath}/${file.name}`;
                                onNavigate(newPath);
                              } else {
                                onPreview(file.name);
                              }
                            }}
                          >
                            <Eye className="w-4 h-4 mr-2" />
                            {file.isDir ? "Open" : "Preview"}
                          </DropdownMenuItem>
                          {!file.isDir && (
                            <DropdownMenuItem onClick={(e) => downloadFile(file.name, e)}>
                              <Download className="w-4 h-4 mr-2" />
                              Download
                            </DropdownMenuItem>
                          )}
                          <DropdownMenuItem onClick={(e) => onToggleStar(file.name, e)}>
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
                            onClick={(e) => deleteFile(file.name, e)}
                            className="text-destructive focus:text-destructive"
                          >
                            <Trash2 className="w-4 h-4 mr-2" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      </div>

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
    </>
  );
};

export default FileTable;