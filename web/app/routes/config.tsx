import type { Route } from "./+types/config";
import Config from "@/components/config";

async function fetchConfig() {
  const response = await fetch("/api/config");
  if (!response.ok) {
    throw new Error(`Failed to fetch config: ${response.statusText}`);
  }
  const data = await response.json();
  return data.backends || [];
}

export async function clientLoader() {
  try {
    const config = await fetchConfig();
    return { config, error: null };
  } catch (error) {
    console.error("Failed to load configuration:", error);
    return { 
      config: [], 
      error: error instanceof Error ? error.message : "Failed to load configuration"
    };
  }
}

export function meta({}: Route.MetaArgs) {
 return [
   { title: "MCP Proxy Configuration" },
   { name: "description", content: "Proxy between MCP and HTTP endpoints with ease" },
 ];
}

export default function ConfigPage({ loaderData }: Route.ComponentProps) {
 return <Config initialData={loaderData} />;
}
