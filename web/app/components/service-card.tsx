"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  ChevronDown,
  ChevronRight,
  Globe,
  Settings,
  Trash2,
  Plus,
} from "lucide-react";
import type { ApiService } from "../../types";
import { EndpointCard } from "./endpoint-card";
import { HeadersSection } from "./headers-section";
import { ConfirmationDialog } from "./confirmation-dialog";

interface ServiceCardProps {
  service: ApiService;
  serviceIndex: number;
  onUpdate: (field: keyof ApiService, value: any) => void;
  onRemove: () => void;
  onAddEndpoint: () => void;
  onUpdateEndpoint: (endpointIndex: number, field: string, value: any) => void;
  onRemoveEndpoint: (endpointIndex: number) => void;
  onMarkChanged: () => void;
}

export function ServiceCard({
  service,
  serviceIndex,
  onUpdate,
  onRemove,
  onAddEndpoint,
  onUpdateEndpoint,
  onRemoveEndpoint,
  onMarkChanged,
}: Readonly<ServiceCardProps>) {
  console.log(service)
  const [isExpanded, setIsExpanded] = useState(serviceIndex === 0);
  const [activeSection, setActiveSection] = useState<"endpoints" | "headers">(
    "endpoints"
  );
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const getServiceName = () => {
    if (service.base_url) {
      try {
        return new URL(service.base_url).hostname;
      } catch {
        return service.base_url;
      }
    }
    return `Service ${serviceIndex + 1}`;
  };

  const handleUpdate = (field: keyof ApiService, value: any) => {
    onUpdate(field, value);
    onMarkChanged();
  };

  const handleUpdateEndpoint = (
    endpointIndex: number,
    field: string,
    value: any
  ) => {
    onUpdateEndpoint(endpointIndex, field, value);
    onMarkChanged();
  };

  const handleRemoveEndpoint = (endpointIndex: number) => {
    onRemoveEndpoint(endpointIndex);
    onMarkChanged();
  };

  const handleRemove = () => {
    setShowDeleteDialog(false);
    onRemove();
    onMarkChanged();
  };

  return (
    <>
      <Card className="overflow-hidden transition-all duration-200">
        <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
          <CollapsibleTrigger asChild>
            <CardHeader className="cursor-pointer hover:bg-muted/50 transition-colors">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  {isExpanded ? (
                    <ChevronDown className="h-4 w-4 text-muted-foreground transition-transform duration-200" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-muted-foreground transition-transform duration-200" />
                  )}
                  <Globe className="h-5 w-5" />
                  <div>
                    <h3 className="font-semibold text-lg">
                      {getServiceName()}
                    </h3>
                    <p className="text-sm text-muted-foreground">
                      {service.base_url || "No URL configured"}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">
                    {service.endpoints.length} endpoints
                  </Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      setShowDeleteDialog(true);
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardHeader>
          </CollapsibleTrigger>

          <CollapsibleContent className="transition-all duration-300 ease-in-out">
            <CardContent className="pt-0">
              {/* Base URL Configuration */}
              <div className="mb-6 p-4 bg-muted/30 rounded-lg">
                <Label
                  htmlFor={`base-url-${serviceIndex}`}
                  className="text-sm font-medium"
                >
                  Base URL *
                </Label>
                <Input
                  id={`base-url-${serviceIndex}`}
                  value={service.base_url}
                  onChange={(e) => handleUpdate("base_url", e.target.value)}
                  placeholder="https://api.example.com"
                  className="mt-1"
                />
              </div>

              {/* Section Navigation */}
              <div className="flex gap-2 mb-4">
                <Button
                  variant={
                    activeSection === "endpoints" ? "default" : "outline"
                  }
                  size="sm"
                  onClick={() => setActiveSection("endpoints")}
                  className="flex items-center gap-2 transition-colors duration-200"
                >
                  <Globe className="h-4 w-4" />
                  Endpoints ({service.endpoints.length})
                </Button>
                <Button
                  variant={activeSection === "headers" ? "default" : "outline"}
                  size="sm"
                  onClick={() => setActiveSection("headers")}
                  className="flex items-center gap-2 transition-colors duration-200"
                >
                  <Settings className="h-4 w-4" />
                  Headers ({service.default_headers?.length || 0})
                </Button>
              </div>

              {/* Content Sections */}
              {activeSection === "endpoints" && (
                <div className="space-y-4">
                  <div className="flex justify-between items-center">
                    <h4 className="font-medium">API Endpoints</h4>
                    <Button onClick={onAddEndpoint} size="sm">
                      <Plus className="h-4 w-4 mr-2" />
                      Add Endpoint
                    </Button>
                  </div>

                  {service.endpoints.length === 0 ? (
                    <div className="text-center py-8 text-muted-foreground border border-dashed rounded-lg">
                      <Globe className="h-12 w-12 mx-auto mb-3 opacity-50" />
                      <p>No endpoints configured</p>
                      <p className="text-sm">
                        Add your first endpoint to get started
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {service.endpoints.map((endpoint, endpointIndex) => (
                        <EndpointCard
                          key={endpoint.path}
                          endpoint={endpoint}
                          endpointIndex={endpointIndex}
                          onUpdate={(field, value) =>
                            handleUpdateEndpoint(
                              endpointIndex,
                              field as string,
                              value
                            )
                          }
                          onRemove={() => handleRemoveEndpoint(endpointIndex)}
                          onMarkChanged={onMarkChanged}
                        />
                      ))}
                    </div>
                  )}
                </div>
              )}

              {activeSection === "headers" && (
                <HeadersSection
                  headers={service.default_headers}
                  onUpdate={(headers) =>
                    handleUpdate("default_headers", headers)
                  }
                  onMarkChanged={onMarkChanged}
                />
              )}
            </CardContent>
          </CollapsibleContent>
        </Collapsible>
      </Card>

      <ConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete API Service"
        description={`Are you sure you want to delete "${getServiceName()}"? This will permanently remove the service and all its endpoints. This action cannot be undone.`}
        onConfirm={handleRemove}
      />
    </>
  );
}
