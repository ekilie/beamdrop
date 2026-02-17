import { createFileRoute } from "@tanstack/react-router";
import ShareAccess from "@/pages/ShareAccess";

export const Route = createFileRoute("/share/$token")({
  component: ShareAccess,
});
