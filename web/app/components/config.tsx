"use client";

import type React from "react";

import { useState, useEffect } from "react";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import {
  Plus,
  Download,
  Upload,
  Zap,
  Globe,
  Save,
  Loader2,
} from "lucide-react";
import type { ApiService } from "../../types";
import { ServiceCard } from "./service-card";
import { ConfirmationDialog } from "./confirmation-dialog";
import { toast } from "sonner";
import { useIsLoading } from "@/lib/hooks";

// API functions
const API_BASE = "/api";

async function saveConfig(services: ApiService[]) {
  const configData = {
    mcp: {
      server_name: "MCP HTTP Proxy",
      version: "1.0.0",
    },
    backends: services,
  };

  const response = await fetch(`${API_BASE}/config`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(configData),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(
      errorText || `Failed to save config: ${response.statusText}`
    );
  }

  return response.json();
}

interface ConfigProps {
  initialData?: {
    config: ApiService[];
    error: string | null;
  };
}

export default function Config({ initialData }: Readonly<ConfigProps>) {
  const [services, setServices] = useState<ApiService[]>(initialData?.config || []);
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [savedServices, setSavedServices] = useState<ApiService[]>(initialData?.config || []);
  const [saving, setSaving] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [deleteServiceIndex, setDeleteServiceIndex] = useState<number | null>(
    null
  );
  const isNavigating = useIsLoading();

  // Show error toast if there was an error loading
  useEffect(() => {
    if (initialData?.error) {
      toast.error(initialData.error);
    }
  }, [initialData?.error]);

  // Check for unsaved changes
  useEffect(() => {
    const hasChanges =
      JSON.stringify(services) !== JSON.stringify(savedServices);
    setHasUnsavedChanges(hasChanges);
  }, [services, savedServices]);

  const markChanged = () => {
    // This function is called whenever any field is updated
    // The useEffect above will automatically detect changes
  };

  const saveChanges = async () => {
    try {
      setSaving(true);
      await saveConfig(services);
      setSavedServices([...services]);
      setHasUnsavedChanges(false);
      toast.success("Configuration saved successfully!");
    } catch (error: any) {
      console.error("Failed to save configuration:", error);
      toast.error(`Failed to save configuration: ${error.message}`);
    } finally {
      setSaving(false);
    }
  };

  const addService = () => {
    const newService: ApiService = {
      base_url: "",
      default_headers: [],
      endpoints: [],
    };
    setServices([...services, newService]);
    markChanged();
  };

  const removeService = (serviceIndex: number) => {
    setServices(services.filter((_, index) => index !== serviceIndex));
    setDeleteServiceIndex(null);
    setShowDeleteDialog(false);
    markChanged();
  };

  const updateService = (
    serviceIndex: number,
    field: keyof ApiService,
    value: any
  ) => {
    const updatedServices = [...services];
    updatedServices[serviceIndex] = {
      ...updatedServices[serviceIndex],
      [field]: value,
    };
    setServices(updatedServices);
  };

  const addEndpoint = (serviceIndex: number) => {
    const newEndpoint = {
      capability: "tool" as const,
      mode: "client" as const,
      name: "",
      path: "",
      method: "GET" as const,
      description: "",
      wait_response: true,
      response_timeout: "30s",
    };
    const updatedServices = [...services];
    updatedServices[serviceIndex].endpoints.push(newEndpoint);
    setServices(updatedServices);
    markChanged();
  };

  const updateEndpoint = (
    serviceIndex: number,
    endpointIndex: number,
    field: string,
    value: any
  ) => {
    const updatedServices = [...services];
    updatedServices[serviceIndex].endpoints[endpointIndex] = {
      ...updatedServices[serviceIndex].endpoints[endpointIndex],
      [field]: value,
    };
    setServices(updatedServices);
  };

  const removeEndpoint = (serviceIndex: number, endpointIndex: number) => {
    const updatedServices = [...services];
    updatedServices[serviceIndex].endpoints.splice(endpointIndex, 1);
    setServices(updatedServices);
  };

  const exportConfig = () => {
    const config = services.map((service) => ({
      base_url: service.base_url,
      default_headers: service.default_headers,
      endpoints: service.endpoints,
    }));

    const blob = new Blob([JSON.stringify(config, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "api-config.json";
    a.click();
    URL.revokeObjectURL(url);
  };

  const importConfig = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        try {
          const config = JSON.parse(e.target?.result as string);
          setServices(config);
          markChanged();
        } catch (error) {
          console.error("Error importing configuration:", error);
          toast.error("Invalid JSON file");
        }
      };
      reader.readAsText(file);
    }
  };

  const getTotalEndpoints = () => {
    return services.reduce(
      (total, service) => total + service.endpoints.length,
      0
    );
  };

  const handleDeleteService = (serviceIndex: number) => {
    setDeleteServiceIndex(serviceIndex);
    setShowDeleteDialog(true);
  };

  const confirmDeleteService = () => {
    if (deleteServiceIndex !== null) {
      removeService(deleteServiceIndex);
    }
  };

  const getServiceName = (service: ApiService, index: number) => {
    if (service.base_url) {
      try {
        return new URL(service.base_url).hostname;
      } catch {
        return service.base_url;
      }
    }
    return `Service ${index + 1}`;
  };

  return (
    <>
      <div className="container mx-auto p-6 max-w-6xl">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-primary rounded-lg">
              <Zap className="h-6 w-6 text-primary-foreground" />
            </div>
            <div>
              <h1 className="text-3xl font-bold">MCP Proxy Configuration</h1>
              <p className="text-muted-foreground">
                Proxy between MCP and HTTP endpoints with ease
              </p>
            </div>
          </div>

          {/* Stats */}
          <div className="flex gap-4 mb-6">
            <Card className="p-4 flex-1">
              <div className="flex items-center gap-2">
                <Globe className="h-5 w-5 text-primary" />
                <span className="font-semibold text-2xl">
                  {services.length}
                </span>
              </div>
              <p className="text-sm text-muted-foreground">API Services</p>
            </Card>
            <Card className="p-4 flex-1">
              <div className="flex items-center gap-2">
                <Zap className="h-5 w-5 text-primary" />
                <span className="font-semibold text-2xl">
                  {getTotalEndpoints()}
                </span>
              </div>
              <p className="text-sm text-muted-foreground">Total Endpoints</p>
            </Card>
          </div>

          {/* Actions */}
          <div className="flex gap-3">
            <Button
              onClick={addService}
              disabled={isNavigating}
              className="transition-all duration-200"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add API Service
            </Button>
            <Button
              variant="outline"
              onClick={exportConfig}
              disabled={isNavigating}
              className="transition-all duration-200"
            >
              <Download className="h-4 w-4 mr-2" />
              Export Config
            </Button>
            <Button
              variant="outline"
              asChild
              disabled={isNavigating}
              className="transition-all duration-200"
            >
              <label className="cursor-pointer">
                <Upload className="h-4 w-4 mr-2" />
                Import Config
                <input
                  type="file"
                  accept=".json"
                  onChange={importConfig}
                  className="hidden"
                  disabled={isNavigating}
                />
              </label>
            </Button>
            {hasUnsavedChanges && (
              <Button
                onClick={saveChanges}
                disabled={saving || isNavigating}
                className="bg-green-600 hover:bg-green-700 transition-all duration-200"
              >
                {saving ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <Save className="h-4 w-4 mr-2" />
                )}
                {saving ? "Saving..." : "Save Changes"}
              </Button>
            )}
          </div>
        </div>

        {/* Services */}
        <div className="space-y-4">
          {services.map((service, serviceIndex) => (
            <ServiceCard
              service={service}
              key={`${service.base_url}-${serviceIndex}`}
              serviceIndex={serviceIndex}
              onUpdate={(field, value) =>
                updateService(serviceIndex, field as keyof ApiService, value)
              }
              onRemove={() => handleDeleteService(serviceIndex)}
              onAddEndpoint={() => addEndpoint(serviceIndex)}
              onUpdateEndpoint={(endpointIndex, field, value) =>
                updateEndpoint(serviceIndex, endpointIndex, field, value)
              }
              onRemoveEndpoint={(endpointIndex) =>
                removeEndpoint(serviceIndex, endpointIndex)
              }
              onMarkChanged={markChanged}
            />
          ))}
        </div>

        {/* Empty State */}
        {services.length === 0 && (
          <Card className="p-12 text-center">
            <div className="max-w-md mx-auto">
              <div className="p-3 bg-primary/10 rounded-full w-fit mx-auto mb-4">
                <Globe className="h-8 w-8 text-primary" />
              </div>
              <h3 className="text-xl font-semibold mb-2">
                No API Services Yet
              </h3>
              <p className="text-muted-foreground mb-6">
                Get started by adding your first API service. You can configure
                endpoints, headers, and parameters all in one place.
              </p>
              <Button onClick={addService} size="lg">
                <Plus className="h-5 w-5 mr-2" />
                Create Your First API Service
              </Button>
            </div>
          </Card>
        )}
      </div>

      <ConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete API Service"
        description={`Are you sure you want to delete "${
          deleteServiceIndex !== null
            ? getServiceName(services[deleteServiceIndex], deleteServiceIndex)
            : ""
        }"? This will permanently remove the service and all its endpoints. This action cannot be undone.`}
        onConfirm={confirmDeleteService}
      />
    </>
  );
}
