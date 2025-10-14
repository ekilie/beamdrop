import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/hooks/use-toast";
import { FolderOpen } from "lucide-react";

interface MoveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fileName: string;
  currentPath: string;
  onSuccess: () => void;
  mode: "move" | "copy";
}

export function MoveDialog({ open, onOpenChange, fileName, currentPath, onSuccess, mode }: MoveDialogProps) {
  const [targetPath, setTargetPath] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);

  useEffect(() => {
    setTargetPath(currentPath === "." ? fileName : `${currentPath}/${fileName}`);
  }, [fileName, currentPath]);

  const handleProcess = async () => {
    if (!targetPath.trim()) {
      toast({
        title: "Error",
        description: "Please enter a target path",
        variant: "destructive",
      });
      return;
    }

    setIsProcessing(true);
    try {
      const sourcePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      const endpoint = mode === "move" ? "/move" : "/copy";
      
      const response = await fetch(endpoint, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ sourcePath, targetPath }),
      });

      if (response.ok) {
        toast({
          title: mode === "move" ? "Moved Successfully" : "Copied Successfully",
          description: `"${fileName}" has been ${mode === "move" ? "moved" : "copied"} to "${targetPath}".`,
        });
        onOpenChange(false);
        onSuccess();
      } else {
        const error = await response.json();
        toast({
          title: "Error",
          description: error.error || `Failed to ${mode}`,
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: `Failed to ${mode} file/folder`,
        variant: "destructive",
      });
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="font-mono uppercase tracking-wide flex items-center gap-2">
            <FolderOpen className="w-5 h-5" />
            {mode === "move" ? "Move" : "Copy"} File/Folder
          </DialogTitle>
          <DialogDescription className="font-mono text-xs">
            {mode === "move" ? "Move" : "Copy"} "{fileName}" to a new location
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="targetPath" className="font-mono text-xs uppercase">
              Target Path
            </Label>
            <Input
              id="targetPath"
              value={targetPath}
              onChange={(e) => setTargetPath(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  handleProcess();
                }
              }}
              placeholder="e.g., documents/new-folder/file.txt"
              className="font-mono"
            />
            <p className="text-xs text-muted-foreground font-mono">
              Current: {currentPath === "." ? "/" : currentPath}/{fileName}
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isProcessing}>
            Cancel
          </Button>
          <Button onClick={handleProcess} disabled={isProcessing || !targetPath.trim()}>
            {isProcessing ? "Processing..." : (mode === "move" ? "Move" : "Copy")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
