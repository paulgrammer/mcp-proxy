import { useState } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Textarea } from "@/components/ui/textarea"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { ChevronDown, ChevronRight, Trash2, Settings, FileText, Search, Route } from "lucide-react"
import type { Endpoint } from "../types"
import { ParametersSection } from "./parameters-section"
import { ConfirmationDialog } from "./confirmation-dialog"

interface EndpointCardProps {
  endpoint: Endpoint
  endpointIndex: number
  onUpdate: (field: keyof Endpoint, value: any) => void
  onRemove: () => void
  onMarkChanged: () => void
}

export function EndpointCard({ endpoint, endpointIndex, onUpdate, onRemove, onMarkChanged }: EndpointCardProps) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [activeTab, setActiveTab] = useState<"body" | "query" | "path">("body")
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)

  const getEndpointName = () => {
    return endpoint.name || `Endpoint ${endpointIndex + 1}`
  }

  const getParameterCount = (type: "body_params" | "query_parameters" | "path_parameters") => {
    return endpoint[type]?.length || 0
  }

  const handleUpdate = (field: keyof Endpoint, value: any) => {
    onUpdate(field, value)
    onMarkChanged()
  }

  const handleRemove = () => {
    setShowDeleteDialog(false)
    onRemove()
  }

  return (
    <>
      <Card className="border-l-4 border-l-primary transition-all duration-200">
        <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
          <CollapsibleTrigger asChild>
            <div className="p-4 cursor-pointer hover:bg-muted/30 transition-colors duration-200">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  {isExpanded ? (
                    <ChevronDown className="h-4 w-4 text-muted-foreground transition-transform duration-200" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-muted-foreground transition-transform duration-200" />
                  )}
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{endpoint.method}</Badge>
                    <span className="font-medium">{getEndpointName()}</span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant="secondary" className="text-xs">
                    {endpoint.capability}
                  </Badge>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      setShowDeleteDialog(true)
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
              <div className="mt-2 ml-7">
                <p className="text-sm text-muted-foreground">
                  {endpoint.path || "/path/not/set"} • {endpoint.description || "No description"}
                </p>
              </div>
            </div>
          </CollapsibleTrigger>

          <CollapsibleContent className="transition-all duration-300 ease-in-out">
            <CardContent className="pt-0 pb-4">
              {/* Basic Configuration */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                <div>
                  <Label htmlFor={`name-${endpointIndex}`}>Name *</Label>
                  <Input
                    id={`name-${endpointIndex}`}
                    value={endpoint.name}
                    onChange={(e) => handleUpdate("name", e.target.value)}
                    placeholder="create_user"
                  />
                </div>
                <div>
                  <Label htmlFor={`path-${endpointIndex}`}>Path *</Label>
                  <Input
                    id={`path-${endpointIndex}`}
                    value={endpoint.path}
                    onChange={(e) => handleUpdate("path", e.target.value)}
                    placeholder="/api/users"
                  />
                </div>
                <div>
                  <Label>Method</Label>
                  <Select value={endpoint.method} onValueChange={(value) => handleUpdate("method", value)}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="GET">GET</SelectItem>
                      <SelectItem value="POST">POST</SelectItem>
                      <SelectItem value="PUT">PUT</SelectItem>
                      <SelectItem value="DELETE">DELETE</SelectItem>
                      <SelectItem value="PATCH">PATCH</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label>Capability</Label>
                  <Select value={endpoint.capability} onValueChange={(value) => handleUpdate("capability", value)}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="tool">Tool</SelectItem>
                      <SelectItem value="prompt">Prompt</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="mb-4">
                <Label htmlFor={`description-${endpointIndex}`}>Description</Label>
                <Textarea
                  id={`description-${endpointIndex}`}
                  value={endpoint.description}
                  onChange={(e) => handleUpdate("description", e.target.value)}
                  placeholder="What does this endpoint do?"
                  rows={2}
                />
              </div>

              {/* Advanced Settings */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6 p-4 bg-muted/30 rounded-lg">
                <div>
                  <Label>Mode</Label>
                  <Select value={endpoint.mode} onValueChange={(value) => handleUpdate("mode", value)}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="client">Client</SelectItem>
                      <SelectItem value="server">Server</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor={`timeout-${endpointIndex}`}>Timeout</Label>
                  <Input
                    id={`timeout-${endpointIndex}`}
                    value={endpoint.response_timeout}
                    onChange={(e) => handleUpdate("response_timeout", e.target.value)}
                    placeholder="30s"
                  />
                </div>
                <div className="flex items-center space-x-2 pt-6">
                  <Switch
                    checked={endpoint.wait_response}
                    onCheckedChange={(checked) => handleUpdate("wait_response", checked)}
                  />
                  <Label>Wait for response</Label>
                </div>
              </div>

              {/* Parameters Section */}
              <div className="space-y-4">
                <h5 className="font-medium flex items-center gap-2">
                  <Settings className="h-4 w-4" />
                  Parameters
                </h5>

                <div className="flex gap-2">
                  <Button
                    variant={activeTab === "body" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setActiveTab("body")}
                    className="flex items-center gap-2 transition-colors duration-200"
                  >
                    <FileText className="h-4 w-4" />
                    Body ({getParameterCount("body_params")})
                  </Button>
                  <Button
                    variant={activeTab === "query" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setActiveTab("query")}
                    className="flex items-center gap-2 transition-colors duration-200"
                  >
                    <Search className="h-4 w-4" />
                    Query ({getParameterCount("query_parameters")})
                  </Button>
                  <Button
                    variant={activeTab === "path" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setActiveTab("path")}
                    className="flex items-center gap-2 transition-colors duration-200"
                  >
                    <Route className="h-4 w-4" />
                    Path ({getParameterCount("path_parameters")})
                  </Button>
                </div>

                <div className="border rounded-lg p-4">
                  {activeTab === "body" && (
                    <ParametersSection
                      parameters={endpoint.body_params || []}
                      onUpdate={(params) => handleUpdate("body_params", params)}
                      type="Body"
                      onMarkChanged={onMarkChanged}
                    />
                  )}
                  {activeTab === "query" && (
                    <ParametersSection
                      parameters={endpoint.query_parameters || []}
                      onUpdate={(params) => handleUpdate("query_parameters", params)}
                      type="Query"
                      onMarkChanged={onMarkChanged}
                    />
                  )}
                  {activeTab === "path" && (
                    <ParametersSection
                      parameters={endpoint.path_parameters || []}
                      onUpdate={(params) => handleUpdate("path_parameters", params)}
                      type="Path"
                      onMarkChanged={onMarkChanged}
                    />
                  )}
                </div>
              </div>
            </CardContent>
          </CollapsibleContent>
        </Collapsible>
      </Card>

      <ConfirmationDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete Endpoint"
        description={`Are you sure you want to delete the endpoint "${getEndpointName()}"? This action cannot be undone.`}
        onConfirm={handleRemove}
      />
    </>
  )
}
