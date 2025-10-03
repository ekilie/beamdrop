import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Upload, FileText, X, Target, Plus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { toast } from "@/hooks/use-toast";
import { Badge } from "@/components/ui/badge";

interface FileUploadModuleProps {
  onUploadSuccess: () => void;
  currentPath?: string;
  className?: string;
}

interface UploadFile {
  file: File;
  id: string;
  progress: number;
  status: "pending" | "uploading" | "success" | "error";
  error?: string;
}

export const FileUploadModule = ({
  onUploadSuccess,
  currentPath = ".",
  className,
}: FileUploadModuleProps) => {
  const [isDragging, setIsDragging] = useState(false);
  const [uploadFiles, setUploadFiles] = useState<UploadFile[]>([]);
  const [isUploading, setIsUploading] = useState(false);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    const files = Array.from(e.dataTransfer.files);
    addFiles(files);
  }, []);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    addFiles(files);
  };

  const addFiles = (files: File[]) => {
    const newUploadFiles: UploadFile[] = files.map((file) => ({
      file,
      id: Math.random().toString(36).substring(7),
      progress: 0,
      status: "pending" as const,
    }));

    setUploadFiles((prev) => [...prev, ...newUploadFiles]);
  };

  const removeFile = (id: string) => {
    setUploadFiles((prev) => prev.filter((f) => f.id !== id));
  };

  const uploadAllFiles = async () => {
    if (uploadFiles.length === 0) return;

    setIsUploading(true);

    const uploadPromises = uploadFiles
      .filter((f) => f.status === "pending")
      .map((uploadFileItem) => uploadSingleFile(uploadFileItem));

    await Promise.all(uploadPromises);

    setIsUploading(false);

    // Check if all uploads were successful
    const allSuccessful = uploadFiles.every((f) => f.status === "success");
    if (allSuccessful) {
      onUploadSuccess();
      setUploadFiles([]);
    }
  };

  const uploadSingleFile = async (uploadFileItem: UploadFile) => {
    const { file, id } = uploadFileItem;

    // Update status to uploading
    setUploadFiles((prev) =>
      prev.map((f) =>
        f.id === id ? { ...f, status: "uploading" as const } : f,
      ),
    );

    try {
      const formData = new FormData();
      formData.append("file", file);
      if (currentPath !== ".") {
        formData.append("path", currentPath);
      }

      const xhr = new XMLHttpRequest();

      xhr.upload.addEventListener("progress", (e) => {
        if (e.lengthComputable) {
          const progress = (e.loaded / e.total) * 100;
          setUploadFiles((prev) =>
            prev.map((f) => (f.id === id ? { ...f, progress } : f)),
          );
        }
      });

      return new Promise<void>((resolve, reject) => {
        xhr.onload = () => {
          if (xhr.status === 200) {
            setUploadFiles((prev) =>
              prev.map((f) =>
                f.id === id
                  ? { ...f, status: "success" as const, progress: 100 }
                  : f,
              ),
            );
            resolve();
          } else {
            let errorMessage = "Upload failed";
            try {
              const error = JSON.parse(xhr.responseText);
              errorMessage = error.error || errorMessage;
            } catch (e) {
              // Use default error message
            }

            setUploadFiles((prev) =>
              prev.map((f) =>
                f.id === id
                  ? { ...f, status: "error" as const, error: errorMessage }
                  : f,
              ),
            );
            reject(new Error(errorMessage));
          }
        };

        xhr.onerror = () => {
          setUploadFiles((prev) =>
            prev.map((f) =>
              f.id === id
                ? { ...f, status: "error" as const, error: "Network error" }
                : f,
            ),
          );
          reject(new Error("Network error"));
        };

        xhr.open("POST", "/upload");
        xhr.send(formData);
      });
    } catch (error) {
      setUploadFiles((prev) =>
        prev.map((f) =>
          f.id === id
            ? { ...f, status: "error" as const, error: "Upload failed" }
            : f,
        ),
      );
    }
  };

  const clearCompleted = () => {
    setUploadFiles((prev) => prev.filter((f) => f.status !== "success"));
  };

  const clearAll = () => {
    setUploadFiles([]);
  };

  const getStatusColor = (status: UploadFile["status"]) => {
    switch (status) {
      case "success":
        return "text-green-600";
      case "error":
        return "text-destructive";
      case "uploading":
        return "text-primary";
      default:
        return "text-muted-foreground";
    }
  };

  return (
    <div className={cn("space-y-6", className)}>
      {/* Drop Zone */}
      <div
        className={cn(
          "relative border-2 border-dashed rounded p-8 text-center transition-smooth bg-background",
          isDragging
            ? "border-primary bg-primary/5"
            : "border-border hover:border-primary",
        )}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        <div className="space-y-4">
          <div className="mx-auto w-16 h-16 bg-accent border-2 border-foreground/20 rounded flex items-center justify-center">
            <Target className="w-8 h-8 text-accent-foreground" />
          </div>

          <div>
            <h3 className="text-lg font-bold font-mono uppercase tracking-wide text-foreground">
              {isDragging ? "DROP FILES HERE" : "FILE UPLOAD ZONE"}
            </h3>
            <p className="text-muted-foreground font-mono text-sm">
              DRAG MULTIPLE FILES HERE OR CLICK TO SELECT
            </p>
            {currentPath !== "." && (
              <Badge variant="outline" className="mt-2 font-mono text-xs">
                Uploading to: {currentPath}
              </Badge>
            )}
          </div>

          <input
            type="file"
            multiple
            onChange={handleFileSelect}
            className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
          />
        </div>
      </div>

      {/* File List */}
      {uploadFiles.length > 0 && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h4 className="font-mono font-bold uppercase tracking-wide text-foreground text-sm">
              Upload Queue ({uploadFiles.length})
            </h4>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={clearCompleted}
                disabled={!uploadFiles.some((f) => f.status === "success")}
              >
                Clear Completed
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={clearAll}
                disabled={isUploading}
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Clear All
              </Button>
            </div>
          </div>

          <div className="space-y-2 max-h-64 overflow-auto">
            {uploadFiles.map((uploadFile) => (
              <div
                key={uploadFile.id}
                className="bg-secondary border border-border rounded p-3"
              >
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-3 min-w-0 flex-1">
                    <div className="bg-primary p-1.5 rounded border border-primary">
                      <FileText className="w-3 h-3 text-primary-foreground" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="font-mono text-sm text-foreground truncate">
                        {uploadFile.file.name}
                      </p>
                      <div className="flex items-center gap-2">
                        <p className="text-xs text-muted-foreground font-mono">
                          {(uploadFile.file.size / 1024 / 1024).toFixed(2)} MB
                        </p>
                        <Badge
                          variant="outline"
                          className={cn(
                            "font-mono text-xs",
                            getStatusColor(uploadFile.status),
                          )}
                        >
                          {uploadFile.status.toUpperCase()}
                        </Badge>
                      </div>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeFile(uploadFile.id)}
                    disabled={uploadFile.status === "uploading"}
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </div>

                {uploadFile.status === "uploading" && (
                  <div className="space-y-1">
                    <Progress value={uploadFile.progress} className="h-2" />
                    <p className="text-xs text-center text-muted-foreground font-mono">
                      {Math.round(uploadFile.progress)}%
                    </p>
                  </div>
                )}

                {uploadFile.status === "error" && uploadFile.error && (
                  <p className="text-xs text-destructive font-mono mt-1">
                    {uploadFile.error}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Upload Button */}
      {uploadFiles.length > 0 &&
        !isUploading &&
        uploadFiles.some((f) => f.status === "pending") && (
          <Button
            onClick={uploadAllFiles}
            variant="utilitarian"
            className="w-full transition-smooth"
          >
            <Upload className="w-4 h-4 mr-2" />
            UPLOAD {
              uploadFiles.filter((f) => f.status === "pending").length
            }{" "}
            FILE(S)
          </Button>
        )}

      {isUploading && (
        <div className="bg-muted p-4 rounded border border-border text-center">
          <p className="text-sm font-mono font-bold uppercase tracking-wide text-muted-foreground">
            UPLOADING FILES...
          </p>
        </div>
      )}
    </div>
  );
};
