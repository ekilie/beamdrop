import { useState, useEffect, useCallback } from "react";
import {
  Webhook as WebhookIcon,
  Plus,
  Trash2,
  RefreshCw,
  Eye,
  Play,
  RotateCcw,
  ChevronDown,
  ChevronUp,
  Power,
  PowerOff,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Clock,
  Copy,
  Check,
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
import { CreateWebhookDialog } from "./CreateWebhookDialog";

interface Webhook {
  id: number;
  name: string;
  targetUrl: string;
  enabled: boolean;
  eventTypes: string;
  bucketScope?: string;
  createdAt: string;
  updatedAt: string;
  lastDeliveryAt?: string;
  lastError?: string;
}

interface Delivery {
  id: number;
  webhookId: number;
  eventId: string;
  status: string;
  attemptCount: number;
  lastHttpStatus?: number;
  lastError?: string;
  lastDurationMs?: number;
  createdAt: string;
  deliveredAt?: string;
}

export function WebhooksPage() {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [webhookToDelete, setWebhookToDelete] = useState<Webhook | null>(null);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [deliveriesLoading, setDeliveriesLoading] = useState(false);
  const [rotatedSecret, setRotatedSecret] = useState<string | null>(null);
  const [copiedSecret, setCopiedSecret] = useState(false);

  const fetchWebhooks = useCallback(async () => {
    try {
      setIsLoading(true);
      const res = await fetch("/api/webhooks");
      if (!res.ok) throw new Error("Failed to fetch webhooks");
      const data = await res.json();
      setWebhooks(data.webhooks || []);
    } catch {
      toast({
        title: "Error",
        description: "Failed to fetch webhooks",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchWebhooks();
  }, [fetchWebhooks]);

  const fetchDeliveries = async (id: number) => {
    setDeliveriesLoading(true);
    try {
      const res = await fetch(`/api/webhooks/${id}/deliveries`);
      if (!res.ok) throw new Error("Failed to fetch deliveries");
      const data = await res.json();
      setDeliveries(data.deliveries || []);
    } catch {
      toast({
        title: "Error",
        description: "Failed to fetch deliveries",
        variant: "destructive",
      });
    } finally {
      setDeliveriesLoading(false);
    }
  };

  const toggleExpand = (id: number) => {
    if (expandedId === id) {
      setExpandedId(null);
      setDeliveries([]);
    } else {
      setExpandedId(id);
      fetchDeliveries(id);
    }
  };

  const handleToggleEnabled = async (wh: Webhook) => {
    try {
      const res = await fetch(`/api/webhooks/${wh.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled: !wh.enabled }),
      });
      if (!res.ok) throw new Error("Failed to update webhook");
      toast({
        title: wh.enabled ? "Disabled" : "Enabled",
        description: `Webhook "${wh.name}" ${wh.enabled ? "disabled" : "enabled"}`,
      });
      fetchWebhooks();
    } catch {
      toast({
        title: "Error",
        description: "Failed to update webhook",
        variant: "destructive",
      });
    }
  };

  const handleSendTest = async (wh: Webhook) => {
    try {
      const res = await fetch(`/api/webhooks/${wh.id}/test`, {
        method: "POST",
      });
      if (!res.ok) throw new Error("Failed to send test event");
      toast({
        title: "Test Sent",
        description: `Test event queued for "${wh.name}"`,
      });
      if (expandedId === wh.id) {
        setTimeout(() => fetchDeliveries(wh.id), 2000);
      }
    } catch {
      toast({
        title: "Error",
        description: "Failed to send test event",
        variant: "destructive",
      });
    }
  };

  const handleRotateSecret = async (wh: Webhook) => {
    try {
      const res = await fetch(`/api/webhooks/${wh.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rotate_secret: true }),
      });
      if (!res.ok) throw new Error("Failed to rotate secret");
      const data = await res.json();
      setRotatedSecret(data.secret);
      toast({
        title: "Secret Rotated",
        description: "New signing secret generated. Save it now.",
      });
    } catch {
      toast({
        title: "Error",
        description: "Failed to rotate secret",
        variant: "destructive",
      });
    }
  };

  const handleDelete = async () => {
    if (!webhookToDelete) return;
    try {
      const res = await fetch(`/api/webhooks/${webhookToDelete.id}`, {
        method: "DELETE",
      });
      if (res.ok || res.status === 204) {
        toast({
          title: "Deleted",
          description: `Webhook "${webhookToDelete.name}" deleted`,
        });
        if (expandedId === webhookToDelete.id) {
          setExpandedId(null);
          setDeliveries([]);
        }
        fetchWebhooks();
      } else {
        throw new Error("Failed to delete");
      }
    } catch {
      toast({
        title: "Error",
        description: "Failed to delete webhook",
        variant: "destructive",
      });
    } finally {
      setDeleteDialogOpen(false);
      setWebhookToDelete(null);
    }
  };

  const handleCopySecret = async () => {
    if (!rotatedSecret) return;
    await navigator.clipboard.writeText(rotatedSecret);
    setCopiedSecret(true);
    setTimeout(() => setCopiedSecret(false), 2000);
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

  const parseEventTypes = (eventTypes: string): string[] => {
    try {
      return JSON.parse(eventTypes);
    } catch {
      return [];
    }
  };

  const deliveryStatusIcon = (status: string) => {
    switch (status) {
      case "delivered":
        return <CheckCircle2 className="w-3.5 h-3.5 text-green-500" />;
      case "failed":
      case "dead_letter":
        return <XCircle className="w-3.5 h-3.5 text-red-500" />;
      case "pending":
      case "delivering":
        return <Clock className="w-3.5 h-3.5 text-yellow-500" />;
      default:
        return <Clock className="w-3.5 h-3.5 text-muted-foreground" />;
    }
  };

  const deliveryStatusColor = (status: string) => {
    switch (status) {
      case "delivered":
        return "bg-green-500/10 text-green-600";
      case "failed":
        return "bg-red-500/10 text-red-600";
      case "dead_letter":
        return "bg-red-500/10 text-red-600";
      case "pending":
        return "bg-yellow-500/10 text-yellow-600";
      case "delivering":
        return "bg-blue-500/10 text-blue-600";
      default:
        return "";
    }
  };

  return (
    <div className="p-6 space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold font-mono uppercase tracking-wide text-foreground flex items-center gap-2">
            <WebhookIcon className="w-6 h-6" />
            Webhooks
          </h1>
          <p className="text-sm text-muted-foreground font-mono mt-1">
            Receive real-time event notifications via HTTP callbacks
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={fetchWebhooks}
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
            Create Webhook
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card className="p-4 bg-card border border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary/10 rounded-lg">
              <WebhookIcon className="w-5 h-5 text-primary" />
            </div>
            <div>
              <p className="text-2xl font-bold font-mono">{webhooks.length}</p>
              <p className="text-xs text-muted-foreground font-mono uppercase">
                Total Webhooks
              </p>
            </div>
          </div>
        </Card>
        <Card className="p-4 bg-card border border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-green-500/10 rounded-lg">
              <Power className="w-5 h-5 text-green-500" />
            </div>
            <div>
              <p className="text-2xl font-bold font-mono">
                {webhooks.filter((w) => w.enabled).length}
              </p>
              <p className="text-xs text-muted-foreground font-mono uppercase">
                Active
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
                {webhooks.filter((w) => w.lastError).length}
              </p>
              <p className="text-xs text-muted-foreground font-mono uppercase">
                With Errors
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Webhooks List */}
      <Card className="border border-border overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-muted-foreground font-mono">
            Loading webhooks...
          </div>
        ) : webhooks.length === 0 ? (
          <div className="p-8 text-center">
            <WebhookIcon className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground font-mono">No webhooks yet</p>
            <p className="text-sm text-muted-foreground font-mono mt-1">
              Create a webhook to start receiving event notifications
            </p>
            <Button
              onClick={() => setCreateDialogOpen(true)}
              className="mt-4 font-mono uppercase text-xs"
            >
              <Plus className="w-4 h-4 mr-2" />
              Create Webhook
            </Button>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {webhooks.map((wh) => (
              <div key={wh.id}>
                {/* Webhook Row */}
                <div className="flex items-center gap-4 p-4 hover:bg-muted/30 transition-colors">
                  {/* Status indicator */}
                  <div
                    className={`w-2 h-2 rounded-full shrink-0 ${
                      wh.enabled ? "bg-green-500" : "bg-muted-foreground"
                    }`}
                  />

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-mono font-medium text-sm">
                        {wh.name}
                      </span>
                      <Badge
                        variant="secondary"
                        className={`font-mono text-xs ${
                          wh.enabled
                            ? "bg-green-500/10 text-green-600"
                            : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {wh.enabled ? "Active" : "Disabled"}
                      </Badge>
                      {wh.lastError && (
                        <Badge
                          variant="secondary"
                          className="font-mono text-xs bg-red-500/10 text-red-600"
                        >
                          Error
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs font-mono text-muted-foreground mt-1 truncate">
                      {wh.targetUrl}
                    </p>
                    <div className="flex flex-wrap gap-1 mt-1.5">
                      {parseEventTypes(wh.eventTypes).map((evt) => (
                        <Badge
                          key={evt}
                          variant="outline"
                          className="font-mono text-xs py-0"
                        >
                          {evt.replace("beamdrop.", "")}
                        </Badge>
                      ))}
                      {wh.bucketScope && (
                        <Badge
                          variant="outline"
                          className="font-mono text-xs py-0 border-blue-500/30 text-blue-600"
                        >
                          scope: {wh.bucketScope}
                        </Badge>
                      )}
                    </div>
                  </div>

                  {/* Meta */}
                  <div className="hidden lg:block text-right shrink-0">
                    <p className="text-xs font-mono text-muted-foreground">
                      Created {formatDate(wh.createdAt)}
                    </p>
                    {wh.lastDeliveryAt && (
                      <p className="text-xs font-mono text-muted-foreground mt-0.5">
                        Last delivery {formatDate(wh.lastDeliveryAt)}
                      </p>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-1 shrink-0">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => handleSendTest(wh)}
                      title="Send test event"
                    >
                      <Play className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => handleToggleEnabled(wh)}
                      title={wh.enabled ? "Disable" : "Enable"}
                    >
                      {wh.enabled ? (
                        <PowerOff className="w-4 h-4" />
                      ) : (
                        <Power className="w-4 h-4 text-green-500" />
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => handleRotateSecret(wh)}
                      title="Rotate signing secret"
                    >
                      <RotateCcw className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => toggleExpand(wh.id)}
                      title="View deliveries"
                    >
                      {expandedId === wh.id ? (
                        <ChevronUp className="w-4 h-4" />
                      ) : (
                        <Eye className="w-4 h-4" />
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => {
                        setWebhookToDelete(wh);
                        setDeleteDialogOpen(true);
                      }}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>

                {/* Expanded: Deliveries */}
                {expandedId === wh.id && (
                  <div className="border-t border-border bg-muted/20 px-4 py-3">
                    {wh.lastError && (
                      <div className="flex items-start gap-2 p-3 rounded-lg bg-red-500/5 border border-red-500/20 mb-3">
                        <AlertTriangle className="w-4 h-4 text-red-500 mt-0.5 shrink-0" />
                        <div>
                          <p className="text-xs font-mono font-medium text-red-600">
                            Last Error
                          </p>
                          <p className="text-xs font-mono text-red-500 mt-0.5">
                            {wh.lastError}
                          </p>
                        </div>
                      </div>
                    )}

                    <div className="flex items-center justify-between mb-2">
                      <p className="text-xs font-mono text-muted-foreground uppercase">
                        Recent Deliveries
                      </p>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => fetchDeliveries(wh.id)}
                        className="h-7 text-xs font-mono"
                      >
                        <RefreshCw className="w-3 h-3 mr-1" />
                        Refresh
                      </Button>
                    </div>

                    {deliveriesLoading ? (
                      <p className="text-xs text-muted-foreground font-mono py-4 text-center">
                        Loading...
                      </p>
                    ) : deliveries.length === 0 ? (
                      <p className="text-xs text-muted-foreground font-mono py-4 text-center">
                        No deliveries yet. Try sending a test event.
                      </p>
                    ) : (
                      <div className="rounded-lg border border-border overflow-hidden">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="font-mono uppercase text-xs">
                                Status
                              </TableHead>
                              <TableHead className="font-mono uppercase text-xs">
                                Event ID
                              </TableHead>
                              <TableHead className="font-mono uppercase text-xs hidden md:table-cell">
                                Attempts
                              </TableHead>
                              <TableHead className="font-mono uppercase text-xs hidden md:table-cell">
                                HTTP Status
                              </TableHead>
                              <TableHead className="font-mono uppercase text-xs hidden lg:table-cell">
                                Duration
                              </TableHead>
                              <TableHead className="font-mono uppercase text-xs">
                                Time
                              </TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {deliveries.slice(0, 20).map((d) => (
                              <TableRow key={d.id}>
                                <TableCell>
                                  <div className="flex items-center gap-1.5">
                                    {deliveryStatusIcon(d.status)}
                                    <Badge
                                      variant="secondary"
                                      className={`font-mono text-xs ${deliveryStatusColor(d.status)}`}
                                    >
                                      {d.status}
                                    </Badge>
                                  </div>
                                </TableCell>
                                <TableCell>
                                  <code className="text-xs font-mono text-muted-foreground">
                                    {d.eventId.slice(0, 8)}...
                                  </code>
                                </TableCell>
                                <TableCell className="font-mono text-xs hidden md:table-cell">
                                  {d.attemptCount}
                                </TableCell>
                                <TableCell className="font-mono text-xs hidden md:table-cell">
                                  {d.lastHttpStatus || "—"}
                                </TableCell>
                                <TableCell className="font-mono text-xs hidden lg:table-cell">
                                  {d.lastDurationMs
                                    ? `${d.lastDurationMs}ms`
                                    : "—"}
                                </TableCell>
                                <TableCell className="font-mono text-xs text-muted-foreground">
                                  {formatDate(d.createdAt)}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Create Dialog */}
      <CreateWebhookDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSuccess={fetchWebhooks}
      />

      {/* Delete Confirmation */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="font-mono uppercase">
              Delete Webhook
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the webhook "
              {webhookToDelete?.name}"? This will also delete all delivery
              history. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="font-mono uppercase text-xs">
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90 font-mono uppercase text-xs"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Rotated Secret Dialog */}
      <AlertDialog
        open={!!rotatedSecret}
        onOpenChange={() => {
          setRotatedSecret(null);
          setCopiedSecret(false);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="font-mono uppercase">
              New Signing Secret
            </AlertDialogTitle>
            <AlertDialogDescription>
              Your webhook's signing secret has been rotated. Save the new
              secret below — it will not be shown again.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="flex items-center gap-2 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/30 my-2">
            <AlertTriangle className="w-4 h-4 text-yellow-500 shrink-0" />
            <p className="text-sm text-yellow-600 dark:text-yellow-400">
              Update your webhook handler with this new secret.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <code className="flex-1 text-xs bg-muted px-3 py-2 rounded font-mono break-all">
              {rotatedSecret}
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
          <AlertDialogFooter>
            <AlertDialogAction
              onClick={() => {
                setRotatedSecret(null);
                setCopiedSecret(false);
              }}
              className="font-mono uppercase text-xs"
            >
              Done
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
