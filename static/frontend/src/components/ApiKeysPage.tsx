import { useState, useEffect, useCallback } from "react";
import {
    Key,
    Plus,
    Trash2,
    Copy,
    Check,
    AlertTriangle,
    Clock,
    Shield,
    Eye,
    EyeOff,
    RefreshCw,
    BookOpen,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
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
import { ApiDocsDialog } from "./ApiDocsDialog";
import { CreateApiKeyDialog } from "./CreateApiKeyDialog";

interface ApiKey {
    id: number;
    name: string;
    accessKeyId: string;
    permissions?: string;
    bucketScope?: string;
    expiresAt?: string;
    lastUsedAt?: string;
    createdAt: string;
    disabled: boolean;
}

export function ApiKeysPage() {
    const [keys, setKeys] = useState<ApiKey[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [createDialogOpen, setCreateDialogOpen] = useState(false);
    const [docsDialogOpen, setDocsDialogOpen] = useState(false);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [keyToDelete, setKeyToDelete] = useState<ApiKey | null>(null);
    const [copiedId, setCopiedId] = useState<string | null>(null);

    const fetchKeys = useCallback(async () => {
        try {
            setIsLoading(true);
            const response = await fetch("/api/v1/keys");
            if (!response.ok) throw new Error("Failed to fetch keys");
            const data = await response.json();
            setKeys(data.keys || []);
        } catch (error) {
            toast({
                title: "Error",
                description: "Failed to fetch API keys",
                variant: "destructive",
            });
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchKeys();
    }, [fetchKeys]);

    const handleCopyKey = async (accessKeyId: string) => {
        try {
            await navigator.clipboard.writeText(accessKeyId);
            setCopiedId(accessKeyId);
            setTimeout(() => setCopiedId(null), 2000);
            toast({
                title: "Copied",
                description: "Access key ID copied to clipboard",
            });
        } catch {
            toast({
                title: "Error",
                description: "Failed to copy to clipboard",
                variant: "destructive",
            });
        }
    };

    const handleDeleteKey = async () => {
        if (!keyToDelete) return;

        try {
            const response = await fetch(
                `/api/v1/keys?accessKeyId=${encodeURIComponent(keyToDelete.accessKeyId)}`,
                { method: "DELETE" }
            );

            if (response.ok) {
                toast({
                    title: "Deleted",
                    description: `API key "${keyToDelete.name}" has been deleted`,
                });
                fetchKeys();
            } else {
                const error = await response.json();
                toast({
                    title: "Error",
                    description: error.error?.message || "Failed to delete key",
                    variant: "destructive",
                });
            }
        } catch {
            toast({
                title: "Error",
                description: "Failed to delete API key",
                variant: "destructive",
            });
        } finally {
            setDeleteDialogOpen(false);
            setKeyToDelete(null);
        }
    };

    const formatDate = (dateString: string) => {
        return new Date(dateString).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });
    };

    const isExpired = (expiresAt?: string) => {
        if (!expiresAt) return false;
        return new Date(expiresAt) < new Date();
    };

    return (
        <div className="p-6 space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold font-mono uppercase tracking-wide text-foreground flex items-center gap-2">
                        <Key className="w-6 h-6" />
                        API Keys
                    </h1>
                    <p className="text-sm text-muted-foreground font-mono mt-1">
                        Manage API keys for S3-compatible storage access
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setDocsDialogOpen(true)}
                        className="font-mono uppercase text-xs"
                    >
                        <BookOpen className="w-4 h-4 mr-2" />
                        API Docs
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={fetchKeys}
                        className="font-mono uppercase text-xs"
                    >
                        <RefreshCw className="w-4 h-4 mr-2" />
                        Refresh
                    </Button>
                    <Button
                        size="sm"
                        onClick={() => setCreateDialogOpen(true)}
                        className="font-mono uppercase text-xs"
                    >
                        <Plus className="w-4 h-4 mr-2" />
                        Create Key
                    </Button>
                </div>
            </div>

            {/* Stats Cards */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <Card className="p-4 bg-card border border-border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-primary/10 rounded-lg">
                            <Key className="w-5 h-5 text-primary" />
                        </div>
                        <div>
                            <p className="text-2xl font-bold font-mono">{keys.length}</p>
                            <p className="text-xs text-muted-foreground font-mono uppercase">
                                Total Keys
                            </p>
                        </div>
                    </div>
                </Card>
                <Card className="p-4 bg-card border border-border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-green-500/10 rounded-lg">
                            <Shield className="w-5 h-5 text-green-500" />
                        </div>
                        <div>
                            <p className="text-2xl font-bold font-mono">
                                {keys.filter((k) => !k.disabled && !isExpired(k.expiresAt)).length}
                            </p>
                            <p className="text-xs text-muted-foreground font-mono uppercase">
                                Active Keys
                            </p>
                        </div>
                    </div>
                </Card>
                <Card className="p-4 bg-card border border-border">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-yellow-500/10 rounded-lg">
                            <AlertTriangle className="w-5 h-5 text-yellow-500" />
                        </div>
                        <div>
                            <p className="text-2xl font-bold font-mono">
                                {keys.filter((k) => isExpired(k.expiresAt)).length}
                            </p>
                            <p className="text-xs text-muted-foreground font-mono uppercase">
                                Expired Keys
                            </p>
                        </div>
                    </div>
                </Card>
            </div>

            {/* Keys Table */}
            <Card className="border border-border overflow-hidden">
                {isLoading ? (
                    <div className="p-8 text-center text-muted-foreground font-mono">
                        Loading API keys...
                    </div>
                ) : keys.length === 0 ? (
                    <div className="p-8 text-center">
                        <Key className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
                        <p className="text-muted-foreground font-mono">No API keys yet</p>
                        <p className="text-sm text-muted-foreground font-mono mt-1">
                            Create your first API key to start using the S3-compatible API
                        </p>
                        <Button
                            onClick={() => setCreateDialogOpen(true)}
                            className="mt-4 font-mono uppercase text-xs"
                        >
                            <Plus className="w-4 h-4 mr-2" />
                            Create API Key
                        </Button>
                    </div>
                ) : (
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead className="font-mono uppercase text-xs">Name</TableHead>
                                <TableHead className="font-mono uppercase text-xs">Access Key ID</TableHead>
                                <TableHead className="font-mono uppercase text-xs">Scope</TableHead>
                                <TableHead className="font-mono uppercase text-xs">Created</TableHead>
                                <TableHead className="font-mono uppercase text-xs">Last Used</TableHead>
                                <TableHead className="font-mono uppercase text-xs">Status</TableHead>
                                <TableHead className="font-mono uppercase text-xs text-right">Actions</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {keys.map((key) => (
                                <TableRow key={key.id} className="group">
                                    <TableCell className="font-mono font-medium">{key.name}</TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-2">
                                            <code className="text-xs bg-muted px-2 py-1 rounded font-mono">
                                                {key.accessKeyId}
                                            </code>
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
                                                onClick={() => handleCopyKey(key.accessKeyId)}
                                            >
                                                {copiedId === key.accessKeyId ? (
                                                    <Check className="w-3 h-3 text-green-500" />
                                                ) : (
                                                    <Copy className="w-3 h-3" />
                                                )}
                                            </Button>
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        {key.bucketScope ? (
                                            <Badge variant="outline" className="font-mono text-xs">
                                                {key.bucketScope}
                                            </Badge>
                                        ) : (
                                            <span className="text-muted-foreground text-xs font-mono">All buckets</span>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-xs font-mono text-muted-foreground">
                                        {formatDate(key.createdAt)}
                                    </TableCell>
                                    <TableCell className="text-xs font-mono text-muted-foreground">
                                        {key.lastUsedAt ? formatDate(key.lastUsedAt) : "Never"}
                                    </TableCell>
                                    <TableCell>
                                        {key.disabled ? (
                                            <Badge variant="destructive" className="font-mono text-xs">
                                                Disabled
                                            </Badge>
                                        ) : isExpired(key.expiresAt) ? (
                                            <Badge variant="secondary" className="font-mono text-xs bg-yellow-500/10 text-yellow-600">
                                                Expired
                                            </Badge>
                                        ) : (
                                            <Badge variant="secondary" className="font-mono text-xs bg-green-500/10 text-green-600">
                                                Active
                                            </Badge>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                                            onClick={() => {
                                                setKeyToDelete(key);
                                                setDeleteDialogOpen(true);
                                            }}
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </Button>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </Card>

            {/* Create Dialog */}
            <CreateApiKeyDialog
                open={createDialogOpen}
                onOpenChange={setCreateDialogOpen}
                onSuccess={fetchKeys}
            />

            {/* API Docs Dialog */}
            <ApiDocsDialog open={docsDialogOpen} onOpenChange={setDocsDialogOpen} />

            {/* Delete Confirmation */}
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle className="font-mono uppercase">Delete API Key</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete the API key "{keyToDelete?.name}"? This action cannot
                            be undone and any applications using this key will lose access.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel className="font-mono uppercase text-xs">Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDeleteKey}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90 font-mono uppercase text-xs"
                        >
                            Delete
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
