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

interface RenameDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fileName: string;
  currentPath: string;
  onSuccess: () => void;
}

export function RenameDialog({ open, onOpenChange, fileName, currentPath, onSuccess }: RenameDialogProps) {
  const [newName, setNewName] = useState(fileName);
  const [isRenaming, setIsRenaming] = useState(false);

  useEffect(() => {
    setNewName(fileName);
  }, [fileName]);

  const handleRename = async () => {
    if (!newName.trim() || newName === fileName) {
      toast({
        title: "Error",
        description: "Please enter a different name",
        variant: "destructive",
      });
      return;
    }

    setIsRenaming(true);
    try {
      const oldPath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      const response = await fetch("/rename", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ oldPath, newName }),
      });

      if (response.ok) {
        toast({
          title: "Renamed Successfully",
          description: `"${fileName}" has been renamed to "${newName}".`,
        });
        onOpenChange(false);
        onSuccess();
      } else {
        const error = await response.json();
        toast({
          title: "Error",
          description: error.error || "Failed to rename",
          variant: "destructive",
        });
      }
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to rename file/folder",
        variant: "destructive",
      });
    } finally {
      setIsRenaming(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="font-mono uppercase tracking-wide">Rename</DialogTitle>
          <DialogDescription className="font-mono text-xs">
            Enter a new name for "{fileName}"
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="newName" className="font-mono text-xs uppercase">
              New Name
            </Label>
            <Input
              id="newName"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  handleRename();
                }
              }}
              className="font-mono"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isRenaming}>
            Cancel
          </Button>
          <Button onClick={handleRename} disabled={isRenaming || !newName.trim()}>
            {isRenaming ? "Renaming..." : "Rename"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
