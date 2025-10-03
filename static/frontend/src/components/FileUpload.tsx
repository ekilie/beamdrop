import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Upload, FileText, X, Target } from "lucide-react";
import { cn } from "@/lib/utils";
import { toast } from "@/hooks/use-toast";

interface FileUploadProps {
  onUploadSuccess: () => void;
}

export const FileUpload = ({ onUploadSuccess }: FileUploadProps) => {
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

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
    if (files.length > 0) {
      setSelectedFile(files[0]);
    }
  }, []);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
    }
  };

  const uploadFile = async () => {
    if (!selectedFile) return;

    setIsUploading(true);
    setUploadProgress(0);

    try {
      const formData = new FormData();
      formData.append("file", selectedFile);

      const xhr = new XMLHttpRequest();

      xhr.upload.addEventListener("progress", (e) => {
        if (e.lengthComputable) {
          const progress = (e.loaded / e.total) * 100;
          setUploadProgress(progress);
        }
      });

      xhr.onload = () => {
        if (xhr.status === 200) {
          try {
            const result = JSON.parse(xhr.responseText);
            onUploadSuccess();
            setSelectedFile(null);
            setUploadProgress(0);
            toast({
              title: "Success",
              description: result.message || "File uploaded successfully",
            });
          } catch (e) {
            // Fallback for non-JSON response
            onUploadSuccess();
            setSelectedFile(null);
            setUploadProgress(0);
            toast({
              title: "Success",
              description: "File uploaded successfully",
            });
          }
        } else {
          try {
            const error = JSON.parse(xhr.responseText);
            toast({
              title: "Error",
              description: error.error || "Upload failed",
              variant: "destructive",
            });
          } catch (e) {
            toast({
              title: "Error",
              description: "Upload failed",
              variant: "destructive",
            });
          }
        }
        setIsUploading(false);
      };

      xhr.onerror = () => {
        toast({
          title: "Error",
          description: "Upload failed",
          variant: "destructive",
        });
        setIsUploading(false);
      };

      xhr.open("POST", "/upload");
      xhr.send(formData);
    } catch (error) {
      toast({
        title: "Error",
        description: "Upload failed",
        variant: "destructive",
      });
      setIsUploading(false);
    }
  };

  const removeFile = () => {
    setSelectedFile(null);
    setUploadProgress(0);
  };

  return (
    <div className="space-y-6">
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
              {isDragging ? "DROP TARGET ACTIVE" : "FILE UPLOAD ZONE"}
            </h3>
            <p className="text-muted-foreground font-mono text-sm">
              DRAG FILE HERE OR CLICK TO SELECT
            </p>
          </div>

          <input
            type="file"
            onChange={handleFileSelect}
            className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
          />
        </div>
      </div>

      {/* Selected File */}
      {selectedFile && (
        <div className="bg-secondary border-2 border-border rounded p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="bg-primary p-2 rounded border border-primary">
                <FileText className="w-4 h-4 text-primary-foreground" />
              </div>
              <div>
                <p className="font-bold font-mono text-foreground">
                  {selectedFile.name}
                </p>
                <p className="text-sm text-muted-foreground font-mono">
                  {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                </p>
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={removeFile}
              disabled={isUploading}
            >
              <X className="w-4 h-4" />
            </Button>
          </div>

          {/* Upload Progress */}
          {isUploading && (
            <div className="mt-4 space-y-3">
              <Progress value={uploadProgress} className="w-full" />
              <div className="bg-muted p-2 rounded border border-border">
                <p className="text-sm text-center text-muted-foreground font-mono font-bold uppercase tracking-wide">
                  UPLOADING... {Math.round(uploadProgress)}%
                </p>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Upload Button */}
      {selectedFile && !isUploading && (
        <Button
          onClick={uploadFile}
          variant="utilitarian"
          className="w-full transition-smooth"
        >
          <Upload className="w-4 h-4 mr-2" />
          UPLOAD FILE
        </Button>
      )}
    </div>
  );
};
