import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
  Eye,
  Calendar
} from "lucide-react";
import { getFileIcon } from "@/lib/utils";

interface ShareFileInfo {
  path: string;
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

export default function ShareAccess() {
  const { token } = useParams<{ token: string }>();
  const navigate = useNavigate();
  const [password, setPassword] = useState("");
  const [fileInfo, setFileInfo] = useState<ShareFileInfo | null>(null);
  const [requiresPassword, setRequiresPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchShareInfo = useCallback(async (pwd?: string) => {
    if (!token) return;

    setIsLoading(true);
    setError(null);

    try {
      const url = `/share/${token}${pwd ? `?password=${encodeURIComponent(pwd)}` : ""}`;
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

      const contentType = response.headers.get("content-type");
      
      if (contentType?.includes("application/json")) {
        // It's a directory
        const data = await response.json();
        setFileInfo(data);
        setRequiresPassword(false);
      } else {
        // It's a file - trigger download
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        
        // Try to get filename from response headers
        const contentDisposition = response.headers.get("content-disposition");
        let filename = "download";
        if (contentDisposition) {
          const match = contentDisposition.match(/filename="?(.+)"?/);
          if (match) filename = match[1];
        }
        
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        
        toast({
          title: "Download started",
          description: "Your file is being downloaded",
        });
      }

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
  }, [token]);

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
    try {
      const response = await fetch(`/share/${token}?password=${encodeURIComponent(password)}`);
      if (!response.ok) throw new Error("Failed to download file");
      
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filePath.split("/").pop() || "download";
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      
      toast({
        title: "Download started",
        description: "Your file is being downloaded",
      });
    } catch (error) {
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
            <Button onClick={() => navigate("/")} variant="outline" className="w-full">
              Go to Home
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (requiresPassword && !fileInfo?.files) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Lock className="w-5 h-5" />
              Password Required
            </CardTitle>
            <CardDescription>
              This shared link is protected. Please enter the password to access it.
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
                {fileInfo.files.length} item{fileInfo.files.length !== 1 ? "s" : ""}
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
                          {file.size} • {file.modTime}
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
