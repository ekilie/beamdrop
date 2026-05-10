import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/hooks/use-toast";
import {
  Copy,
  Check,
  Link2,
  Lock,
  Calendar,
  ExternalLink,
  Loader2,
  FolderIcon,
  FileIcon,
} from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface ShareLinkDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fileName: string;
  currentPath: string;
  isDir?: boolean;
}

const EXPIRY_PRESETS = [
  { label: "No expiry", value: "" },
  { label: "1 hour", value: "1" },
  { label: "24 hours", value: "24" },
  { label: "7 days", value: "168" },
  { label: "30 days", value: "720" },
  { label: "Custom...", value: "custom" },
];

export function ShareLinkDialog({
  open,
  onOpenChange,
  fileName,
  currentPath,
  isDir = false,
}: ShareLinkDialogProps) {
  const [shareLink, setShareLink] = useState<string>("");
  const [expiresAt, setExpiresAt] = useState<string | null>(null);
  const [password, setPassword] = useState<string>("");
  const [expiresIn, setExpiresIn] = useState<string>(""); // in hours
  const [expiryPreset, setExpiryPreset] = useState<string>("");
  const [isGenerating, setIsGenerating] = useState(false);
  const [copied, setCopied] = useState(false);

  const getExpiryHours = (): number => {
    if (expiryPreset === "custom") return parseFloat(expiresIn);
    if (expiryPreset) return parseFloat(expiryPreset);
    return NaN;
  };

  const generateShareLink = async () => {
    setIsGenerating(true);
    try {
      const filePath =
        currentPath === "." ? fileName : `${currentPath}/${fileName}`;

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

      const hours = getExpiryHours();
      if (!isNaN(hours) && hours > 0) {
        requestBody.expiresIn = hours * 3600; // Convert hours to seconds
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
      setExpiresAt(data.expiresAt || null);

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
    setExpiryPreset("");
    setExpiresAt(null);
    setCopied(false);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="font-mono uppercase">Share Link</DialogTitle>
          <DialogDescription>
            <span className="inline-flex items-center gap-1.5">
              {isDir ? (
                <FolderIcon className="w-4 h-4 text-primary" />
              ) : (
                <FileIcon className="w-4 h-4 text-muted-foreground" />
              )}
              Create a shareable link for{" "}
              <span className="font-semibold">{fileName}</span>
            </span>
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
              <Label htmlFor="expiryPreset" className="flex items-center gap-2">
                <Calendar className="w-4 h-4" />
                Expiry
              </Label>
              <Select value={expiryPreset} onValueChange={setExpiryPreset}>
                <SelectTrigger id="expiryPreset">
                  <SelectValue placeholder="No expiry" />
                </SelectTrigger>
                <SelectContent>
                  {EXPIRY_PRESETS.map((preset) => (
                    <SelectItem key={preset.value} value={preset.value}>
                      {preset.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {expiryPreset === "custom" && (
                <div className="pt-1">
                  <Input
                    id="expiresIn"
                    type="number"
                    placeholder="Hours until expiry"
                    min="1"
                    value={expiresIn}
                    onChange={(e) => setExpiresIn(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground mt-1">
                    Enter number of hours
                  </p>
                </div>
              )}
            </div>

            <Button
              onClick={generateShareLink}
              disabled={isGenerating}
              className="w-full"
            >
              {isGenerating ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Generating...
                </>
              ) : (
                <>
                  <Link2 className="w-4 h-4 mr-2" />
                  Generate Link
                </>
              )}
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

            {expiresAt && (
              <div className="p-3 bg-blue-500/10 border border-blue-500/20 rounded-md">
                <p className="text-sm text-blue-700 dark:text-blue-400 flex items-center gap-2">
                  <Calendar className="w-4 h-4" />
                  Expires: {new Date(expiresAt).toLocaleString()}
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
