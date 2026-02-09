import { useState } from "react";
import {
    Key,
    Copy,
    Check,
    AlertTriangle,
    Eye,
    EyeOff,
} from "lucide-react";
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
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/hooks/use-toast";

interface CreateApiKeyDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSuccess: () => void;
}

interface CreatedKey {
    name: string;
    accessKeyId: string;
    secretKey: string;
}

export function CreateApiKeyDialog({
    open,
    onOpenChange,
    onSuccess,
}: CreateApiKeyDialogProps) {
    const [name, setName] = useState("");
    const [bucketScope, setBucketScope] = useState("");
    const [expiresIn, setExpiresIn] = useState<string>("never");
    const [isCreating, setIsCreating] = useState(false);
    const [createdKey, setCreatedKey] = useState<CreatedKey | null>(null);
    const [showSecret, setShowSecret] = useState(false);
    const [copiedField, setCopiedField] = useState<string | null>(null);

    const handleCreate = async () => {
        if (!name.trim()) {
            toast({
                title: "Error",
                description: "Please enter a name for the API key",
                variant: "destructive",
            });
            return;
        }

        setIsCreating(true);
        try {
            const body: Record<string, unknown> = { name: name.trim() };

            if (bucketScope.trim()) {
                body.bucketScope = bucketScope.trim();
            }

            if (expiresIn !== "never") {
                const seconds = parseInt(expiresIn, 10);
                body.expiresIn = seconds;
            }

            const response = await fetch("/api/v1/keys", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });

            if (response.ok) {
                const data = await response.json();
                setCreatedKey({
                    name: data.name,
                    accessKeyId: data.accessKeyId,
                    secretKey: data.secretKey,
                });
                toast({
                    title: "API Key Created",
                    description: "Make sure to copy your secret key now!",
                });
                onSuccess();
            } else {
                const error = await response.json();
                toast({
                    title: "Error",
                    description: error.error?.message || "Failed to create API key",
                    variant: "destructive",
                });
            }
        } catch {
            toast({
                title: "Error",
                description: "Failed to create API key",
                variant: "destructive",
            });
        } finally {
            setIsCreating(false);
        }
    };

    const handleCopy = async (text: string, field: string) => {
        try {
            await navigator.clipboard.writeText(text);
            setCopiedField(field);
            setTimeout(() => setCopiedField(null), 2000);
            toast({
                title: "Copied",
                description: `${field} copied to clipboard`,
            });
        } catch {
            toast({
                title: "Error",
                description: "Failed to copy to clipboard",
                variant: "destructive",
            });
        }
    };

    const handleClose = () => {
        setName("");
        setBucketScope("");
        setExpiresIn("never");
        setCreatedKey(null);
        setShowSecret(false);
        onOpenChange(false);
    };

    const expirationOptions = [
        { value: "never", label: "Never expires" },
        { value: "86400", label: "1 day" },
        { value: "604800", label: "7 days" },
        { value: "2592000", label: "30 days" },
        { value: "7776000", label: "90 days" },
        { value: "31536000", label: "1 year" },
    ];

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle className="font-mono uppercase flex items-center gap-2">
                        <Key className="w-5 h-5" />
                        {createdKey ? "API Key Created" : "Create API Key"}
                    </DialogTitle>
                    <DialogDescription className="font-mono text-xs">
                        {createdKey
                            ? "Save your credentials now. The secret key will not be shown again."
                            : "Create a new API key for S3-compatible storage access."}
                    </DialogDescription>
                </DialogHeader>

                {createdKey ? (
                    <div className="space-y-4">
                        {/* Warning Banner */}
                        <Card className="p-4 bg-yellow-500/10 border-yellow-500/20">
                            <div className="flex items-start gap-3">
                                <AlertTriangle className="w-5 h-5 text-yellow-500 flex-shrink-0 mt-0.5" />
                                <div>
                                    <p className="text-sm font-medium text-yellow-600 dark:text-yellow-400">
                                        Save your secret key now!
                                    </p>
                                    <p className="text-xs text-yellow-600/80 dark:text-yellow-400/80 mt-1">
                                        This is the only time you'll see this secret key. Copy it and store it securely.
                                    </p>
                                </div>
                            </div>
                        </Card>

                        {/* Key Details */}
                        <div className="space-y-4">
                            <div className="space-y-2">
                                <Label className="font-mono text-xs uppercase">Name</Label>
                                <div className="flex items-center justify-between p-3 bg-muted rounded-lg">
                                    <span className="font-mono text-sm">{createdKey.name}</span>
                                    <Badge variant="secondary" className="font-mono text-xs bg-green-500/10 text-green-600">
                                        Active
                                    </Badge>
                                </div>
                            </div>

                            <div className="space-y-2">
                                <Label className="font-mono text-xs uppercase">Access Key ID</Label>
                                <div className="flex items-center gap-2">
                                    <code className="flex-1 p-3 bg-muted rounded-lg font-mono text-sm break-all">
                                        {createdKey.accessKeyId}
                                    </code>
                                    <Button
                                        variant="outline"
                                        size="icon"
                                        onClick={() => handleCopy(createdKey.accessKeyId, "Access Key ID")}
                                    >
                                        {copiedField === "Access Key ID" ? (
                                            <Check className="w-4 h-4 text-green-500" />
                                        ) : (
                                            <Copy className="w-4 h-4" />
                                        )}
                                    </Button>
                                </div>
                            </div>

                            <div className="space-y-2">
                                <Label className="font-mono text-xs uppercase">Secret Key</Label>
                                <div className="flex items-center gap-2">
                                    <code className="flex-1 p-3 bg-muted rounded-lg font-mono text-sm break-all">
                                        {showSecret ? createdKey.secretKey : "•".repeat(40)}
                                    </code>
                                    <Button
                                        variant="outline"
                                        size="icon"
                                        onClick={() => setShowSecret(!showSecret)}
                                    >
                                        {showSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                    </Button>
                                    <Button
                                        variant="outline"
                                        size="icon"
                                        onClick={() => handleCopy(createdKey.secretKey, "Secret Key")}
                                    >
                                        {copiedField === "Secret Key" ? (
                                            <Check className="w-4 h-4 text-green-500" />
                                        ) : (
                                            <Copy className="w-4 h-4" />
                                        )}
                                    </Button>
                                </div>
                            </div>
                        </div>

                        <DialogFooter>
                            <Button
                                onClick={handleClose}
                                className="w-full font-mono uppercase text-xs"
                            >
                                Done
                            </Button>
                        </DialogFooter>
                    </div>
                ) : (
                    <div className="space-y-4">
                        <div className="space-y-2">
                            <Label htmlFor="name" className="font-mono text-xs uppercase">
                                Key Name *
                            </Label>
                            <Input
                                id="name"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                placeholder="e.g., CI Pipeline, Mobile App"
                                className="font-mono"
                            />
                            <p className="text-xs text-muted-foreground font-mono">
                                A descriptive name to identify this key
                            </p>
                        </div>

                        <div className="space-y-2">
                            <Label htmlFor="bucketScope" className="font-mono text-xs uppercase">
                                Bucket Scope (Optional)
                            </Label>
                            <Input
                                id="bucketScope"
                                value={bucketScope}
                                onChange={(e) => setBucketScope(e.target.value)}
                                placeholder="e.g., my-bucket"
                                className="font-mono"
                            />
                            <p className="text-xs text-muted-foreground font-mono">
                                Limit this key to a specific bucket. Leave empty for all buckets.
                            </p>
                        </div>

                        <div className="space-y-2">
                            <Label className="font-mono text-xs uppercase">Expiration</Label>
                            <Select value={expiresIn} onValueChange={setExpiresIn}>
                                <SelectTrigger className="font-mono">
                                    <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                    {expirationOptions.map((opt) => (
                                        <SelectItem key={opt.value} value={opt.value} className="font-mono">
                                            {opt.label}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>

                        <DialogFooter>
                            <Button
                                variant="outline"
                                onClick={handleClose}
                                className="font-mono uppercase text-xs"
                            >
                                Cancel
                            </Button>
                            <Button
                                onClick={handleCreate}
                                disabled={isCreating || !name.trim()}
                                className="font-mono uppercase text-xs"
                            >
                                {isCreating ? "Creating..." : "Create Key"}
                            </Button>
                        </DialogFooter>
                    </div>
                )}
            </DialogContent>
        </Dialog>
    );
}
