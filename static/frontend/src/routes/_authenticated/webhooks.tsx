import { createFileRoute } from "@tanstack/react-router";
import { WebhooksPage } from "@/components/WebhooksPage";

export const Route = createFileRoute("/_authenticated/webhooks")({
  component: WebhooksPage,
});
