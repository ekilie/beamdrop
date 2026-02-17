import { createFileRoute } from "@tanstack/react-router";
import { ApiKeysPage } from "@/components/ApiKeysPage";

export const Route = createFileRoute("/_authenticated/api-keys")({
    component: ApiKeysPage,
});
