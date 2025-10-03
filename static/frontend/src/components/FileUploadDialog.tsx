import { useState } from "react";
import { Settings, Upload } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { FileUploadModule } from "@/components/FileUploadModule";
import { toast } from "@/hooks/use-toast";

interface FileUploadDialogProps {
  currentPath?: string;
}

export function FileUploadDialog({ currentPath = "." }: FileUploadDialogProps) {
  const [open, setOpen] = useState(false);

  const handleUploadSuccess = () => {
    toast({
      title: "Success",
      description: "File uploaded successfully",
    });
    // Refresh the page or trigger a refresh callback
    window.location.reload();
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant="default"
          className="flex items-center gap-2 font-mono text-xs font-bold uppercase tracking-wide w-full sm:w-auto"
        >
          <Upload className="h-4 w-4" />
          Upload Files
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] w-[400px] mx-auto overflow-y-auto bg-card border-2 border-border">
        <DialogHeader>
          <DialogTitle className="text-xl font-bold font-mono uppercase tracking-wide text-foreground flex items-center gap-2">
            <Upload className="h-4 w-4" />
            Upload Files
          </DialogTitle>
          <DialogDescription className="text-muted-foreground font-mono text-sm">
            UPLOAD FILES TO THE SERVER
            {currentPath !== "." && (
              <span className="block mt-1">
                Current directory: {currentPath}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          <FileUploadModule
            onUploadSuccess={handleUploadSuccess}
            currentPath={currentPath}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
