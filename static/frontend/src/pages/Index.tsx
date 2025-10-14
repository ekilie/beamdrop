import { useState, useEffect, useCallback } from "react";
import { FileUpload } from "@/components/FileUpload";
import { Card } from "@/components/ui/card";
import { Upload, Download, Server, Search, Grid3x3, List, FolderPlus } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { toast } from "@/hooks/use-toast";
import Footer from "@/components/Footer";
import FileList from "@/components/FileList";
import { FileUploadDialog } from "@/components/FileUploadDialog";
import FileTable from "@/components/FileTable";
import { BreadcrumbNav } from "@/components/BreadcrumbNav";
import { FilePreview } from "@/components/FilePreview";
import { FileUploadModule } from "@/components/FileUploadModule";
import { EmptyState } from "@/components/EmptyState";
import { FileGridView } from "@/components/FileGridView";
import { DropZone } from "@/components/DropZone";
import { CreateFolderDialog } from "@/components/CreateFolderDialog";
import { AdvancedSearch } from "@/components/AdvancedSearch";

export interface FileItem {
  name: string;
  size: string;
  isDir: boolean;
  modTime: string;
  path: string;
}

const Index = () => {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [filteredFiles, setFilteredFiles] = useState<FileItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [currentPath, setCurrentPath] = useState(".");
  const [previewFile, setPreviewFile] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<"table" | "grid">("table");
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [starredFiles, setStarredFiles] = useState<Set<string>>(new Set());

  const fetchFiles = useCallback(
    async (path: string = currentPath) => {
      try {
        setIsLoading(true);
        const response = await fetch(`/files?path=${encodeURIComponent(path)}`);
        if (!response.ok) {
          throw new Error("Failed to fetch files");
        }
        const fileList: FileItem[] = await response.json();
        setFiles(fileList);
        setFilteredFiles(fileList);
      } catch (error) {
        toast({
          title: "Error",
          description: "Failed to fetch files",
          variant: "destructive",
        });
      } finally {
        setIsLoading(false);
      }
    },
    [currentPath],
  );

  const handleSearch = (term: string) => {
    setSearchTerm(term);
    if (!term.trim()) {
      setFilteredFiles(files);
    } else {
      const filtered = files.filter((file) =>
        file.name.toLowerCase().includes(term.toLowerCase()),
      );
      setFilteredFiles(filtered);
    }
  };

  const handleNavigate = (path: string) => {
    setCurrentPath(path);
    setSearchTerm(""); // Clear search when navigating
    fetchFiles(path);
  };

  const handlePreview = (fileName: string) => {
    setPreviewFile(fileName);
  };

  useEffect(() => {
    const initializeApp = () => {
      fetchFiles();
    };

    initializeApp();

    // Add keyboard shortcuts
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.ctrlKey || e.metaKey) {
        switch (e.key) {
          case "f": {
            e.preventDefault();
            // Focus search input
            const searchInput = document.querySelector(
              'input[placeholder="SEARCH FILES..."]',
            ) as HTMLInputElement;
            searchInput?.focus();
            break;
          }
          case "r":
            e.preventDefault();
            fetchFiles();
            break;
        }
      }
      if (e.key === "Escape" && searchTerm) {
        setSearchTerm("");
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [searchTerm, fetchFiles]);

  const handleUploadSuccess = () => {
    fetchFiles();
    toast({
      title: "Success",
      description: "File uploaded successfully",
    });
  };

  const handleDrop = async (droppedFiles: File[]) => {
    const formData = new FormData();
    droppedFiles.forEach((file) => {
      formData.append("files", file);
    });
    formData.append("path", currentPath);

    try {
      const response = await fetch("/upload", {
        method: "POST",
        body: formData,
      });

      if (response.ok) {
        handleUploadSuccess();
      } else {
        toast({
          title: "Error",
          description: "Failed to upload files",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to upload files",
        variant: "destructive",
      });
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
        fetchFiles();
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

  const toggleStar = async (fileName: string, event: React.MouseEvent) => {
    event.stopPropagation();
    try {
      const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      const response = await fetch("/star", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ filePath }),
      });

      if (response.ok) {
        setStarredFiles(prev => {
          const newSet = new Set(prev);
          if (newSet.has(fileName)) {
            newSet.delete(fileName);
            toast({
              title: "Unstarred",
              description: `${fileName} removed from starred files.`,
            });
          } else {
            newSet.add(fileName);
            toast({
              title: "Starred",
              description: `${fileName} added to starred files.`,
            });
          }
          return newSet;
        });
      } else {
        toast({
          title: "Error",
          description: "Failed to update starred status",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to update starred status",
        variant: "destructive",
      });
    }
  };

  return (
    <DropZone onDrop={handleDrop} className="bg-background min-h-screen flex flex-col">
      {/* Modern Header */}
      <header className="border-b border-border bg-card backdrop-blur-sm sticky top-0 z-40">
        <div className="container mx-auto px-4 sm:px-6 py-4">
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            {/* Logo and Breadcrumb */}
            <div className="flex flex-col gap-3 min-w-0 flex-1 lg:flex-initial">
              {/* <div className="flex items-center gap-4">
                <div className="bg-primary p-2 rounded border border-primary shadow-subtle">
                  <Server className="w-6 h-6 text-primary-foreground" />
                </div>
                <div>
                  <h1 className="text-2xl font-bold font-mono uppercase tracking-wide text-foreground">
                    beamdrop
                  </h1>
                  <p className="text-muted-foreground font-mono text-xs">
                    FILE MANAGEMENT SYSTEM
                  </p>
                </div>
              </div> */}

              {/* Breadcrumb Navigation */}
              <BreadcrumbNav
                currentPath={currentPath}
                onNavigate={handleNavigate}
                className="max-w-full"
              />
            </div>

            {/* Search and Upload */}
            <div className="flex flex-col sm:flex-row gap-3 lg:w-auto w-full flex-shrink-0">
              <div className="relative flex-1 lg:flex-initial lg:w-80">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                  placeholder="SEARCH FILES..."
                  value={searchTerm}
                  onChange={(e) => handleSearch(e.target.value)}
                  className="pl-10 pr-16 font-mono text-sm uppercase tracking-wide border-2 border-border"
                />
                {searchTerm && (
                  <button
                    onClick={() => setSearchTerm("")}
                    className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                    aria-label="Clear search"
                  >
                    ✕
                  </button>
                )}
              </div>
              <div className="flex items-center gap-2">
                <div className="flex items-center border border-border rounded-lg p-1 bg-muted/30">
                  <Button
                    variant={viewMode === "table" ? "default" : "ghost"}
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => setViewMode("table")}
                  >
                    <List className="w-4 h-4" />
                  </Button>
                  <Button
                    variant={viewMode === "grid" ? "default" : "ghost"}
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => setViewMode("grid")}
                  >
                    <Grid3x3 className="w-4 h-4" />
                  </Button>
                </div>
                <CreateFolderDialog currentPath={currentPath} onSuccess={() => fetchFiles()} />
                <AdvancedSearch 
                  currentPath={currentPath} 
                  onNavigate={handleNavigate} 
                  onPreview={handlePreview}
                />
                <FileUploadDialog />
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 container mx-auto px-4 sm:px-6 py-6">
        <div className="flex flex-col gap-6 lg:gap-8 h-full">
          {/* Upload Section */}
          {/* <section className="xl:col-span-1 space-y-4">
            <header className="flex items-center gap-3 pb-4 border-b border-border">
              <div className="bg-primary p-2 rounded border border-primary">
                <Upload className="w-5 h-5 text-primary-foreground" />
              </div>
              <div>
                <h2 className="text-xl font-bold font-mono uppercase tracking-wide text-foreground">
                  Upload Files
                </h2>
                <p className="text-muted-foreground font-mono text-xs">
                  TRANSFER FILES TO SERVER
                </p>
              </div>
            </header>
            <Card className="p-4 sm:p-6 bg-card border-2 border-border shadow-medium">
              <FileUploadModule
                onUploadSuccess={handleUploadSuccess}
                currentPath={currentPath}
              />
            </Card>
          </section> */}

          {/* Files Section */}
          <section className="space-y-4">
            <header className="flex items-center gap-3 pb-4 border-b border-border">
              <div className="bg-primary p-2 rounded border border-primary">
                <Download className="w-5 h-5 text-primary-foreground" />
              </div>
              <div>
                <h2 className="text-xl font-bold font-mono uppercase tracking-wide text-foreground">
                  Files & Folders
                </h2>
                <p className="text-muted-foreground font-mono text-xs">
                  BROWSE AND MANAGE FILES
                </p>
              </div>
            </header>
            <Card className="p-4 sm:p-6 bg-card border border-border min-h-[600px]">
              {isLoading ? (
                <div className="space-y-4">
                  {viewMode === "table" ? (
                    <FileTable
                      files={[]}
                      isLoading={true}
                      onRefresh={() => fetchFiles()}
                      onNavigate={handleNavigate}
                      onPreview={handlePreview}
                      searchTerm={searchTerm}
                      currentPath={currentPath}
                      starredFiles={starredFiles}
                      onToggleStar={toggleStar}
                    />
                  ) : (
                    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
                      {[1, 2, 3, 4, 5, 6].map((i) => (
                        <div key={i} className="animate-pulse">
                          <div className="aspect-square bg-muted rounded-lg mb-2" />
                          <div className="h-4 bg-muted rounded w-3/4 mb-1" />
                          <div className="h-3 bg-muted rounded w-1/2" />
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ) : filteredFiles.length === 0 ? (
                <EmptyState 
                  searchTerm={searchTerm} 
                  onUploadClick={() => setUploadDialogOpen(true)}
                />
              ) : viewMode === "table" ? (
                <FileTable
                  files={filteredFiles}
                  isLoading={false}
                  onRefresh={() => fetchFiles()}
                  onNavigate={handleNavigate}
                  onPreview={handlePreview}
                  searchTerm={searchTerm}
                  currentPath={currentPath}
                  starredFiles={starredFiles}
                  onToggleStar={toggleStar}
                />
              ) : (
                <FileGridView
                  files={filteredFiles}
                  onNavigate={handleNavigate}
                  onPreview={handlePreview}
                  onDownload={downloadFile}
                  onDelete={deleteFile}
                  onStar={toggleStar}
                  starredFiles={starredFiles}
                  currentPath={currentPath}
                  onRefresh={() => fetchFiles()}
                />
              )}
            </Card>
          </section>
        </div>
      </main>

      {/* File Preview Modal */}
      {previewFile && (
        <FilePreview
          fileName={previewFile}
          isOpen={!!previewFile}
          onClose={() => setPreviewFile(null)}
          currentPath={currentPath}
        />
      )}

      <Footer />
    </DropZone>
  );
};

export default Index;