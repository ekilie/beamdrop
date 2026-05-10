import { createFileRoute } from "@tanstack/react-router";
import { UsageDashboard } from "@/components/UsageDashboard";

export const Route = createFileRoute("/_authenticated/usage")({
    component: UsageDashboard,
});
