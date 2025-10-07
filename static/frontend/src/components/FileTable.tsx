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
  Archive,
  Folder,
  Clock,
  MoreHorizontal,
  Eye,
  Trash2,
  Star,
  ArrowUpDown,
  FolderOpen,
  ChevronUp,
  ChevronDown,
} from "lucide-react";
import { toast } from "@/hooks/use-toast";
import { getFileIcon } from "@/lib/utils";
import { cn } from "@/lib/utils";
import { useSettings } from "@/context/settings";

interface FileItem {
  name: string;
  size: string;
  modTime: string;
  isDir: boolean;
  path?: string;
}

interface FileTableProps {
  files: FileItem[];
  isLoading: boolean;
  onRefresh: () => void;
  onNavigate: (path: string) => void;
  onPreview: (fileName: string) => void;
  searchTerm?: string;
  currentPath?: string;
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
}) => {
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortOrder, setSortOrder] = useState<SortOrder>("asc");
  const [starredFiles, setStarredFiles] = useState<Set<string>>(new Set());
  const { showHiddenFiles } = useSettings();

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
      const response = await fetch(`/delete?file=${encodeURIComponent(filePath)}`, {
        method: 'DELETE',
      });
      
      if (response.ok) {
        onRefresh();
        toast({
          title: "File Deleted",
          description: `${fileName} has been deleted.`,
        });
      } else {
        const error = await response.json();
        toast({
          title: "Error",
          description: error.error || "Failed to delete file",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to delete file",
        variant: "destructive",
      });
    }
  };

  const toggleStar = (fileName: string, event: React.MouseEvent) => {
    event.stopPropagation();
    setStarredFiles(prev => {
      const newSet = new Set(prev);
      if (newSet.has(fileName)) {
        newSet.delete(fileName);
      } else {
        newSet.add(fileName);
      }
      return newSet;
    });
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
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-8 w-24" />
        </div>
        <div className="border border-border rounded-lg">
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
                      <Skeleton className="h-6 w-6" />
                      <Skeleton className="h-4 w-32" />
                    </div>
                  </TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-8 w-8" /></TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="font-mono text-sm font-bold uppercase tracking-wide text-foreground">
          {sortedFiles.length} ITEM{sortedFiles.length !== 1 ? "S" : ""}{