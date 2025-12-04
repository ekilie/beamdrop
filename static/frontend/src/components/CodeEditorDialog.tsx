import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { CodeEditor } from "./CodeEditor";
import { Code, FilePlus, Edit } from "lucide-react";

interface CodeEditorDialogProps {
  currentPath?: string;
  initialFileName?: string;
  initialContent?: string;
  onSaveSuccess?: () => void;
  triggerButton?: React.ReactNode;
  mode?: "create" | "edit";
}

export function CodeEditorDialog({
  currentPath = ".",
  initialFileName = "",
  initialContent = "",
  onSaveSuccess,
  triggerButton,
  mode = "create",
}: CodeEditorDialogProps) {
  const [open, setOpen] = useState(false);
  const [fileName, setFileName] = useState(initialFileName);
  const [content, setContent] = useState(initialContent);

  const handleSave = (savedFileName: string, savedContent: string) => {
    setFileName(savedFileName);
    setContent(savedContent);
    setOpen(false);
    
    if (onSaveSuccess) {
      onSaveSuccess();
    }
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      // Reset to initial values when closing
      setFileName(initialFileName);
      setContent(initialContent);
    }
  };

  const defaultTrigger = (
    <Button
      variant="outline"
      size="sm"
      className="gap-2 font-mono uppercase text-xs hover-lift transition-smooth"
    >
      {mode === "create" ? (
        <>
          <FilePlus className="w-4 h-4" />
          <span className="hidden sm:inline">New File</span>
        </>
      ) : (
        <>
          <Edit className="w-4 h-4" />
          <span className="hidden sm:inline">Edit</span>
        </>
      )}
    </Button>
  );

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        {triggerButton || defaultTrigger}
      </DialogTrigger>
      <DialogContent className="max-w-6xl max-h-[95vh] p-0 bg-card border-2 border-border overflow-hidden">
        <DialogHeader className="px-6 py-4 border-b border-border">
          <DialogTitle className="font-mono font-bold text-foreground flex items-center gap-2">
            <Code className="w-5 h-5" />
            {mode === "create" ? "Create New File" : "Edit File"}
          </DialogTitle>
        </DialogHeader>
        <div className="flex-1 overflow-hidden" style={{ height: 'calc(95vh - 5rem)' }}>
          <CodeEditor
            initialFileName={initialFileName}
            initialContent={initialContent}
            currentPath={currentPath}
            onSave={handleSave}
            onClose={() => setOpen(false)}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

