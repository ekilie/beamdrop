import { createFileRoute } from "@tanstack/react-router";
import { SharesManagementPage } from "@/components/SharesManagementPage";

export const Route = createFileRoute("/_authenticated/shares")({
  component: SharesManagementPage,
});
