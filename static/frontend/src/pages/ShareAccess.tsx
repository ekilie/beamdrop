import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "@tanstack/react-router";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/hooks/use-toast";
import {
  Download,
  Lock,
  FileIcon,
  FolderIcon,
  Loader2,
  AlertCircle,
  FileQuestion,
} from "lucide-react";
import { getFileIcon } from "@/lib/utils";

interface ShareFileInfo {
  path: string;
  name?: string;
  size?: string;
  sizeBytes?: number;
  contentType?: string;
  isFile?: boolean;
  files?: Array<{
    name: string;
    size: string;
    modTime: string;
    isDir: boolean;
    path: string;
  }>;
  isDir?: boolean;
  requiresPassword?: boolean;
}

type PreviewType = "image" | "video" | "audio" | "pdf" | "unsupported";

function getPreviewType(contentType: string): PreviewType {
  if (contentType.startsWith("image/")) return "image";
  if (contentType.startsWith("video/")) return "video";
  if (contentType.startsWith("audio/")) return "audio";
  if (contentType === "application/pdf") return "pdf";
  return "unsupported";
}

function getInlineUrl(token: string, password?: string): string {
  let url = `/api/shares/access/${token}?mode=inline`;
  if (password) url += `&password=${encodeURIComponent(password)}`;
  return url;
}

function getDownloadUrl(token: string, password?: string): string {
  let url = `/api/shares/access/${token}?mode=download`;
  if (password) url += `&password=${encodeURIComponent(password)}`;
  return url;
}

function FilePreview({
  fileInfo,
  token,
  password,
}: {
  fileInfo: ShareFileInfo;
  token: string;
  password?: string;
}) {
  const previewType = getPreviewType(fileInfo.contentType || "");
  const inlineUrl = getInlineUrl(token, password);
  const downloadUrl = getDownloadUrl(token, password);

  return (
    <div className="min-h-screen bg-background p-4">
      <div className="max-w-4xl mx-auto space-y-4">
        {/* File info header */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3 min-w-0 flex-1">
                <div className="flex-shrink-0">
                  {getFileIcon(fileInfo.name || "", "w-6 h-6")}
                </div>
                <div className="min-w-0">
                  <CardTitle className="truncate text-lg">
                    {fileInfo.name}
                  </CardTitle>
                  <CardDescription>{fileInfo.size}</CardDescription>
                </div>
              </div>
              <Button asChild>
                <a href={downloadUrl} download>
                  <Download className="w-4 h-4 mr-2" />
                  Download
                </a>
              </Button>
            </div>
          </CardHeader>
        </Card>

        {/* Preview area */}
        <Card>
          <CardContent className="p-0 overflow-hidden">
            {previewType === "image" && (
              <div className="flex items-center justify-center bg-muted/30 p-4">
                <img
                  src={inlineUrl}
                  alt={fileInfo.name}
                  className="max-w-full max-h-[70vh] object-contain rounded"
                />
              </div>
            )}

            {previewType === "video" && (
              <div className="flex items-center justify-center bg-black">
                <video
                  src={inlineUrl}
                  controls
                  className="max-w-full max-h-[70vh]"
                  preload="metadata"
                >
                  Your browser does not support the video tag.
                </video>
              </div>
            )}

            {previewType === "audio" && (
              <div className="flex flex-col items-center justify-center gap-4 p-12 bg-muted/30">
                <FileIcon className="w-16 h-16 text-muted-foreground" />
                <p className="text-sm text-muted-foreground font-mono">
                  {fileInfo.name}
                </p>
                <audio
                  src={inlineUrl}
                  controls
                  preload="metadata"
                  className="w-full max-w-md"
                >
                  Your browser does not support the audio tag.
                </audio>
              </div>
            )}

            {previewType === "pdf" && (
              <div className="w-full" style={{ height: "80vh" }}>
                <iframe
                  src={inlineUrl}
                  className="w-full h-full border-0"
                  title={fileInfo.name}
                />
              </div>
            )}

            {previewType === "unsupported" && (
              <div className="flex flex-col items-center justify-center gap-6 py-20 px-4">
                <FileQuestion className="w-20 h-20 text-muted-foreground" />
                <div className="text-center space-y-2">
                  <h3 className="text-lg font-semibold">
                    File preview not supported
                  </h3>
                  <p className="text-sm text-muted-foreground max-w-sm">
                    This file type ({fileInfo.contentType || "unknown"}) cannot
                    be previewed. You can download it instead.
                  </p>
                </div>
                <Button asChild size="lg">
                  <a href={downloadUrl} download>
                    <Download className="w-5 h-5 mr-2" />
                    Download File
                  </a>
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default function ShareAccess() {
  const { token } = useParams({ from: "/share/$token" });
  const navigate = useNavigate();
  const [password, setPassword] = useState("");
  const [fileInfo, setFileInfo] = useState<ShareFileInfo | null>(null);
  const [requiresPassword, setRequiresPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchShareInfo = useCallback(
    async (pwd?: string) => {
      if (!token) return;

      setIsLoading(true);
      setError(null);

      try {
        const url = `/api/shares/access/${token}${pwd ? `?password=${encodeURIComponent(pwd)}` : ""}`;
        const response = await fetch(url);

        if (response.status === 401) {
          const data = await response.json();
          if (data.requiresPassword) {
            setRequiresPassword(true);
            setFileInfo(data);
            setIsLoading(false);
            return;
          }
          throw new Error("Invalid password");
        }

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.error || "Failed to access shared link");
        }

        // Response is always JSON now (metadata for files, listing for dirs)
        const data = await response.json();
        setFileInfo(data);
        setRequiresPassword(false);
        setIsLoading(false);
      } catch (error: any) {
        setError(error.message);
        setIsLoading(false);
        toast({
          title: "Error",
          description: error.message,
          variant: "destructive",
        });
      }
    },
    [token]
  );

  useEffect(() => {
    fetchShareInfo();
  }, [fetchShareInfo]);

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!password.trim()) {
      toast({
        title: "Error",
        description: "Please enter a password",
        variant: "destructive",
      });
      return;
    }
    fetchShareInfo(password);
  };

  const downloadFile = async (filePath: string) => {
    if (!token) return;
    try {
      const url = getDownloadUrl(token, password);
      const a = document.createElement("a");
      a.href = url;
      a.download = filePath.split("/").pop() || "download";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      toast({
        title: "Download started",
        description: "Your file is being downloaded",
      });
    } catch {
      toast({
        title: "Error",
        description: "Failed to download file",
        variant: "destructive",
      });
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-12 h-12 animate-spin text-primary" />
          <p className="text-muted-foreground font-mono uppercase text-sm">
            Loading shared content...
          </p>
        </div>
      </div>
    );
  }

  if (error && !requiresPassword) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle className="w-6 h-6" />
              <CardTitle>Access Error</CardTitle>
            </div>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground mb-4">{error}</p>
            <Button
              onClick={() => navigate("/")}
              variant="outline"
              className="w-full"
            >
              Go to Home
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (requiresPassword && !fileInfo?.files && !fileInfo?.isFile) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Lock className="w-5 h-5" />
              Password Required
            </CardTitle>
            <CardDescription>
              This shared link is protected. Please enter the password to access
              it.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handlePasswordSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter password"
                  autoFocus
                />
              </div>
              <Button type="submit" className="w-full">
                Access Shared Content
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    );
  }

  // Preview a single file
  if (fileInfo?.isFile && token) {
    return (
      <FilePreview fileInfo={fileInfo} token={token} password={password} />
    );
  }

  // Display directory contents
  if (fileInfo?.isDir && fileInfo.files) {
    return (
      <div className="min-h-screen bg-background p-4">
        <div className="max-w-4xl mx-auto">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FolderIcon className="w-6 h-6 text-primary" />
                Shared Folder: {fileInfo.path}
              </CardTitle>
              <CardDescription>
                {fileInfo.files.length} item
                {fileInfo.files.length !== 1 ? "s" : ""}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {fileInfo.files.map((file, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/50 transition-colors"
                  >
                    <div className="flex items-center gap-3 flex-1 min-w-0">
                      <div className="flex-shrink-0">
                        {file.isDir ? (
                          <FolderIcon className="w-5 h-5 text-primary" />
                        ) : (
                          getFileIcon(file.name, "w-5 h-5")
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="font-mono truncate">{file.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {file.size} &bull; {file.modTime}
                        </p>
                      </div>
                    </div>
                    {!file.isDir && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => downloadFile(file.path)}
                      >
                        <Download className="w-4 h-4" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <div className="mt-4 text-center">
            <Button onClick={() => navigate("/")} variant="outline">
              Go to Beamdrop
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return null;
}
