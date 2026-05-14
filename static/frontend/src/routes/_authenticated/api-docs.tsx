import { createFileRoute } from "@tanstack/react-router";
import ApiDocsPage from "@/pages/ApiDocs";

export const Route = createFileRoute("/_authenticated/api-docs")({
  component: ApiDocsPage,
});
