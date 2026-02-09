import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/hooks/use-toast";
import { Copy, Check, Link2, Lock, Calendar, ExternalLink } from "lucide-react";
import { cn } from "@/lib/utils";

interface ShareLinkDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fileName: string;
  currentPath: string;
}

export function ShareLinkDialog({
  open,
  onOpenChange,
  fileName,
  currentPath,
}: ShareLinkDialogProps) {
  const [shareLink, setShareLink] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [expiresIn, setExpiresIn] = useState<string>(""); // in hours
  const [isGenerating, setIsGenerating] = useState(false);
  const [copied, setCopied] = useState(false);

  const generateShareLink = async () => {
    setIsGenerating(true);
    try {
      const filePath = currentPath === "." ? fileName : `${currentPath}/${fileName}`;
      
      const requestBody: {
        path: string;
        password?: string;
        expiresIn?: number;
      } = {
        path: filePath,
      };

      if (password) {
        requestBody.password = password;
      }

      if (expiresIn) {
        const hours = parseFloat(expiresIn);
        if (!isNaN(hours) && hours > 0) {
          requestBody.expiresIn = hours * 3600; // Convert hours to seconds
        }
      }

      const response = await fetch("/api/shares", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        throw new Error("Failed to create shareable link");
      }

      const data = await response.json();
      setShareLink(data.url);

      toast({
        title: "Success",
        description: "Shareable link created successfully",
      });
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to create shareable link",
        variant: "destructive",
      });
    } finally {
      setIsGenerating(false);
    }
  };

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(shareLink);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      toast({
        title: "Copied",
        description: "Link copied to clipboard",
      });
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to copy link",
        variant: "destructive",
      });
    }
  };

  const openInNewTab = () => {
    window.open(shareLink, "_blank");
  };

  const handleClose = () => {
    setShareLink("");
    setPassword("");
    setExpiresIn("");
    setCopied(false);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="font-mono uppercase">Share Link</DialogTitle>
          <DialogDescription>
            Create a shareable link for <span className="font-semibold">{fileName}</span>
          </DialogDescription>
        </DialogHeader>

        {!shareLink ? (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="password" className="flex items-center gap-2">
                <Lock className="w-4 h-4" />
                Password (optional)
              </Label>
              <Input
                id="password"
                type="password"
                placeholder="Enter password to protect link"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Leave empty for public access
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="expiresIn" className="flex items-center gap-2">
                <Calendar className="w-4 h-4" />
                Expires in (hours, optional)
              </Label>
              <Input
                id="expiresIn"
                type="number"
                placeholder="24"
                min="1"
                value={expiresIn}
                onChange={(e) => setExpiresIn(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Leave empty for permanent link
              </p>
            </div>

            <Button
              onClick={generateShareLink}
              disabled={isGenerating}
              className="w-full"
            >
              <Link2 className="w-4 h-4 mr-2" />
              {isGenerating ? "Generating..." : "Generate Link"}
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="shareLink">Shareable Link</Label>
              <div className="flex gap-2">
                <Input
                  id="shareLink"
                  value={shareLink}
                  readOnly
                  className="font-mono text-sm"
                />
                <Button
                  size="icon"
                  variant="outline"
                  onClick={copyToClipboard}
                  className="flex-shrink-0"
                >
                  {copied ? (
                    <Check className="w-4 h-4 text-green-500" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </Button>
                <Button
                  size="icon"
                  variant="outline"
                  onClick={openInNewTab}
                  className="flex-shrink-0"
                >
                  <ExternalLink className="w-4 h-4" />
                </Button>
              </div>
            </div>

            {password && (
              <div className="p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-md">
                <p className="text-sm text-yellow-700 dark:text-yellow-400 flex items-center gap-2">
                  <Lock className="w-4 h-4" />
                  This link is password protected
                </p>
              </div>
            )}

            {expiresIn && (
              <div className="p-3 bg-blue-500/10 border border-blue-500/20 rounded-md">
                <p className="text-sm text-blue-700 dark:text-blue-400 flex items-center gap-2">
                  <Calendar className="w-4 h-4" />
                  This link will expire in {expiresIn} hour{parseFloat(expiresIn) === 1 ? '' : 's'}
                </p>
              </div>
            )}

            <Button onClick={handleClose} className="w-full" variant="outline">
              Done
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
