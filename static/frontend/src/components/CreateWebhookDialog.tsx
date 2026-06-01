import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "@/hooks/use-toast";
import { Copy, Check, AlertTriangle } from "lucide-react";

const ALL_EVENT_TYPES = [
  { value: "beamdrop.object.*", label: "All Object Events", group: "Objects" },
  {
    value: "beamdrop.object.created",
    label: "Object Created",
    group: "Objects",
  },
  {
    value: "beamdrop.object.updated",
    label: "Object Updated",
    group: "Objects",
  },
  {
    value: "beamdrop.object.deleted",
    label: "Object Deleted",
    group: "Objects",
  },
  { value: "beamdrop.bucket.*", label: "All Bucket Events", group: "Buckets" },
  {
    value: "beamdrop.bucket.created",
    label: "Bucket Created",
    group: "Buckets",
  },
  {
    value: "beamdrop.bucket.deleted",
    label: "Bucket Deleted",
    group: "Buckets",
  },
  { value: "beamdrop.share.*", label: "All Share Events", group: "Shares" },
  { value: "beamdrop.share.created", label: "Share Created", group: "Shares" },
  { value: "beamdrop.share.deleted", label: "Share Deleted", group: "Shares" },
  {
    value: "beamdrop.presign.*",
    label: "All Presign Events",
    group: "Presigned URLs",
  },
  {
    value: "beamdrop.presign.created",
    label: "Presign Created",
    group: "Presigned URLs",
  },
  {
    value: "beamdrop.presign.deleted",
    label: "Presign Deleted",
    group: "Presigned URLs",
  },
];

interface CreateWebhookDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function CreateWebhookDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateWebhookDialogProps) {
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [bucketScope, setBucketScope] = useState("");
  const [selectedEvents, setSelectedEvents] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);
  const [copiedSecret, setCopiedSecret] = useState(false);

  const toggleEvent = (event: string) => {
    setSelectedEvents((prev) =>
      prev.includes(event) ? prev.filter((e) => e !== event) : [...prev, event],
    );
  };

  const handleCreate = async () => {
    if (!url) {
      toast({
        title: "Error",
        description: "URL is required",
        variant: "destructive",
      });
      return;
    }
    if (selectedEvents.length === 0) {
      toast({
        title: "Error",
        description: "Select at least one event type",
        variant: "destructive",
      });
      return;
    }

    setIsSubmitting(true);
    try {
      const res = await fetch("/api/v1/webhooks", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name || "webhook",
          url,
          event_types: selectedEvents,
          bucket_scope: bucketScope || undefined,
        }),
      });

      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error?.message || "Failed to create webhook");
      }

      const data = await res.json();
      setCreatedSecret(data.secret);
      toast({ title: "Created", description: "Webhook created successfully" });
    } catch (error: any) {
      toast({
        title: "Error",
        description: error.message,
        variant: "destructive",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    setName("");
    setUrl("");
    setBucketScope("");
    setSelectedEvents([]);
    setCreatedSecret(null);
    setCopiedSecret(false);
    onOpenChange(false);
    if (createdSecret) onSuccess();
  };

  const handleCopySecret = async () => {
    if (!createdSecret) return;
    await navigator.clipboard.writeText(createdSecret);
    setCopiedSecret(true);
    setTimeout(() => setCopiedSecret(false), 2000);
  };

  if (createdSecret) {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="font-mono uppercase">
              Webhook Created
            </DialogTitle>
            <DialogDescription>
              Save the signing secret below. It will not be shown again.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="flex items-center gap-2 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/30">
              <AlertTriangle className="w-4 h-4 text-yellow-500 shrink-0" />
              <p className="text-sm text-yellow-600 dark:text-yellow-400">
                Copy this secret now — you won't be able to see it again.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 text-xs bg-muted px-3 py-2 rounded font-mono break-all">
                {createdSecret}
              </code>
              <Button
                variant="ghost"
                size="icon"
                className="shrink-0"
                onClick={handleCopySecret}
              >
                {copiedSecret ? (
                  <Check className="w-4 h-4 text-green-500" />
                ) : (
                  <Copy className="w-4 h-4" />
                )}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={handleClose}
              className="font-mono uppercase text-xs"
            >
              Done
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="font-mono uppercase">
            Create Webhook
          </DialogTitle>
          <DialogDescription>
            Configure a new webhook endpoint to receive event notifications.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="wh-name" className="font-mono text-xs uppercase">
              Name
            </Label>
            <Input
              id="wh-name"
              placeholder="my-webhook"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="font-mono"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="wh-url" className="font-mono text-xs uppercase">
              Target URL *
            </Label>
            <Input
              id="wh-url"
              placeholder="https://example.com/webhook"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              className="font-mono"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="wh-scope" className="font-mono text-xs uppercase">
              Bucket Scope (optional)
            </Label>
            <Input
              id="wh-scope"
              placeholder="Leave empty for all buckets"
              value={bucketScope}
              onChange={(e) => setBucketScope(e.target.value)}
              className="font-mono"
            />
          </div>
          <div className="space-y-2">
            <Label className="font-mono text-xs uppercase">Event Types *</Label>
            <div className="space-y-3 max-h-60 overflow-y-auto p-1">
              {["Objects", "Buckets", "Shares", "Presigned URLs"].map(
                (group) => (
                  <div key={group}>
                    <p className="text-xs font-mono text-muted-foreground mb-1.5">
                      {group}
                    </p>
                    <div className="grid grid-cols-1 gap-1">
                      {ALL_EVENT_TYPES.filter((e) => e.group === group).map(
                        (evt) => (
                          <label
                            key={evt.value}
                            className="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-muted/50 cursor-pointer"
                          >
                            <Switch
                              checked={selectedEvents.includes(evt.value)}
                              onCheckedChange={() => toggleEvent(evt.value)}
                              className="scale-75"
                            />
                            <span className="text-sm font-mono">
                              {evt.label}
                            </span>
                            <code className="text-xs text-muted-foreground ml-auto hidden sm:block">
                              {evt.value}
                            </code>
                          </label>
                        ),
                      )}
                    </div>
                  </div>
                ),
              )}
            </div>
            {selectedEvents.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-2">
                {selectedEvents.map((e) => (
                  <Badge
                    key={e}
                    variant="secondary"
                    className="font-mono text-xs"
                  >
                    {e}
                  </Badge>
                ))}
              </div>
            )}
          </div>
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
            disabled={isSubmitting || !url || selectedEvents.length === 0}
            className="font-mono uppercase text-xs"
          >
            {isSubmitting ? "Creating..." : "Create Webhook"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
