"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Card } from "@/components/ui/card"
import { Plus, Trash2, Type, Hash, ToggleLeft } from "lucide-react"
import type { Parameter } from "../types"
import { ConfirmationDialog } from "./confirmation-dialog"

interface ParametersSectionProps {
  parameters: Parameter[]
  onUpdate: (parameters: Parameter[]) => void
  type: string
  onMarkChanged: () => void
}

export function ParametersSection({ parameters, onUpdate, type, onMarkChanged }: ParametersSectionProps) {
  const [deleteIndex, setDeleteIndex] = useState<number | null>(null)

  const addParameter = () => {
    const newParam: Parameter = {
      data_type: "string",
      value_type: "dynamic",
      description: "",
      identifier: "",
      required: false,
    }
    onUpdate([...parameters, newParam])
    onMarkChanged()
  }

  const removeParameter = (index: number) => {
    onUpdate(parameters.filter((_, i) => i !== index))
    setDeleteIndex(null)
    onMarkChanged()
  }

  const updateParameter = (index: number, field: keyof Parameter, value: any) => {
    const updated = [...parameters]
    updated[index] = { ...updated[index], [field]: value }
    onUpdate(updated)
    onMarkChanged()
  }

  const getTypeIcon = (dataType: string) => {
    switch (dataType) {
      case "string":
        return <Type className="h-4 w-4" />
      case "number":
        return <Hash className="h-4 w-4" />
      case "boolean":
        return <ToggleLeft className="h-4 w-4" />
      default:
        return <Type className="h-4 w-4" />
    }
  }

  return (
    <>
      <div className="space-y-4">
        <div className="flex justify-between items-center">
          <h6 className="font-medium text-sm text-muted-foreground">{type} Parameters</h6>
          <Button variant="outline" size="sm" onClick={addParameter}>
            <Plus className="h-4 w-4 mr-2" />
            Add {type} Parameter
          </Button>
        </div>

        {parameters.length === 0 ? (
          <div className="text-center py-6 text-muted-foreground border border-dashed rounded-lg">
            <div className="flex justify-center mb-2">{getTypeIcon("string")}</div>
            <p className="text-sm">No {type.toLowerCase()} parameters</p>
            <p className="text-xs">Add parameters to configure this endpoint</p>
          </div>
        ) : (
          <div className="space-y-3">
            {parameters.map((param, index) => (
              <Card key={index} className="p-4 transition-all duration-200 hover:shadow-sm">
                <div className="flex justify-between items-start mb-4">
                  <div className="flex items-center gap-2">
                    {getTypeIcon(param.data_type)}
                    <span className="font-medium text-sm">{param.identifier || `Parameter ${index + 1}`}</span>
                    {param.required && <span className="text-destructive text-xs">*</span>}
                  </div>
                  <Button variant="ghost" size="sm" onClick={() => setDeleteIndex(index)}>
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <Label>Identifier *</Label>
                    <Input
                      value={param.identifier}
                      onChange={(e) => updateParameter(index, "identifier", e.target.value)}
                      placeholder="parameter_name"
                    />
                  </div>

                  <div>
                    <Label>Data Type</Label>
                    <Select
                      value={param.data_type}
                      onValueChange={(value) => updateParameter(index, "data_type", value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="string">String</SelectItem>
                        <SelectItem value="number">Number</SelectItem>
                        <SelectItem value="boolean">Boolean</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div>
                    <Label>Value Type</Label>
                    <Select
                      value={param.value_type}
                      onValueChange={(value) => updateParameter(index, "value_type", value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="dynamic">Dynamic</SelectItem>
                        <SelectItem value="constant">Constant</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex items-center space-x-2 pt-6">
                    <Switch
                      checked={param.required}
                      onCheckedChange={(checked) => updateParameter(index, "required", checked)}
                    />
                    <Label>Required</Label>
                  </div>
                </div>

                <div className="mt-4">
                  <Label>Description</Label>
                  <Textarea
                    value={param.description}
                    onChange={(e) => updateParameter(index, "description", e.target.value)}
                    placeholder="What is this parameter for?"
                    rows={2}
                  />
                </div>

                {param.value_type === "constant" && (
                  <div className="mt-4">
                    <Label>Default Value</Label>
                    <Input
                      value={param.value || ""}
                      onChange={(e) => updateParameter(index, "value", e.target.value)}
                      placeholder="Enter constant value"
                    />
                  </div>
                )}
              </Card>
            ))}
          </div>
        )}
      </div>

      <ConfirmationDialog
        open={deleteIndex !== null}
        onOpenChange={(open) => !open && setDeleteIndex(null)}
        title="Delete Parameter"
        description={`Are you sure you want to delete the parameter "${
          deleteIndex !== null ? parameters[deleteIndex]?.identifier || `Parameter ${deleteIndex + 1}` : ""
        }"? This action cannot be undone.`}
        onConfirm={() => deleteIndex !== null && removeParameter(deleteIndex)}
      />
    </>
  )
}
