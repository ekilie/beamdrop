import { useState, useEffect, useCallback } from "react";
import {
  Share2,
  Trash2,
  Copy,
  Check,
  RefreshCw,
  ExternalLink,
  Lock,
  Calendar,
  Eye,
  Folder,
  FileIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { toast } from "@/hooks/use-toast";
import { getFileIcon } from "@/lib/utils";

interface ShareableLink {
  id: number;
  path: string;
  token: string;
  hasPassword: boolean;
  expiresAt?: string;
  accessCount: number;
  createdAt: string;
}

export function SharesManagementPage() {
  const [links, setLinks] = useState<ShareableLink[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [linkToDelete, setLinkToDelete] = useState<ShareableLink | null>(null);
  const [copiedToken, setCopiedToken] = useState<string | null>(null);

  const fetchLinks = useCallback(async () => {
    try {
      setIsLoading(true);
      const response = await fetch("/api/shares/list");
      if (!response.ok) throw new Error("Failed to fetch shareable links");
      const data = await response.json();
      setLinks(data || []);
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to fetch shareable links",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchLinks();
  }, [fetchLinks]);

  const handleCopyLink = async (token: string) => {
    try {
      const url = `${window.location.origin}/share/${token}`;
      await navigator.clipboard.writeText(url);
      setCopiedToken(token);
      setTimeout(() => setCopiedToken(null), 2000);
      toast({
        title: "Copied",
        description: "Shareable link copied to clipboard",
      });
    } catch {
      toast({
        title: "Error",
        description: "Failed to copy to clipboard",
        variant: "destructive",
      });
    }
  };

  const handleOpenLink = (token: string) => {
    window.open(`/share/${token}`, "_blank");
  };

  const handleDeleteLink = async () => {
    if (!linkToDelete) return;

    try {
      const response = await fetch(`/api/shares/delete?token=${linkToDelete.token}`, {
        method: "DELETE",
      });

      if (!response.ok) throw new Error("Failed to delete link");

      toast({
        title: "Success",
        description: "Shareable link deleted successfully",
      });

      fetchLinks();
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to delete shareable link",
        variant: "destructive",
      });
    } finally {
      setDeleteDialogOpen(false);
      setLinkToDelete(null);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const isExpired = (expiresAt?: string) => {
    if (!expiresAt) return false;
    return new Date(expiresAt) < new Date();
  };

  const getFileName = (path: string) => {
    return path.split("/").pop() || path;
  };

  return (
    <div className="p-6 space-y-6 animate-fade-in">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-3xl font-bold font-mono uppercase tracking-wide">
            Shareable Links
          </h1>
          <p className="text-muted-foreground mt-1">
            Manage your shared files and folders
          </p>
        </div>
        <Button onClick={fetchLinks} variant="outline" size="sm" className="gap-2">
          <RefreshCw className="w-4 h-4" />
          Refresh
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Share2 className="w-5 h-5" />
            Active Shares
          </CardTitle>
          <CardDescription>
            {links.length} shareable link{links.length !== 1 ? "s" : ""}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : links.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Share2 className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>No shareable links yet</p>
              <p className="text-sm mt-2">
                Share files from the file browser to create links
              </p>
            </div>
          ) : (
            <div className="border rounded-lg overflow-hidden">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="font-mono uppercase">Path</TableHead>
                    <TableHead className="font-mono uppercase text-center">Status</TableHead>
                    <TableHead className="font-mono uppercase text-center hidden md:table-cell">
                      Access Count
                    </TableHead>
                    <TableHead className="font-mono uppercase hidden lg:table-cell">
                      Created
                    </TableHead>
                    <TableHead className="font-mono uppercase text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {links.map((link) => (
                    <TableRow key={link.id}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <FileIcon className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                          <span className="font-mono truncate max-w-xs">
                            {getFileName(link.path)}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="text-center">
                        <div className="flex flex-wrap gap-1 justify-center">
                          {link.hasPassword && (
                            <Badge variant="secondary" className="gap-1">
                              <Lock className="w-3 h-3" />
                              <span className="hidden sm:inline">Password</span>
                            </Badge>
                          )}
                          {link.expiresAt && (
                            <Badge
                              variant={isExpired(link.expiresAt) ? "destructive" : "default"}
                              className="gap-1"
                            >
                              <Calendar className="w-3 h-3" />
                              <span className="hidden sm:inline">
                                {isExpired(link.expiresAt) ? "Expired" : "Expiring"}
                              </span>
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-center font-mono hidden md:table-cell">
                        {link.accessCount}
                      </TableCell>
                      <TableCell className="font-mono text-sm text-muted-foreground hidden lg:table-cell">
                        {formatDate(link.createdAt)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleCopyLink(link.token)}
                            title="Copy link"
                          >
                            {copiedToken === link.token ? (
                              <Check className="w-4 h-4 text-green-500" />
                            ) : (
                              <Copy className="w-4 h-4" />
                            )}
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleOpenLink(link.token)}
                            title="Open link"
                          >
                            <ExternalLink className="w-4 h-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => {
                              setLinkToDelete(link);
                              setDeleteDialogOpen(true);
                            }}
                            title="Delete link"
                          >
                            <Trash2 className="w-4 h-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Shareable Link</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this shareable link for{" "}
              <span className="font-semibold">{linkToDelete?.path}</span>? This action cannot
              be undone and the link will no longer be accessible.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDeleteLink} className="bg-destructive text-destructive-foreground">
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
