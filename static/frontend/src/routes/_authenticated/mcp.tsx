import { createFileRoute } from "@tanstack/react-router";
import McpPage from "@/pages/McpPage";

export const Route = createFileRoute("/_authenticated/mcp")({
  component: McpPage,
});
