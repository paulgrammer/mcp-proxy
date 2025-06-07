import type { Route } from "./+types/home";
import Config from "@/components/config";

export function meta({}: Route.MetaArgs) {
 return [
   { title: "MCP Proxy Configuration" },
   { name: "description", content: "Proxy between MCP and HTTP endpoints with ease" },
 ];
}

export default function ConfigPage() {
 return <Config />;
}
